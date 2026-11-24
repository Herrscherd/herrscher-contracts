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
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	contracts "github.com/Herrscherd/herrscher-contracts"
)

// EnvApprovalSocket names the environment variable that carries the socket path
// from the backend to the hook it spawns.
const EnvApprovalSocket = "HERRSCHER_APPROVAL_SOCKET"

// MaxConcurrentConnections bounds how many connections Serve answers at once. A
// connection that arrives while all of them are taken is closed at once without an
// answer, and reported with OpBusy.
const MaxConcurrentConnections = 16

const (
	maxMessageBytes = 64 << 10
	messageDeadline = 5 * time.Second
	answerDeadline  = 10 * time.Minute
	probeTimeout    = time.Second
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
	// ErrMissingRequestID is returned by Ask when the request has an empty ID, so no
	// answer could be matched to it. Nothing is sent.
	ErrMissingRequestID = errors.New("approval request has no ID")
	// ErrRequestTooLarge is returned when a message exceeds 64 KiB.
	ErrRequestTooLarge = errors.New("approval message exceeds its size bound")
	// ErrAnswerMismatch is returned by Ask when the answer carries another ID than
	// the request it answers.
	ErrAnswerMismatch = errors.New("approval answer does not match its request")
	// ErrUnreadableAnswer is returned by Ask when the answer line is not one JSON
	// object that decodes as an approval answer, or when it exceeds 64 KiB.
	ErrUnreadableAnswer = errors.New("approval answer is unreadable")
	// ErrForeignPeer is returned by Ask when the process listening on the socket does
	// not run as the current user.
	ErrForeignPeer = errors.New("approval socket is answered by another user")
	// ErrBusy is the cause Serve reports for a connection refused because
	// MaxConcurrentConnections are already being answered. Ask never returns it: the
	// client sees its connection closed or reset without an answer.
	ErrBusy = errors.New("approval socket is answering the maximum number of connections")
	// ErrReportsDropped is the cause of a ReportsDroppedError.
	ErrReportsDropped = errors.New("approval connection errors were dropped while the error handler was busy")
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
	// OpBusy is a connection refused because MaxConcurrentConnections are taken.
	OpBusy Op = "busy"
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

// ReportsDroppedError reports how many failures could not be queued for the error
// handler, because the handler was still busy with earlier ones. Count is that
// number.
type ReportsDroppedError struct {
	Count uint64
}

// Error renders the number of dropped failures.
func (e *ReportsDroppedError) Error() string {
	return fmt.Sprintf("%d approval connection errors dropped while the error handler was busy", e.Count)
}

// Unwrap returns ErrReportsDropped.
func (e *ReportsDroppedError) Unwrap() error {
	return ErrReportsDropped
}

// Option tunes Serve.
type Option func(*serveSettings)

type serveSettings struct {
	report func(error)
}

// WithErrorHandler sends every failure that ends a connection, every refused
// connection and every failed accept to report as a *ConnectionError. Serve calls
// report from one goroutine of its own, through a bounded queue: a slow report
// never delays a connection, and a failure that finds the queue full is counted
// instead, then reported as a *ReportsDroppedError once the queue has room. Serve
// returns only after report has returned for every failure it queued, so report
// must return quickly and must not panic: Serve does not recover from a panic in
// report. Serve refuses to start without WithErrorHandler.
func WithErrorHandler(report func(error)) Option {
	return func(settings *serveSettings) {
		settings.report = report
	}
}

// Serve listens on the Unix socket at path and answers each approval request with
// asker, until ctx ends. The directory of path must exist and be owned by the
// current user with mode 0700, or Serve returns an error matching
// ErrInsecureDirectory. A nil asker returns ErrNoAsker, and no WithErrorHandler
// returns ErrNoErrorHandler. While it binds, Serve holds a lock file beside the
// socket, so two Serve calls on one path never both bind: the second one finds the
// first socket alive and returns an error matching ErrPathInUse. A context that
// ends while Serve waits for that lock makes Serve return its error at once. A
// socket left by a run that was killed is removed and replaced. Any other file at
// path, a live socket included, returns an error matching ErrPathInUse, and is
// left alone.
//
// Each connection carries one request line of at most 64 KiB, read within five
// seconds, and receives one answer line. At most MaxConcurrentConnections are
// answered at once. A connection that arrives beyond that bound is closed at once
// without an answer and reported with Op OpBusy and ErrBusy. A malformed,
// oversized or silent request closes its connection without calling asker, and the
// hook reads that as a denial. The context of asker ends when ctx ends, after ten
// minutes, or when the hook closes its connection or half-closes its write side: a
// hook keeps its write side open until the answer arrives. Once that context has
// ended, Serve writes no answer, and the hook reads the closed stream as a denial.
// Serve returns nil once ctx ends, after the socket file is removed, every
// connection in progress has returned, and report has returned for every failure
// it queued. asker must therefore return when its context ends.
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
	listener, err := bindSocket(ctx, path, directory)
	if err != nil {
		return err
	}
	return serveListener(ctx, listener, asker, settings.report)
}

// Ask sends req to the approval socket at path and waits for its answer. The
// wait lasts as long as ctx allows, so the caller sets the deadline. Ask returns
// an error when the directory of path is not private to the current user, when
// the socket cannot be reached, when the process answering it runs as another
// user, or when ctx ends first. Ask keeps its write side open while it waits: a
// half-closed connection ends the wait without an answer. The peer check runs on
// Linux only: on other platforms Ask relies on the private directory alone, and
// on Windows Ask returns an error matching errors.ErrUnsupported. A request whose
// line exceeds 64 KiB matches ErrRequestTooLarge and is not sent. An answer that
// is not one JSON object decoding as an approval answer, or that exceeds 64 KiB,
// matches ErrUnreadableAnswer and never ErrRequestTooLarge. A decoded answer whose
// decision is unknown or missing matches ErrInvalidDecision. A connection closed
// or reset before an answer is a read error that matches neither sentinel. An
// answer whose refusal is a code this version does not know is returned as it
// is, and it is never an allowance: Allowed reports false for any refusal. An
// answer whose ID is not req.ID, or is absent, matches ErrAnswerMismatch. A request
// whose ID is empty matches ErrMissingRequestID and is not sent. Every
// error means the daemon was not reached, and the caller must treat it as a
// denial.
func Ask(ctx context.Context, path string, req contracts.ApprovalRequest) (contracts.ApprovalAnswer, error) {
	if req.ID == "" {
		return contracts.ApprovalAnswer{}, ErrMissingRequestID
	}
	if _, err := checkPrivateDirectory(filepath.Dir(path)); err != nil {
		return contracts.ApprovalAnswer{}, err
	}
	line, err := encodeLine(req)
	if err != nil {
		return contracts.ApprovalAnswer{}, err
	}
	if len(line) > maxMessageBytes {
		return contracts.ApprovalAnswer{}, fmt.Errorf("%w: request is %d bytes", ErrRequestTooLarge, len(line))
	}
	var dialer net.Dialer
	connection, err := dialer.DialContext(ctx, "unix", path)
	if err != nil {
		return contracts.ApprovalAnswer{}, fmt.Errorf("dial approval socket: %w", err)
	}
	var closeErr error
	closeOnce := sync.OnceFunc(func() { closeErr = connection.Close() })
	answer, exchangeErr := exchange(ctx, connection, closeOnce, line, req.ID)
	closeOnce()
	if joined := errors.Join(exchangeErr, connectionError(OpClose, closeErr)); joined != nil {
		return contracts.ApprovalAnswer{}, joined
	}
	return answer, nil
}

func serveListener(ctx context.Context, listener net.Listener, asker contracts.ApprovalAsker, report func(error)) error {
	closed := make(chan error, 1)
	stopOnCancel := context.AfterFunc(ctx, func() { closed <- listener.Close() })
	var connections sync.WaitGroup
	slots := make(chan struct{}, MaxConcurrentConnections)
	failures := newReporter()
	delivered := make(chan struct{})
	go func() {
		failures.deliver(report)
		close(delivered)
	}()
	acceptErr := acceptConnections(ctx, listener, asker, failures, slots, &connections)
	var closeErr error
	if stopOnCancel() {
		closeErr = listener.Close()
	} else {
		closeErr = <-closed
	}
	connections.Wait()
	close(failures.queue)
	<-delivered
	if ctx.Err() != nil {
		return closeErr
	}
	return errors.Join(acceptErr, closeErr)
}

func acceptConnections(ctx context.Context, listener net.Listener, asker contracts.ApprovalAsker, failures *reporter, slots chan struct{}, connections *sync.WaitGroup) error {
	var delay time.Duration
	for {
		connection, err := listener.Accept()
		if err != nil {
			if ctx.Err() != nil || errors.Is(err, net.ErrClosed) {
				return fmt.Errorf("accept on approval socket: %w", err)
			}
			failures.send(&ConnectionError{Op: OpAccept, Err: err})
			delay = nextAcceptDelay(delay)
			if !pause(ctx, delay) {
				return nil
			}
			continue
		}
		delay = 0
		select {
		case slots <- struct{}{}:
			connections.Go(func() { answerConnection(ctx, connection, asker, failures, slots) })
		default:
			failures.send(refuseBusy(connection))
		}
	}
}

func refuseBusy(connection net.Conn) error {
	return errors.Join(connectionError(OpBusy, ErrBusy), closeConnection(connection))
}

func answerConnection(ctx context.Context, connection net.Conn, asker contracts.ApprovalAsker, failures *reporter, slots <-chan struct{}) {
	err := serveRequest(ctx, connection, asker)
	<-slots
	if err != nil {
		failures.send(err)
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
	var gate sync.Mutex
	departed := make(chan error, 1)
	go func() {
		departed <- readUntilDeparture(connection, func() {
			gate.Lock()
			defer gate.Unlock()
			cancel()
		})
	}()
	answer := asker(askCtx, request)
	gate.Lock()
	var writeErr error
	if askCtx.Err() == nil {
		writeErr = writeMessage(connection, answer)
	}
	gate.Unlock()
	err := connectionError(OpAnswer, writeErr)
	err = errors.Join(err, closeConnection(connection))
	return errors.Join(err, <-departed)
}

func readUntilDeparture(connection net.Conn, depart func()) error {
	defer depart()
	var scratch [1]byte
	for {
		if _, err := connection.Read(scratch[:]); err != nil {
			return departureError(err)
		}
	}
}

func departureError(err error) error {
	if errors.Is(err, net.ErrClosed) || errors.Is(err, io.EOF) || errors.Is(err, syscall.ECONNRESET) {
		return nil
	}
	return connectionError(OpRead, fmt.Errorf("watch approval client: %w", err))
}

func exchange(ctx context.Context, connection net.Conn, closeOnce func(), line []byte, requestID string) (contracts.ApprovalAnswer, error) {
	if err := checkPeer(connection); err != nil {
		return contracts.ApprovalAnswer{}, err
	}
	stopOnCancel := context.AfterFunc(ctx, closeOnce)
	answer, err := roundTrip(connection, line, requestID)
	if stopOnCancel() {
		return answer, err
	}
	return contracts.ApprovalAnswer{}, interrupted(ctx, err)
}

func roundTrip(connection net.Conn, line []byte, requestID string) (contracts.ApprovalAnswer, error) {
	if err := writeLine(connection, line); err != nil {
		return contracts.ApprovalAnswer{}, err
	}
	answerLine, err := readAnswerLine(connection)
	if err != nil {
		return contracts.ApprovalAnswer{}, err
	}
	if err := requireObject(answerLine); err != nil {
		return contracts.ApprovalAnswer{}, err
	}
	var answer contracts.ApprovalAnswer
	if err := json.Unmarshal(answerLine, &answer); err != nil {
		return contracts.ApprovalAnswer{}, decodeAnswerError(err)
	}
	if _, err := contracts.ParseApprovalDecision(string(answer.Decision)); err != nil {
		return contracts.ApprovalAnswer{}, fmt.Errorf("read approval answer: %w", err)
	}
	if answer.ID != requestID {
		return contracts.ApprovalAnswer{}, ErrAnswerMismatch
	}
	return answer, nil
}

func readAnswerLine(connection net.Conn) ([]byte, error) {
	answerLine, err := readLine(connection)
	if errors.Is(err, ErrRequestTooLarge) {
		return nil, fmt.Errorf("%w: the answer exceeds %d bytes", ErrUnreadableAnswer, maxMessageBytes)
	}
	return answerLine, err
}

func requireObject(answerLine []byte) error {
	if !bytes.HasPrefix(bytes.TrimSpace(answerLine), []byte("{")) {
		return fmt.Errorf("%w: the answer is not a JSON object", ErrUnreadableAnswer)
	}
	return nil
}

func decodeAnswerError(err error) error {
	if errors.Is(err, contracts.ErrInvalidDecision) {
		return fmt.Errorf("read approval answer: %w", err)
	}
	return fmt.Errorf("%w: the answer is not an approval answer", ErrUnreadableAnswer)
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

func encodeLine(value any) ([]byte, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("encode approval message: %w", err)
	}
	return append(encoded, '\n'), nil
}

func writeMessage(connection net.Conn, value any) error {
	line, err := encodeLine(value)
	if err != nil {
		return err
	}
	return writeLine(connection, line)
}

func writeLine(connection net.Conn, line []byte) error {
	if err := connection.SetWriteDeadline(time.Now().Add(messageDeadline)); err != nil {
		return fmt.Errorf("set approval write deadline: %w", err)
	}
	if _, err := connection.Write(line); err != nil {
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
