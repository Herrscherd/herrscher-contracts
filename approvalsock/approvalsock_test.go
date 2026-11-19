package approvalsock_test

import (
	"bufio"
	"context"
	"errors"
	"io"
	"io/fs"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
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
	path := filepath.Join(privateDir(t), "a.sock")
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() { done <- approvalsock.Serve(ctx, path, asker) }()
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
	if closeErr := connection.Close(); closeErr != nil {
		t.Fatalf("close: %v", closeErr)
	}
	if err != nil || len(reply) != 0 {
		t.Fatalf("reply = %q, err = %v; want no reply and a closed connection", reply, err)
	}
	if asked.Load() {
		t.Fatal("asker was called for a malformed request")
	}
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
	if closeErr := connection.Close(); closeErr != nil {
		t.Fatalf("close: %v", closeErr)
	}
	if err != nil || len(reply) != 0 {
		t.Fatalf("reply = %q, err = %v; want the server to drop the silent connection", reply, err)
	}
	if asked.Load() {
		t.Fatal("asker was called for a silent connection")
	}
}

func TestAskRefusesOversizedRequest(t *testing.T) {
	t.Parallel()
	var asked atomic.Bool
	path := startServer(t, func(ctx context.Context, req contracts.ApprovalRequest) contracts.ApprovalAnswer {
		asked.Store(true)
		return allowAll(ctx, req)
	})
	oversized := contracts.ApprovalRequest{ID: "4", Tool: "Bash", Subject: strings.Repeat("x", 70<<10)}
	answer, err := approvalsock.Ask(t.Context(), path, oversized)
	if err == nil {
		t.Fatalf("Ask returned %+v for an oversized request, want an error", answer)
	}
	if asked.Load() {
		t.Fatal("asker was called for an oversized request")
	}
}

func TestAskFailsWhenSocketIsAbsent(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "absent.sock")
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
	if err == nil {
		t.Fatal("Ask accepted an answer that is not JSON")
	}
}

func fakeServer(t *testing.T, reply string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "fake.sock")
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

func TestServeRefusesOpenDirectory(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := os.Chmod(dir, 0o755); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	err := approvalsock.Serve(t.Context(), filepath.Join(dir, "a.sock"), allowAll)
	if !errors.Is(err, approvalsock.ErrInsecureDirectory) {
		t.Fatalf("Serve error = %v, want ErrInsecureDirectory", err)
	}
}

func TestServeRefusesMissingDirectory(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "missing", "a.sock")
	err := approvalsock.Serve(t.Context(), path, allowAll)
	if !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("Serve error = %v, want fs.ErrNotExist", err)
	}
}

func TestServeRefusesNilAsker(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "a.sock")
	err := approvalsock.Serve(t.Context(), path, nil)
	if !errors.Is(err, approvalsock.ErrNoAsker) {
		t.Fatalf("Serve error = %v, want ErrNoAsker", err)
	}
}

func TestServeRemovesSocketOnCancel(t *testing.T) {
	t.Parallel()
	path := filepath.Join(privateDir(t), "a.sock")
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() { done <- approvalsock.Serve(ctx, path, allowAll) }()
	waitForServer(t, path)
	cancel()
	if err := <-done; err != nil {
		t.Fatalf("Serve after cancel: %v", err)
	}
	if _, err := os.Stat(path); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("socket after cancel: stat error %v, want fs.ErrNotExist", err)
	}
}
