// Package approvalsock carries one tool approval between the hook a CLI agent
// spawns and the backend process that asks the daemon about it, over a Unix
// domain socket on the same machine.
//
// The backend calls Serve with the asker it gets from the daemon, and hands the
// socket path to the hook through EnvApprovalSocket. The hook calls Ask with that
// path. Each connection carries one request line and one answer line, both JSON.
// No session name and no token travels on the socket.
package approvalsock

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"path/filepath"
	"sync"
	"time"

	contracts "github.com/Herrscherd/herrscher-contracts"
)

// EnvApprovalSocket names the environment variable that carries the socket path
// from the backend to the hook it spawns.
const EnvApprovalSocket = "HERRSCHER_APPROVAL_SOCKET"

const (
	maxMessageBytes = 64 << 10
	messageDeadline = 5 * time.Second
)

var (
	// ErrNoAsker is returned by Serve when it is given no asker to answer with.
	ErrNoAsker = errors.New("no approval asker to answer with")
	// ErrInsecureDirectory is returned when the directory of the socket is not a
	// directory owned by the current user with mode 0700.
	ErrInsecureDirectory = errors.New("approval socket directory is not private to this user")
	// ErrRequestTooLarge is returned when a message exceeds 64 KiB.
	ErrRequestTooLarge = errors.New("approval message exceeds its size bound")
	// ErrAnswerMismatch is returned by Ask when the answer carries another ID than
	// the request it answers.
	ErrAnswerMismatch = errors.New("approval answer does not match its request")
)

// Serve listens on the Unix socket at path and answers each approval request with
// asker, until ctx ends. The directory of path must exist and be owned by the
// current user with mode 0700, or Serve returns an error matching
// ErrInsecureDirectory. A nil asker returns ErrNoAsker. Serve does not remove a
// file already at path: a socket left by an earlier run must be removed first.
//
// Each connection carries one request line of at most 64 KiB, read within five
// seconds, and receives one answer line. A malformed, oversized or silent request
// closes its connection without calling asker, and the hook reads that as a
// denial. Serve returns nil once ctx ends, after the socket file is removed and
// every connection in progress has returned. asker must therefore return when ctx
// ends.
func Serve(ctx context.Context, path string, asker contracts.ApprovalAsker) error {
	if asker == nil {
		return ErrNoAsker
	}
	if err := checkPrivateDirectory(filepath.Dir(path)); err != nil {
		return err
	}
	var listenConfig net.ListenConfig
	listener, err := listenConfig.Listen(ctx, "unix", path)
	if err != nil {
		return fmt.Errorf("listen on approval socket: %w", err)
	}
	return serveListener(ctx, listener, asker)
}

// Ask sends req to the approval socket at path and waits for its answer. The wait
// lasts as long as ctx allows, so the caller sets the deadline. Ask returns an
// error when the socket cannot be reached, when ctx ends first, or when the answer
// is not one JSON line. An answer whose ID is not req.ID matches ErrAnswerMismatch,
// and an answer above 64 KiB matches ErrRequestTooLarge. Every error means the
// daemon was not reached, and the caller must treat it as a denial.
func Ask(ctx context.Context, path string, req contracts.ApprovalRequest) (contracts.ApprovalAnswer, error) {
	var dialer net.Dialer
	connection, err := dialer.DialContext(ctx, "unix", path)
	if err != nil {
		return contracts.ApprovalAnswer{}, fmt.Errorf("dial approval socket: %w", err)
	}
	answer, err := exchange(ctx, connection, req)
	err = errors.Join(err, connection.Close())
	if err != nil {
		return contracts.ApprovalAnswer{}, err
	}
	return answer, nil
}

func serveListener(ctx context.Context, listener net.Listener, asker contracts.ApprovalAsker) error {
	closed := make(chan error, 1)
	stopOnCancel := context.AfterFunc(ctx, func() { closed <- listener.Close() })
	var connections sync.WaitGroup
	acceptErr := acceptConnections(ctx, listener, asker, &connections)
	var closeErr error
	if stopOnCancel() {
		closeErr = listener.Close()
	} else {
		closeErr = <-closed
	}
	connections.Wait()
	if ctx.Err() != nil {
		return closeErr
	}
	return errors.Join(acceptErr, closeErr)
}

func acceptConnections(ctx context.Context, listener net.Listener, asker contracts.ApprovalAsker, connections *sync.WaitGroup) error {
	for {
		connection, err := listener.Accept()
		if err != nil {
			return fmt.Errorf("accept on approval socket: %w", err)
		}
		connections.Go(func() { answerConnection(ctx, connection, asker) })
	}
}

func closeConnection(connection net.Conn) {
	if err := connection.Close(); err != nil {
		return
	}
}

func answerConnection(ctx context.Context, connection net.Conn, asker contracts.ApprovalAsker) {
	defer closeConnection(connection)
	request, err := readRequest(connection)
	if err != nil {
		return
	}
	if err := writeMessage(connection, asker(ctx, request)); err != nil {
		return
	}
}

func exchange(ctx context.Context, connection net.Conn, req contracts.ApprovalRequest) (contracts.ApprovalAnswer, error) {
	stopOnCancel := context.AfterFunc(ctx, func() { closeConnection(connection) })
	defer stopOnCancel()
	if err := writeMessage(connection, req); err != nil {
		return contracts.ApprovalAnswer{}, interrupted(ctx, err)
	}
	line, err := readLine(connection)
	if err != nil {
		return contracts.ApprovalAnswer{}, interrupted(ctx, err)
	}
	var answer contracts.ApprovalAnswer
	if err := json.Unmarshal(line, &answer); err != nil {
		return contracts.ApprovalAnswer{}, fmt.Errorf("decode approval answer: %w", err)
	}
	if answer.ID != req.ID {
		return contracts.ApprovalAnswer{}, fmt.Errorf("%w: got %q for %q", ErrAnswerMismatch, answer.ID, req.ID)
	}
	return answer, nil
}

func interrupted(ctx context.Context, err error) error {
	if ctxErr := ctx.Err(); ctxErr != nil {
		return fmt.Errorf("wait for approval answer: %w", ctxErr)
	}
	return err
}

func readRequest(connection net.Conn) (contracts.ApprovalRequest, error) {
	var request contracts.ApprovalRequest
	if err := connection.SetReadDeadline(time.Now().Add(messageDeadline)); err != nil {
		return request, fmt.Errorf("set approval read deadline: %w", err)
	}
	line, err := readLine(connection)
	if err != nil {
		return request, err
	}
	if err := json.Unmarshal(line, &request); err != nil {
		return request, fmt.Errorf("decode approval request: %w", err)
	}
	return request, nil
}

func readLine(reader io.Reader) ([]byte, error) {
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(nil, maxMessageBytes)
	if scanner.Scan() {
		return scanner.Bytes(), nil
	}
	err := scanner.Err()
	if errors.Is(err, bufio.ErrTooLong) {
		return nil, ErrRequestTooLarge
	}
	if err == nil {
		err = io.ErrUnexpectedEOF
	}
	return nil, fmt.Errorf("read approval message: %w", err)
}

func writeMessage(connection net.Conn, value any) error {
	encoded, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("encode approval message: %w", err)
	}
	if err := connection.SetWriteDeadline(time.Now().Add(messageDeadline)); err != nil {
		return fmt.Errorf("set approval write deadline: %w", err)
	}
	if _, err := connection.Write(append(encoded, '\n')); err != nil {
		return fmt.Errorf("write approval message: %w", err)
	}
	return nil
}
