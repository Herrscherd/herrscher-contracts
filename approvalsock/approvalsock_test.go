package approvalsock_test

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	contracts "github.com/Herrscherd/herrscher-contracts"
	"github.com/Herrscherd/herrscher-contracts/approvalsock"
)

func allowAll(_ context.Context, req contracts.ApprovalRequest) contracts.ApprovalAnswer {
	return contracts.ApprovalAnswer{ID: req.ID, Decision: contracts.DecisionAllow, Reason: "reviewed"}
}

func denyAll(_ context.Context, req contracts.ApprovalRequest) contracts.ApprovalAnswer {
	return contracts.ApprovalAnswer{ID: req.ID, Decision: contracts.DecisionDeny, Reason: "refused"}
}

func logErrors(t *testing.T) approvalsock.Option {
	t.Helper()
	return approvalsock.WithErrorHandler(func(err error) { t.Logf("approval connection: %v", err) })
}

func privateDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.Chmod(dir, 0o700); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	return dir
}

func startServer(t *testing.T, asker contracts.ApprovalAsker) string {
	t.Helper()
	return startServerWith(t, asker, logErrors(t))
}

func startServerWith(t *testing.T, asker contracts.ApprovalAsker, opt approvalsock.Option) string {
	t.Helper()
	path := filepath.Join(privateDir(t), "a.sock")
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() { done <- approvalsock.Serve(ctx, path, asker, opt) }()
	t.Cleanup(func() {
		cancel()
		if err := <-done; err != nil {
			t.Errorf("Serve returned %v after cancel", err)
		}
	})
	waitForServer(t, path)
	return path
}

func waitForServer(t *testing.T, path string) {
	t.Helper()
	var dialer net.Dialer
	for range 200 {
		connection, err := dialer.DialContext(t.Context(), "unix", path)
		if err == nil {
			if closeErr := connection.Close(); closeErr != nil {
				t.Fatalf("close probe connection: %v", closeErr)
			}
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("approval socket never accepted a connection")
}

func sendRequest(t *testing.T, path, id string) net.Conn {
	t.Helper()
	var dialer net.Dialer
	connection, err := dialer.DialContext(t.Context(), "unix", path)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	encoded, err := json.Marshal(contracts.ApprovalRequest{ID: id, Tool: "Bash", Subject: "ls"})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if _, err := connection.Write(append(encoded, '\n')); err != nil {
		t.Fatalf("write request: %v", err)
	}
	return connection
}

func probeRequest(t *testing.T, path, id string) (net.Conn, error) {
	t.Helper()
	var dialer net.Dialer
	connection, err := dialer.DialContext(t.Context(), "unix", path)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	encoded, err := json.Marshal(contracts.ApprovalRequest{ID: id, Tool: "Bash", Subject: "ls"})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	_, writeErr := connection.Write(append(encoded, '\n'))
	return connection, writeErr
}

func closeConnection(t *testing.T, connection net.Conn) {
	t.Helper()
	if err := connection.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
}

func awaitReport(t *testing.T, reports <-chan error, matches func(error) bool) {
	t.Helper()
	timeout := time.After(5 * time.Second)
	for {
		select {
		case err := <-reports:
			if matches(err) {
				return
			}
		case <-timeout:
			t.Fatal("expected connection error was not reported")
		}
	}
}

func closeQuietly(t *testing.T, connection net.Conn) {
	t.Helper()
	if err := connection.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
		t.Errorf("close: %v", err)
	}
}

func waitUntil(t *testing.T, condition func() bool) {
	t.Helper()
	for range 400 {
		if condition() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("condition never held")
}

func TestAskReturnsAllowFromServer(t *testing.T) {
	t.Parallel()
	path := startServer(t, allowAll)
	answer, err := approvalsock.Ask(t.Context(), path, contracts.ApprovalRequest{ID: "1", Tool: "Bash", Subject: "ls"})
	if err != nil {
		t.Fatalf("Ask: %v", err)
	}
	if !answer.Allowed() || answer.Reason != "reviewed" {
		t.Fatalf("Ask = %+v, want an allowed answer with reason reviewed", answer)
	}
}

func TestAskReturnsDenyFromServer(t *testing.T) {
	t.Parallel()
	path := startServer(t, denyAll)
	answer, err := approvalsock.Ask(t.Context(), path, contracts.ApprovalRequest{ID: "2", Tool: "Bash", Subject: "rm"})
	if err != nil {
		t.Fatalf("Ask: %v", err)
	}
	if answer.Allowed() || answer.Reason != "refused" {
		t.Fatalf("Ask = %+v, want a denied answer with reason refused", answer)
	}
}

func TestAskTimesOutWhenNoAnswerComes(t *testing.T) {
	t.Parallel()
	path := startServer(t, func(ctx context.Context, req contracts.ApprovalRequest) contracts.ApprovalAnswer {
		<-ctx.Done()
		return contracts.ApprovalAnswer{ID: req.ID}
	})
	ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
	defer cancel()
	_, err := approvalsock.Ask(ctx, path, contracts.ApprovalRequest{ID: "3"})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Ask error = %v, want context.DeadlineExceeded", err)
	}
}

func TestServeDropsMalformedRequestWithoutAsking(t *testing.T) {
	t.Parallel()
	var asked atomic.Bool
	path := startServer(t, func(ctx context.Context, req contracts.ApprovalRequest) contracts.ApprovalAnswer {
		asked.Store(true)
		return allowAll(ctx, req)
	})
	var dialer net.Dialer
	connection, err := dialer.DialContext(t.Context(), "unix", path)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	if _, writeErr := connection.Write([]byte("not json\n")); writeErr != nil {
		t.Fatalf("write: %v", writeErr)
	}
	reply, err := io.ReadAll(connection)
	closeConnection(t, connection)
	if err != nil || len(reply) != 0 {
		t.Fatalf("reply = %q, err = %v; want no reply and a closed connection", reply, err)
	}
	if asked.Load() {
		t.Fatal("asker was called for a malformed request")
	}
}

func TestServeReportsMalformedRequest(t *testing.T) {
	t.Parallel()
	reports := make(chan error, 8)
	path := startServerWith(t, allowAll, approvalsock.WithErrorHandler(func(err error) {
		select {
		case reports <- err:
		default:
		}
	}))
	var dialer net.Dialer
	connection, err := dialer.DialContext(t.Context(), "unix", path)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	if _, writeErr := connection.Write([]byte("not json\n")); writeErr != nil {
		t.Fatalf("write: %v", writeErr)
	}
	closeConnection(t, connection)
	awaitReport(t, reports, func(err error) bool {
		connectionErr, ok := errors.AsType[*approvalsock.ConnectionError](err)
		_, isSyntax := errors.AsType[*json.SyntaxError](err)
		return ok && connectionErr.Op == approvalsock.OpRead && isSyntax
	})
}

func TestServeDropsSilentConnectionAfterDeadline(t *testing.T) {
	t.Parallel()
	var asked atomic.Bool
	path := startServer(t, func(ctx context.Context, req contracts.ApprovalRequest) contracts.ApprovalAnswer {
		asked.Store(true)
		return allowAll(ctx, req)
	})
	var dialer net.Dialer
	connection, err := dialer.DialContext(t.Context(), "unix", path)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	reply, err := io.ReadAll(connection)
	closeConnection(t, connection)
	if err != nil || len(reply) != 0 {
		t.Fatalf("reply = %q, err = %v; want the server to drop the silent connection", reply, err)
	}
	if asked.Load() {
		t.Fatal("asker was called for a silent connection")
	}
}

func requestLineOfLength(t *testing.T, length int) []byte {
	t.Helper()
	empty, err := json.Marshal(contracts.ApprovalRequest{ID: "10", Tool: "Bash"})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	subject := strings.Repeat("x", length-len(empty))
	line, err := json.Marshal(contracts.ApprovalRequest{ID: "10", Tool: "Bash", Subject: subject})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if len(line) != length {
		t.Fatalf("request line is %d bytes, want %d", len(line), length)
	}
	return append(line, '\n')
}

func TestServeAcceptsLineAtMaximumSize(t *testing.T) {
	t.Parallel()
	var asked atomic.Bool
	path := startServer(t, func(ctx context.Context, req contracts.ApprovalRequest) contracts.ApprovalAnswer {
		asked.Store(true)
		return allowAll(ctx, req)
	})
	var dialer net.Dialer
	connection, err := dialer.DialContext(t.Context(), "unix", path)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	if _, writeErr := connection.Write(requestLineOfLength(t, 65535)); writeErr != nil {
		t.Fatalf("write: %v", writeErr)
	}
	reply, err := bufio.NewReader(connection).ReadBytes('\n')
	closeConnection(t, connection)
	if err != nil {
		t.Fatalf("read answer: %v", err)
	}
	var answer contracts.ApprovalAnswer
	if err := json.Unmarshal(reply, &answer); err != nil || !answer.Allowed() {
		t.Fatalf("answer = %q, error %v; want an allowed answer for a 65535-byte line", reply, err)
	}
	if !asked.Load() {
		t.Fatal("asker was not called for a 65535-byte line")
	}
}

func TestServeRefusesLineOverMaximumSize(t *testing.T) {
	t.Parallel()
	var asked atomic.Bool
	reports := make(chan error, 8)
	path := startServerWith(t, func(ctx context.Context, req contracts.ApprovalRequest) contracts.ApprovalAnswer {
		asked.Store(true)
		return allowAll(ctx, req)
	}, approvalsock.WithErrorHandler(func(err error) {
		select {
		case reports <- err:
		default:
		}
	}))
	var dialer net.Dialer
	connection, err := dialer.DialContext(t.Context(), "unix", path)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	if _, writeErr := connection.Write(requestLineOfLength(t, 65536)); writeErr != nil {
		t.Fatalf("write: %v", writeErr)
	}
	reply, err := io.ReadAll(connection)
	closeConnection(t, connection)
	if err != nil && !errors.Is(err, syscall.ECONNRESET) {
		t.Fatalf("read reply: %v", err)
	}
	if len(reply) != 0 {
		t.Fatalf("reply = %q, want no reply to a 65536-byte line", reply)
	}
	if asked.Load() {
		t.Fatal("asker was called for a 65536-byte line")
	}
	awaitReport(t, reports, func(err error) bool {
		return errors.Is(err, approvalsock.ErrRequestTooLarge)
	})
}

func TestAskFailsWhenSocketIsAbsent(t *testing.T) {
	t.Parallel()
	path := filepath.Join(privateDir(t), "absent.sock")
	_, err := approvalsock.Ask(t.Context(), path, contracts.ApprovalRequest{ID: "5"})
	if !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("Ask error = %v, want fs.ErrNotExist", err)
	}
}

func TestAskRejectsAnswerWithAnotherID(t *testing.T) {
	t.Parallel()
	path := startServer(t, func(_ context.Context, _ contracts.ApprovalRequest) contracts.ApprovalAnswer {
		return contracts.ApprovalAnswer{ID: "other", Decision: contracts.DecisionAllow}
	})
	answer, err := approvalsock.Ask(t.Context(), path, contracts.ApprovalRequest{ID: "6"})
	if !errors.Is(err, approvalsock.ErrAnswerMismatch) {
		t.Fatalf("Ask = %+v, error %v; want ErrAnswerMismatch", answer, err)
	}
}

func TestAskRejectsUnreadableAnswer(t *testing.T) {
	t.Parallel()
	path := fakeServer(t, "not json\n")
	_, err := approvalsock.Ask(t.Context(), path, contracts.ApprovalRequest{ID: "7"})
	if !errors.Is(err, approvalsock.ErrUnreadableAnswer) {
		t.Fatalf("Ask error = %v, want approvalsock.ErrUnreadableAnswer", err)
	}
}

func TestAskClassifiesUnreadableAnswerShapes(t *testing.T) {
	t.Parallel()
	for name, reply := range map[string]string{
		"truncated":  `{"ID":"12","Decision":"al` + "\n",
		"wrong type": `{"ID":12,"Decision":"allow"}` + "\n",
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			path := fakeServer(t, reply)
			_, err := approvalsock.Ask(t.Context(), path, contracts.ApprovalRequest{ID: "12"})
			if !errors.Is(err, approvalsock.ErrUnreadableAnswer) {
				t.Fatalf("Ask error = %v, want approvalsock.ErrUnreadableAnswer", err)
			}
			if errors.Is(err, contracts.ErrInvalidDecision) {
				t.Fatalf("Ask error = %v, an unreadable answer must not match ErrInvalidDecision", err)
			}
		})
	}
}

func TestAskReportsAnAnswerClosedWithoutSentinel(t *testing.T) {
	t.Parallel()
	path := fakeServer(t, "")
	_, err := approvalsock.Ask(t.Context(), path, contracts.ApprovalRequest{ID: "13"})
	if err == nil {
		t.Fatal("Ask accepted a connection closed without an answer")
	}
	if errors.Is(err, approvalsock.ErrUnreadableAnswer) || errors.Is(err, contracts.ErrInvalidDecision) {
		t.Fatalf("Ask error = %v, a closed connection matches neither ErrUnreadableAnswer nor ErrInvalidDecision", err)
	}
}

func TestAskRefusesUnknownDecision(t *testing.T) {
	t.Parallel()
	for name, reply := range map[string]string{
		"maybe":   `{"ID":"11","Decision":"maybe"}`,
		"missing": `{"ID":"11"}`,
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			path := fakeServer(t, reply+"\n")
			_, err := approvalsock.Ask(t.Context(), path, contracts.ApprovalRequest{ID: "11"})
			if !errors.Is(err, contracts.ErrInvalidDecision) {
				t.Fatalf("Ask error = %v, want contracts.ErrInvalidDecision", err)
			}
		})
	}
}

func TestServeReportsUnsendableDecision(t *testing.T) {
	t.Parallel()
	reports := make(chan error, 8)
	path := startServerWith(t, func(_ context.Context, req contracts.ApprovalRequest) contracts.ApprovalAnswer {
		return contracts.ApprovalAnswer{ID: req.ID, Decision: "maybe"}
	}, approvalsock.WithErrorHandler(func(err error) {
		select {
		case reports <- err:
		default:
		}
	}))
	_, err := approvalsock.Ask(t.Context(), path, contracts.ApprovalRequest{ID: "12"})
	if err == nil {
		t.Fatal("Ask accepted an answer the server could not send")
	}
	awaitReport(t, reports, func(err error) bool {
		connectionErr, ok := errors.AsType[*approvalsock.ConnectionError](err)
		return ok && connectionErr.Op == approvalsock.OpAnswer && errors.Is(err, contracts.ErrInvalidDecision)
	})
}

func fakeServer(t *testing.T, reply string) string {
	t.Helper()
	path := filepath.Join(privateDir(t), "fake.sock")
	var listenConfig net.ListenConfig
	listener, err := listenConfig.Listen(t.Context(), "unix", path)
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	served := make(chan error, 1)
	go func() { served <- answerOnce(listener, reply) }()
	t.Cleanup(func() {
		if err := listener.Close(); err != nil {
			t.Errorf("close fake listener: %v", err)
		}
		if err := <-served; err != nil && !errors.Is(err, net.ErrClosed) {
			t.Errorf("fake server: %v", err)
		}
	})
	return path
}

func answerOnce(listener net.Listener, reply string) error {
	connection, err := listener.Accept()
	if err != nil {
		return err
	}
	if _, err := bufio.NewReader(connection).ReadBytes('\n'); err != nil {
		return err
	}
	if _, err := connection.Write([]byte(reply)); err != nil {
		return err
	}
	return connection.Close()
}

func TestAskRefusesOpenDirectoryWithoutDialing(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := os.Chmod(dir, 0o755); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	path := filepath.Join(dir, "a.sock")
	var listenConfig net.ListenConfig
	listener, err := listenConfig.Listen(t.Context(), "unix", path)
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() {
		if closeErr := listener.Close(); closeErr != nil {
			t.Errorf("close listener: %v", closeErr)
		}
	})
	_, err = approvalsock.Ask(t.Context(), path, contracts.ApprovalRequest{ID: "13"})
	if !errors.Is(err, approvalsock.ErrInsecureDirectory) {
		t.Fatalf("Ask error = %v, want ErrInsecureDirectory", err)
	}
	unixListener, isUnix := listener.(*net.UnixListener)
	if !isUnix {
		t.Fatalf("listener is %T, want *net.UnixListener", listener)
	}
	if deadlineErr := unixListener.SetDeadline(time.Now().Add(100 * time.Millisecond)); deadlineErr != nil {
		t.Fatalf("set deadline: %v", deadlineErr)
	}
	connection, err := unixListener.Accept()
	if err == nil {
		closeConnection(t, connection)
		t.Fatal("Ask dialed a socket in an open directory")
	}
}

func TestServeRefusesOpenDirectory(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := os.Chmod(dir, 0o755); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	err := approvalsock.Serve(t.Context(), filepath.Join(dir, "a.sock"), allowAll, logErrors(t))
	if !errors.Is(err, approvalsock.ErrInsecureDirectory) {
		t.Fatalf("Serve error = %v, want ErrInsecureDirectory", err)
	}
}

func TestServeRefusesMissingDirectory(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "missing", "a.sock")
	err := approvalsock.Serve(t.Context(), path, allowAll, logErrors(t))
	if !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("Serve error = %v, want fs.ErrNotExist", err)
	}
}

func TestServeRefusesNilAsker(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "a.sock")
	err := approvalsock.Serve(t.Context(), path, nil, logErrors(t))
	if !errors.Is(err, approvalsock.ErrNoAsker) {
		t.Fatalf("Serve error = %v, want ErrNoAsker", err)
	}
}

func TestServeRefusesMissingErrorHandler(t *testing.T) {
	t.Parallel()
	path := filepath.Join(privateDir(t), "a.sock")
	err := approvalsock.Serve(t.Context(), path, allowAll)
	if !errors.Is(err, approvalsock.ErrNoErrorHandler) {
		t.Fatalf("Serve error = %v, want ErrNoErrorHandler", err)
	}
}

func TestServeRemovesSocketOnCancel(t *testing.T) {
	t.Parallel()
	path := filepath.Join(privateDir(t), "a.sock")
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() { done <- approvalsock.Serve(ctx, path, allowAll, logErrors(t)) }()
	waitForServer(t, path)
	cancel()
	if err := <-done; err != nil {
		t.Fatalf("Serve after cancel: %v", err)
	}
	if _, err := os.Stat(path); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("socket after cancel: stat error %v, want fs.ErrNotExist", err)
	}
}

func listenOrphan(t *testing.T, path string) {
	t.Helper()
	var listenConfig net.ListenConfig
	listener, err := listenConfig.Listen(t.Context(), "unix", path)
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	unixListener, isUnix := listener.(*net.UnixListener)
	if !isUnix {
		t.Fatalf("listener is %T, want *net.UnixListener", listener)
	}
	unixListener.SetUnlinkOnClose(false)
	if err := listener.Close(); err != nil {
		t.Fatalf("close listener: %v", err)
	}
}

func TestServeStartsOverOrphanSocket(t *testing.T) {
	t.Parallel()
	path := filepath.Join(privateDir(t), "a.sock")
	listenOrphan(t, path)
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() { done <- approvalsock.Serve(ctx, path, allowAll, logErrors(t)) }()
	waitForServer(t, path)
	cancel()
	if err := <-done; err != nil {
		t.Fatalf("Serve over orphan socket: %v", err)
	}
}

func TestServeRefusesRegularFileAtPath(t *testing.T) {
	t.Parallel()
	path := filepath.Join(privateDir(t), "a.sock")
	if err := os.WriteFile(path, []byte("keep"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	err := approvalsock.Serve(t.Context(), path, allowAll, logErrors(t))
	if !errors.Is(err, approvalsock.ErrPathInUse) {
		t.Fatalf("Serve error = %v, want ErrPathInUse", err)
	}
	if content, readErr := os.ReadFile(path); readErr != nil || string(content) != "keep" {
		t.Fatalf("regular file was touched: %q, %v", content, readErr)
	}
}

func TestServeRefusesLiveSocketAtPath(t *testing.T) {
	t.Parallel()
	path := filepath.Join(privateDir(t), "a.sock")
	var listenConfig net.ListenConfig
	listener, err := listenConfig.Listen(t.Context(), "unix", path)
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() {
		if closeErr := listener.Close(); closeErr != nil {
			t.Errorf("close live listener: %v", closeErr)
		}
	})
	err = approvalsock.Serve(t.Context(), path, allowAll, logErrors(t))
	if !errors.Is(err, approvalsock.ErrPathInUse) {
		t.Fatalf("Serve error = %v, want ErrPathInUse", err)
	}
}

func TestServeRefusesSymlinkAtPath(t *testing.T) {
	t.Parallel()
	dir := privateDir(t)
	target := filepath.Join(dir, "target.sock")
	path := filepath.Join(dir, "a.sock")
	if err := os.Symlink(target, path); err != nil {
		t.Fatalf("symlink: %v", err)
	}
	err := approvalsock.Serve(t.Context(), path, allowAll, logErrors(t))
	if !errors.Is(err, approvalsock.ErrPathInUse) {
		t.Fatalf("Serve error = %v, want ErrPathInUse", err)
	}
	if _, err := os.Lstat(path); err != nil {
		t.Fatalf("symlink was removed: %v", err)
	}
}

func TestServeRefusesConnectionsBeyondTheBound(t *testing.T) {
	t.Parallel()
	var active atomic.Int32
	reports := make(chan error, 64)
	path := startServerWith(t, func(ctx context.Context, req contracts.ApprovalRequest) contracts.ApprovalAnswer {
		active.Add(1)
		<-ctx.Done()
		return contracts.ApprovalAnswer{ID: req.ID}
	}, approvalsock.WithErrorHandler(func(err error) {
		select {
		case reports <- err:
		default:
		}
	}))
	awaitReport(t, reports, func(err error) bool {
		connectionErr, ok := errors.AsType[*approvalsock.ConnectionError](err)
		return ok && connectionErr.Op == approvalsock.OpRead && errors.Is(err, io.ErrUnexpectedEOF)
	})
	held := make([]net.Conn, 0, approvalsock.MaxConcurrentConnections)
	for range approvalsock.MaxConcurrentConnections {
		held = append(held, sendRequest(t, path, "held"))
	}
	t.Cleanup(func() {
		for _, connection := range held[1:] {
			closeQuietly(t, connection)
		}
	})
	waitUntil(t, func() bool { return active.Load() == approvalsock.MaxConcurrentConnections })
	extraCtx, cancelExtra := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancelExtra()
	_, err := approvalsock.Ask(extraCtx, path, contracts.ApprovalRequest{ID: "extra"})
	if err == nil {
		t.Fatal("Ask was answered beyond the connection bound")
	}
	awaitReport(t, reports, func(err error) bool {
		connectionErr, ok := errors.AsType[*approvalsock.ConnectionError](err)
		return ok && connectionErr.Op == approvalsock.OpBusy && errors.Is(err, approvalsock.ErrBusy)
	})
	if got := active.Load(); got != approvalsock.MaxConcurrentConnections {
		t.Fatalf("asker ran %d times with the bound reached, want %d", got, approvalsock.MaxConcurrentConnections)
	}
	closeConnection(t, held[0])
	waitUntil(t, func() bool {
		probe, writeErr := probeRequest(t, path, "freed")
		time.Sleep(20 * time.Millisecond)
		answered := writeErr == nil && active.Load() == approvalsock.MaxConcurrentConnections+1
		closeQuietly(t, probe)
		return answered
	})
}

func TestServeKeepsOrphanSocketWhenCancelledBeforeProbe(t *testing.T) {
	t.Parallel()
	path := filepath.Join(privateDir(t), "a.sock")
	listenOrphan(t, path)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	err := approvalsock.Serve(ctx, path, allowAll, logErrors(t))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Serve error = %v, want context.Canceled", err)
	}
	if errors.Is(err, approvalsock.ErrPathInUse) {
		t.Fatalf("cancelled probe was reported as a live socket: %v", err)
	}
	if _, statErr := os.Lstat(path); statErr != nil {
		t.Fatalf("orphan socket was removed by a cancelled Serve: %v", statErr)
	}
}

func TestAskRefusesOversizedRequestWithoutDialing(t *testing.T) {
	t.Parallel()
	dir := privateDir(t)
	path := filepath.Join(dir, "a.sock")
	var listenConfig net.ListenConfig
	listener, err := listenConfig.Listen(t.Context(), "unix", path)
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() {
		if closeErr := listener.Close(); closeErr != nil {
			t.Errorf("close listener: %v", closeErr)
		}
	})
	_, err = approvalsock.Ask(t.Context(), path, contracts.ApprovalRequest{ID: "16", Subject: strings.Repeat("s", 64<<10)})
	if !errors.Is(err, approvalsock.ErrRequestTooLarge) {
		t.Fatalf("Ask error = %v, want ErrRequestTooLarge", err)
	}
	unixListener, isUnix := listener.(*net.UnixListener)
	if !isUnix {
		t.Fatalf("listener is %T, want *net.UnixListener", listener)
	}
	if deadlineErr := unixListener.SetDeadline(time.Now().Add(100 * time.Millisecond)); deadlineErr != nil {
		t.Fatalf("set deadline: %v", deadlineErr)
	}
	connection, err := unixListener.Accept()
	if err == nil {
		closeConnection(t, connection)
		t.Fatal("Ask dialed the socket with an oversized request")
	}
}

func silentServer(t *testing.T) string {
	t.Helper()
	path := filepath.Join(privateDir(t), "silent.sock")
	var listenConfig net.ListenConfig
	listener, err := listenConfig.Listen(t.Context(), "unix", path)
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() {
		if closeErr := listener.Close(); closeErr != nil && !errors.Is(closeErr, net.ErrClosed) {
			t.Errorf("close listener: %v", closeErr)
		}
	})
	go func() {
		connection, acceptErr := listener.Accept()
		if acceptErr != nil {
			return
		}
		<-t.Context().Done()
		closeQuietly(t, connection)
	}()
	return path
}

func TestAskReportsDeadlineWithoutItsOwnCloseError(t *testing.T) {
	t.Parallel()
	path := silentServer(t)
	ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
	defer cancel()
	_, err := approvalsock.Ask(ctx, path, contracts.ApprovalRequest{ID: "17"})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Ask error = %v, want context.DeadlineExceeded", err)
	}
	if errors.Is(err, net.ErrClosed) {
		t.Fatalf("Ask reported its own close as an error: %v", err)
	}
}

func TestServeCancelsAskerWhenClientLeaves(t *testing.T) {
	t.Parallel()
	cancelled := make(chan struct{})
	path := startServer(t, func(ctx context.Context, req contracts.ApprovalRequest) contracts.ApprovalAnswer {
		<-ctx.Done()
		close(cancelled)
		return contracts.ApprovalAnswer{ID: req.ID}
	})
	connection := sendRequest(t, path, "15")
	time.Sleep(50 * time.Millisecond)
	closeConnection(t, connection)
	select {
	case <-cancelled:
	case <-time.After(5 * time.Second):
		t.Fatal("asker context was not cancelled after the client left")
	}
}

func TestServeWritesNoAnswerAfterClientHalfCloses(t *testing.T) {
	t.Parallel()
	cancelled := make(chan struct{})
	path := startServer(t, func(ctx context.Context, req contracts.ApprovalRequest) contracts.ApprovalAnswer {
		<-ctx.Done()
		close(cancelled)
		return contracts.ApprovalAnswer{ID: req.ID, Decision: contracts.DecisionAllow}
	})
	connection := sendRequest(t, path, "18")
	unixConnection, isUnix := connection.(*net.UnixConn)
	if !isUnix {
		t.Fatalf("client connection is %T, want *net.UnixConn", connection)
	}
	if err := unixConnection.CloseWrite(); err != nil {
		t.Fatalf("half-close: %v", err)
	}
	select {
	case <-cancelled:
	case <-time.After(5 * time.Second):
		t.Fatal("asker context survived the client's half-close")
	}
	if err := connection.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
		t.Fatalf("set read deadline: %v", err)
	}
	reply, err := io.ReadAll(connection)
	if err != nil {
		t.Fatalf("read reply: %v", err)
	}
	if len(reply) != 0 {
		t.Fatalf("client received %q after its half-close, want no answer", reply)
	}
	closeQuietly(t, connection)
}
