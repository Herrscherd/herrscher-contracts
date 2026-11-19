package approvalsock

import (
	"errors"
	"testing"
	"time"
)

func awaitClosed(t *testing.T, ch <-chan struct{}, message string) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(5 * time.Second):
		t.Fatal(message)
	}
}

func TestReporterCountsOverflowWhileHandlerIsBusy(t *testing.T) {
	t.Parallel()
	reporter := newReporter()
	first := errors.New("first failure")
	entered := make(chan struct{})
	release := make(chan struct{})
	delivered := make(chan struct{})
	var received []error
	go func() {
		defer close(delivered)
		reporter.deliver(func(err error) {
			received = append(received, err)
			if len(received) == 1 {
				close(entered)
				<-release
			}
		})
	}()

	reporter.send(first)
	awaitClosed(t, entered, "handler never received the first failure")

	const overflow = 8
	sent := make(chan struct{})
	go func() {
		defer close(sent)
		for range reportQueueSize + overflow {
			reporter.send(errors.New("queued failure"))
		}
	}()
	awaitClosed(t, sent, "send blocked while the handler was busy")

	if got := reporter.dropped.Load(); got != overflow {
		t.Fatalf("dropped = %d while the handler is busy, want %d", got, overflow)
	}
	if got := len(reporter.queue); got != reportQueueSize {
		t.Fatalf("queued = %d while the handler is busy, want %d", got, reportQueueSize)
	}

	close(release)
	close(reporter.queue)
	awaitClosed(t, delivered, "deliver did not return once the queue was closed")

	if len(received) != reportQueueSize+2 {
		t.Fatalf("handler received %d reports, want %d", len(received), reportQueueSize+2)
	}
	if !errors.Is(received[0], first) {
		t.Fatalf("first report = %v, want the first failure", received[0])
	}
	dropped, ok := errors.AsType[*ReportsDroppedError](received[1])
	if !ok || dropped.Count != overflow {
		t.Fatalf("second report = %v, want ReportsDroppedError with Count %d", received[1], overflow)
	}
	var total uint64
	for _, err := range received {
		if count, ok := errors.AsType[*ReportsDroppedError](err); ok {
			total += count.Count
			continue
		}
		total++
	}
	if total != 1+reportQueueSize+overflow {
		t.Fatalf("reported failures = %d, want %d", total, 1+reportQueueSize+overflow)
	}
}
