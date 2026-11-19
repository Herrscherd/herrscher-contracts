package approvalsock_test

import (
	"context"
	"errors"
	"net"
	"path/filepath"
	"sync"
	"testing"
	"time"

	contracts "github.com/Herrscherd/herrscher-contracts"
	"github.com/Herrscherd/herrscher-contracts/approvalsock"
)

const malformedFailures = 41

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

func startReportingServer(t *testing.T, opt approvalsock.Option) string {
	t.Helper()
	path := filepath.Join(privateDir(t), "a.sock")
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() { done <- approvalsock.Serve(ctx, path, allowAll, opt) }()
	t.Cleanup(func() {
		cancel()
		if err := <-done; err != nil {
			t.Errorf("Serve returned %v after cancel", err)
		}
	})
	awaitAnswer(t, path)
	return path
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

func TestServeBlockedReportNeverStallsAcceptOrAnswer(t *testing.T) {
	t.Parallel()
	reports := newBlockingReports()
	releaseOnce := sync.OnceFunc(func() { close(reports.release) })
	path := startReportingServer(t, approvalsock.WithErrorHandler(reports.report))
	t.Cleanup(releaseOnce)

	sendMalformed(t, path)
	awaitEntered(t, reports)

	legitimate := func() {
		ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
		defer cancel()
		answer, err := approvalsock.Ask(ctx, path, contracts.ApprovalRequest{ID: "legit", Tool: "Bash", Subject: "ls"})
		if err != nil {
			t.Fatalf("Ask while the handler is blocked: %v", err)
		}
		if !answer.Allowed() {
			t.Fatalf("answer = %+v, want allow", answer)
		}
	}
	legitimate()

	started := time.Now()
	for range malformedFailures - 1 {
		sendMalformed(t, path)
	}
	if elapsed := time.Since(started); elapsed > 5*time.Second {
		t.Fatalf("%d failures took %v with a blocked handler", malformedFailures-1, elapsed)
	}
	legitimate()

	if got := reports.count(); got != 1 {
		t.Fatalf("handler saw %d reports while blocked, want only the first", got)
	}
}

func TestServeReportsQueueOverflowOnceHandlerIsFree(t *testing.T) {
	t.Parallel()
	reports := newBlockingReports()
	releaseOnce := sync.OnceFunc(func() { close(reports.release) })
	path := startReportingServer(t, approvalsock.WithErrorHandler(reports.report))
	t.Cleanup(releaseOnce)

	sendMalformed(t, path)
	awaitEntered(t, reports)
	for range malformedFailures - 1 {
		sendMalformed(t, path)
	}
	releaseOnce()

	waitUntil(t, func() bool { return reports.total() == malformedFailures })
	reports.mu.Lock()
	defer reports.mu.Unlock()
	var sawDropped bool
	for _, err := range reports.received {
		if dropped, ok := errors.AsType[*approvalsock.ReportsDroppedError](err); ok && errors.Is(dropped, approvalsock.ErrReportsDropped) {
			sawDropped = true
		}
	}
	if !sawDropped {
		t.Fatal("no ReportsDroppedError reached the handler after the overflow")
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
