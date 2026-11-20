package approvalsock_test

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"net"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	contracts "github.com/Herrscherd/herrscher-contracts"
	"github.com/Herrscherd/herrscher-contracts/approvalsock"
)

const refusedBeyondSlots = 40

type blockingReports struct {
	mu       sync.Mutex
	received []error
	entered  chan struct{}
	release  chan struct{}
}

func newBlockingReports() *blockingReports {
	return &blockingReports{entered: make(chan struct{}, 1), release: make(chan struct{})}
}

func (b *blockingReports) report(err error) {
	b.mu.Lock()
	b.received = append(b.received, err)
	b.mu.Unlock()
	select {
	case b.entered <- struct{}{}:
	default:
	}
	<-b.release
}

func (b *blockingReports) count() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.received)
}

func (b *blockingReports) total() uint64 {
	b.mu.Lock()
	defer b.mu.Unlock()
	var total uint64
	for _, err := range b.received {
		if dropped, ok := errors.AsType[*approvalsock.ReportsDroppedError](err); ok {
			total += dropped.Count
			continue
		}
		total++
	}
	return total
}

func sendMalformed(t *testing.T, path string) {
	t.Helper()
	var dialer net.Dialer
	connection, err := dialer.DialContext(t.Context(), "unix", path)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	if _, err := connection.Write([]byte("not json\n")); err != nil {
		t.Fatalf("write: %v", err)
	}
	closeConnection(t, connection)
}

func awaitEntered(t *testing.T, reports *blockingReports) {
	t.Helper()
	select {
	case <-reports.entered:
	case <-time.After(5 * time.Second):
		t.Fatal("error handler was never called")
	}
}

type reportingServer struct {
	path string
	stop func()
}

func startReportingServer(t *testing.T, asker contracts.ApprovalAsker, reports *blockingReports) reportingServer {
	t.Helper()
	path := filepath.Join(privateDir(t), "a.sock")
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() { done <- approvalsock.Serve(ctx, path, asker, approvalsock.WithErrorHandler(reports.report)) }()
	stop := sync.OnceFunc(func() {
		cancel()
		if err := <-done; err != nil {
			t.Errorf("Serve returned %v after cancel", err)
		}
	})
	t.Cleanup(stop)
	waitForServer(t, path)
	awaitEntered(t, reports)
	return reportingServer{path: path, stop: stop}
}

func awaitAnswer(t *testing.T, path string) {
	t.Helper()
	for range 200 {
		ctx, cancel := context.WithTimeout(t.Context(), time.Second)
		_, err := approvalsock.Ask(ctx, path, contracts.ApprovalRequest{ID: "ready", Tool: "Bash", Subject: "ls"})
		cancel()
		if err == nil {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("approval socket never answered")
}

func askUntilAllowed(t *testing.T, path, id string) int {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	refused := 0
	for {
		ctx, cancel := context.WithTimeout(t.Context(), time.Second)
		answer, err := approvalsock.Ask(ctx, path, contracts.ApprovalRequest{ID: id, Tool: "Bash", Subject: "ls"})
		cancel()
		if err == nil {
			if !answer.Allowed() {
				t.Fatalf("answer = %+v, want allow", answer)
			}
			return refused
		}
		refused++
		if time.Now().After(deadline) {
			t.Fatalf("Ask %q never answered: %v", id, err)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func holdEverySlot(t *testing.T, path string, asked <-chan struct{}) []net.Conn {
	t.Helper()
	holders := make([]net.Conn, 0, approvalsock.MaxConcurrentConnections)
	t.Cleanup(func() {
		for _, connection := range holders {
			closeQuietly(t, connection)
		}
	})
	for range approvalsock.MaxConcurrentConnections {
		holders = append(holders, sendRequest(t, path, "hold"))
	}
	for range approvalsock.MaxConcurrentConnections {
		awaitSignal(t, asked, "a holding request never reached the asker")
	}
	return holders
}

func awaitHolderAnswers(t *testing.T, holders []net.Conn) {
	t.Helper()
	for _, connection := range holders {
		if err := connection.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
			t.Fatalf("set holder deadline: %v", err)
		}
		if _, err := io.Copy(io.Discard, connection); err != nil {
			t.Fatalf("read holder answer to its close: %v", err)
		}
	}
}

func awaitHolderReturns(t *testing.T, returned <-chan struct{}) {
	t.Helper()
	for range approvalsock.MaxConcurrentConnections {
		awaitSignal(t, returned, "a holding request never returned from the asker")
	}
}

func awaitSignal(t *testing.T, signal <-chan struct{}, message string) {
	t.Helper()
	select {
	case <-signal:
	case <-time.After(5 * time.Second):
		t.Fatal(message)
	}
}

func refuseBeyondSlots(t *testing.T, path string) {
	t.Helper()
	var dialer net.Dialer
	for range refusedBeyondSlots {
		connection, err := dialer.DialContext(t.Context(), "unix", path)
		if err != nil {
			t.Fatalf("dial while slots are held: %v", err)
		}
		closeConnection(t, connection)
	}
}

func requireRefusedWhileFull(t *testing.T, path string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	_, err := approvalsock.Ask(ctx, path, contracts.ApprovalRequest{ID: "barrier", Tool: "Bash", Subject: "ls"})
	if err == nil {
		t.Fatal("a request beyond the slots was answered while every slot was held")
	}
	if errors.Is(err, context.DeadlineExceeded) {
		t.Fatal("accept loop stalled behind a blocked report")
	}
}

func TestServeBlockedReportNeverStallsAcceptOrAnswer(t *testing.T) {
	t.Parallel()
	reports := newBlockingReports()
	releaseReports := sync.OnceFunc(func() { close(reports.release) })
	holdAsker := make(chan struct{})
	releaseHolders := sync.OnceFunc(func() { close(holdAsker) })
	asked := make(chan struct{}, approvalsock.MaxConcurrentConnections)
	returned := make(chan struct{}, approvalsock.MaxConcurrentConnections)
	asker := func(ctx context.Context, req contracts.ApprovalRequest) contracts.ApprovalAnswer {
		if req.ID == "hold" {
			asked <- struct{}{}
			select {
			case <-holdAsker:
			case <-ctx.Done():
			}
			returned <- struct{}{}
		}
		return allowAll(ctx, req)
	}
	server := startReportingServer(t, asker, reports)
	path := server.path
	t.Cleanup(releaseReports)
	t.Cleanup(releaseHolders)

	holders := holdEverySlot(t, path, asked)
	refuseBeyondSlots(t, path)
	requireRefusedWhileFull(t, path)

	releaseHolders()
	awaitHolderReturns(t, returned)
	awaitHolderAnswers(t, holders)
	retries := askUntilAllowed(t, path, "legit")

	if got := reports.count(); got != 1 {
		t.Fatalf("handler saw %d reports while blocked, want only the first", got)
	}
	releaseReports()
	server.stop()
	const probe = 1
	const barrier = 1
	if want := uint64(probe + refusedBeyondSlots + barrier + retries); reports.total() != want {
		t.Fatalf("reported failures = %d, want %d (%d probe, %d refused, %d barrier, %d retries)", reports.total(), want, probe, refusedBeyondSlots, barrier, retries)
	}
}

func TestServeReturnsOnlyAfterBlockedReportIsReleased(t *testing.T) {
	t.Parallel()
	reports := newBlockingReports()
	releaseOnce := sync.OnceFunc(func() { close(reports.release) })
	t.Cleanup(releaseOnce)
	path := filepath.Join(privateDir(t), "a.sock")
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() { done <- approvalsock.Serve(ctx, path, allowAll, approvalsock.WithErrorHandler(reports.report)) }()
	awaitAnswer(t, path)

	sendMalformed(t, path)
	awaitEntered(t, reports)
	cancel()

	waitUntil(t, func() bool {
		_, err := os.Lstat(path)
		return errors.Is(err, fs.ErrNotExist)
	})
	select {
	case err := <-done:
		t.Fatalf("Serve returned %v while the handler was still blocked", err)
	case <-time.After(200 * time.Millisecond):
	}
	releaseOnce()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Serve after cancel: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Serve did not return once the handler was released")
	}
}
