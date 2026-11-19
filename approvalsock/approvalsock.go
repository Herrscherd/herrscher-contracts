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

// MaxConcurrentConnections bounds how many connections Serve answers at once. A
// connection beyond the bound waits in the listen backlog until one of them ends.
const MaxConcurrentConnections = 16

const (
	maxMessageBytes = 64 << 10
	messageDeadline = 5 * time.Second
	answerDeadline  = 10 * time.Minute
	probeTimeout    = time.Second
	socketMode      = 0o600
	minAcceptDelay  = 5 * time.Millisecond
	maxAcceptDelay  = time.Second
	backoffFactor   = 2
)

var (
	// ErrNoAsker is returned by Serve when it is given no asker to answer with.
	ErrNoAsker = errors.New("no approval asker to answer with")
	// ErrNoErrorHandler is returned by Serve when it is given no WithErrorHandler,
	// so that no connection failure could go unseen.
	ErrNoErrorHandler = errors.New("no error handler to report connection errors to")
	// ErrInsecureDirectory is returned when the directory of the socket is not a
	// directory owned by the current user with mode 0700.
	ErrInsecureDirectory = errors.New("approval socket directory is not private to this user")
	// ErrPathInUse is returned by Serve when the socket path holds something other
	// than a stale socket, or a socket that still answers.
	ErrPathInUse = errors.New("approval socket path is held by something other than a stale socket")
	// ErrDirectoryChanged is returned by Serve when the directory of the socket was
	// replaced while the socket was being created.
	ErrDirectoryChanged = errors.New("approval socket directory changed while the socket was created")
	// ErrRequestTooLarge is returned when a message exceeds 64 KiB.
	ErrRequestTooLarge = errors.New("approval message exceeds its size bound")
	// ErrAnswerMismatch is returned by Ask when the answer carries another ID than
	// the request it answers.
	ErrAnswerMismatch = errors.New("approval answer does not match its request")
	// ErrForeignPeer is returned by Ask when the process listening on the socket does
	// not run as the current user.
	ErrForeignPeer = errors.New("approval socket is answered by another user")
)

// Op names the step of a connection that failed.
type Op string

const (
	// OpAccept is a failed accept on the socket.
	OpAccept Op = "accept"
	// OpRead is a request that could not be read or decoded.
	OpRead Op = "read"
	// OpAnswer is an answer that could not be encoded or written.
	OpAnswer Op = "answer"
	// OpClose is a connection that failed to close.
	OpClose Op = "close"
)

// ConnectionError reports a failure that Serve met on one connection, or on its
// accept loop. Err holds the cause, so errors.Is and errors.As reach it.
type ConnectionError struct {
	Op  Op
	Err error
}

// Error renders the operation and its cause.
func (e *ConnectionError) Error() string {
	return fmt.Sprintf("approval connection %s: %v", e.Op, e.Err)
}

// Unwrap returns the cause of the failure.
func (e *ConnectionError) Unwrap() error {
	return e.Err
}

// Option tunes Serve.
type Option func(*serveSettings)

type serveSettings struct {
	report func(error)
}

// WithErrorHandler sends every failure that ends a connection, and every failed
// accept, to report as a *ConnectionError. Serve calls report from its connection
// goroutines, so report must be safe for concurrent use. Serve refuses to start
// without it.
func WithErrorHandler(report func(error)) Option {
	return func(settings *serveSettings) {
		settings.report = report
	}
}

// Serve listens on the Unix socket at path and answers each approval request with
// asker, until ctx ends. The directory of path must exist and be owned by the
// current user with mode 0700, or Serve returns an error matching
// ErrInsecureDirectory. A nil asker returns ErrNoAsker, and no WithErrorHandler
// returns ErrNoErrorHandler. A socket left by a run that was killed is removed and
// replaced. Any other file at path, a live socket included, returns an error
// matching ErrPathInUse, and is left alone.
//
// Each connection carries one request line of at most 64 KiB, read within five
// seconds, and receives one answer line. At most MaxConcurrentConnections are
// answered at once. A malformed, oversized or silent request closes its connection
// without calling asker, and the hook reads that as a denial. The context of asker
// ends when ctx ends, after ten minutes, or when the hook closes its connection.
// Serve returns nil once ctx ends, after the socket file is removed and every
// connection in progress has returned. asker must therefore return when its
// context ends.
func Serve(ctx context.Context, path string, asker contracts.ApprovalAsker, opts ...Option) error {
	if asker == nil {
		return ErrNoAsker
	}
	var settings serveSettings
	for _, apply := range opts {
		apply(&settings)
	}
	if settings.report == nil {
		return ErrNoErrorHandler
	}
	directory, err := checkPrivateDirectory(filepath.Dir(path))
	if err != nil {
		return err
	}
	err = clearStaleSocket(ctx, path)
	if err != nil {
		return err
	}
	var listenConfig net.ListenConfig
	listener, err := listenConfig.Listen(ctx, "unix", path)
	if err != nil {
		return fmt.Errorf("listen on approval socket: %w", err)
	}
	err = secureSocket(listener, path, directory)
	if err != nil {
		return err
	}
	return serveListener(ctx, listener, asker, settings.report)
}

// Ask sends req to the approval socket at path and waits for its answer. The wait
// lasts as long as ctx allows, so the caller sets the deadline. Ask returns an
// error when the directory of path is not private to the current user, when the
// socket cannot be reached, when the process answering it runs as another user, or
// when ctx ends first. An answer that is not one JSON line, or whose decision is
// unknown, matches ErrInvalidDecision. An answer whose ID is not req.ID matches
// ErrAnswerMismatch, and an answer above 64 KiB matches ErrRequestTooLarge. Every
// error means the daemon was not reached, and the caller must treat it as a denial.
func Ask(ctx context.Context, path string, req contracts.ApprovalRequest) (contracts.ApprovalAnswer, error) {
	if _, err := checkPrivateDirectory(filepath.Dir(path)); err != nil {
		return contracts.ApprovalAnswer{}, err
	}
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

func serveListener(ctx context.Context, listener net.Listener, asker contracts.ApprovalAsker, report func(error)) error {
	closed := make(chan error, 1)
	stopOnCancel := context.AfterFunc(ctx, func() { closed <- listener.Close() })
	var connections sync.WaitGroup
	slots := make(chan struct{}, MaxConcurrentConnections)
	acceptErr := acceptConnections(ctx, listener, asker, report, slots, &connections)
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

func acceptConnections(ctx context.Context, listener net.Listener, asker contracts.ApprovalAsker, report func(error), slots chan struct{}, connections *sync.WaitGroup) error {
	var delay time.Duration
	for {
		select {
		case slots <- struct{}{}:
		case <-ctx.Done():
			return nil
		}
		connection, err := listener.Accept()
		if err != nil {
			<-slots
			if ctx.Err() != nil || errors.Is(err, net.ErrClosed) {
				return fmt.Errorf("accept on approval socket: %w", err)
			}
			report(&ConnectionError{Op: OpAccept, Err: err})
			delay = nextAcceptDelay(delay)
			if !pause(ctx, delay) {
				return nil
			}
			continue
		}
		delay = 0
		connections.Go(func() {
			defer func() { <-slots }()
			answerConnection(ctx, connection, asker, report)
		})
	}
}

func answerConnection(ctx context.Context, connection net.Conn, asker contracts.ApprovalAsker, report func(error)) {
	if err := serveRequest(ctx, connection, asker); err != nil {
		report(err)
	}
}

func serveRequest(ctx context.Context, connection net.Conn, asker contracts.ApprovalAsker) error {
	request, err := readRequest(connection)
	if err != nil {
		return errors.Join(err, closeConnection(connection))
	}
	if err = connection.SetReadDeadline(time.Time{}); err != nil {
		return errors.Join(connectionError(OpRead, fmt.Errorf("clear read deadline: %w", err)), closeConnection(connection))
	}
	return answerRequest(ctx, connection, asker, request)
}

func answerRequest(ctx context.Context, connection net.Conn, asker contracts.ApprovalAsker, request contracts.ApprovalRequest) error {
	askCtx, cancel := context.WithTimeout(ctx, answerDeadline)
	defer cancel()
	var watcher sync.WaitGroup
	watcher.Go(func() { cancelWhenClientLeaves(connection, cancel) })
	answer := asker(askCtx, request)
	cancel()
	err := connectionError(OpAnswer, writeMessage(connection, answer))
	err = errors.Join(err, closeConnection(connection))
	watcher.Wait()
	return err
}

func cancelWhenClientLeaves(connection net.Conn, cancel context.CancelFunc) {
	defer cancel()
	var scratch [1]byte
	for {
		if _, err := connection.Read(scratch[:]); err != nil {
			return
		}
	}
}

func exchange(ctx context.Context, connection net.Conn, req contracts.ApprovalRequest) (contracts.ApprovalAnswer, error) {
	if err := checkPeer(connection); err != nil {
		return contracts.ApprovalAnswer{}, err
	}
	closed := make(chan error, 1)
	stopOnCancel := context.AfterFunc(ctx, func() { closed <- connection.Close() })
	answer, err := roundTrip(connection, req)
	if stopOnCancel() {
		return answer, err
	}
	return contracts.ApprovalAnswer{}, errors.Join(interrupted(ctx, err), <-closed)
}

func roundTrip(connection net.Conn, req contracts.ApprovalRequest) (contracts.ApprovalAnswer, error) {
	if err := writeMessage(connection, req); err != nil {
		return contracts.ApprovalAnswer{}, err
	}
	line, err := readLine(connection)
	if err != nil {
		return contracts.ApprovalAnswer{}, err
	}
	var answer contracts.ApprovalAnswer
	if err := json.Unmarshal(line, &answer); err != nil {
		return contracts.ApprovalAnswer{}, fmt.Errorf("decode approval answer: %w", err)
	}
	if _, err := contracts.ParseApprovalDecision(string(answer.Decision)); err != nil {
		return contracts.ApprovalAnswer{}, fmt.Errorf("read approval answer: %w", err)
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
		return request, connectionError(OpRead, fmt.Errorf("set approval read deadline: %w", err))
	}
	line, err := readLine(connection)
	if err != nil {
		return request, connectionError(OpRead, err)
	}
	if err := json.Unmarshal(line, &request); err != nil {
		return request, connectionError(OpRead, fmt.Errorf("decode approval request: %w", err))
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

func closeConnection(connection net.Conn) error {
	return connectionError(OpClose, connection.Close())
}

func connectionError(operation Op, err error) error {
	if err == nil {
		return nil
	}
	return &ConnectionError{Op: operation, Err: err}
}
