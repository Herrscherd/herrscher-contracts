package contracts

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

type feedlessForge struct {
	Forge
}

type feedForge struct {
	feedlessForge
}

func (feedForge) PullRequestEvents(context.Context, PullRequestRef) (<-chan PullRequestSignal, error) {
	return nil, ErrFeedRefused
}

type streamFailingFeed struct {
	feedlessForge
}

func (streamFailingFeed) PullRequestEvents(context.Context, PullRequestRef) (<-chan PullRequestSignal, error) {
	signals := make(chan PullRequestSignal, 2)
	signals <- PullRequestSignal{Kind: PullRequestNewCommit}
	signals <- PullRequestSignal{Err: fmt.Errorf("event stream dropped: %w", ErrFeedRefused)}
	close(signals)
	return signals, nil
}

type floodingFeed struct {
	feedlessForge
}

func (floodingFeed) PullRequestEvents(ctx context.Context, _ PullRequestRef) (<-chan PullRequestSignal, error) {
	signals := make(chan PullRequestSignal)
	go func() {
		defer close(signals)
		for {
			select {
			case signals <- PullRequestSignal{Kind: PullRequestChecksChanged}:
			case <-ctx.Done():
				return
			}
		}
	}()
	return signals, nil
}

func receiveSignal(t *testing.T, signals <-chan PullRequestSignal) (PullRequestSignal, bool) {
	t.Helper()
	select {
	case signal, open := <-signals:
		return signal, open
	case <-time.After(5 * time.Second):
		t.Fatal("feed neither sent a signal nor closed within 5s")
		return PullRequestSignal{}, false
	}
}

func TestPullRequestFeedIsFoundOnlyWhenImplemented(t *testing.T) {
	var plain any = feedlessForge{}
	if _, ok := plain.(PullRequestFeed); ok {
		t.Fatal("a forge without PullRequestEvents satisfies PullRequestFeed")
	}
	var streaming any = feedForge{}
	if _, ok := streaming.(PullRequestFeed); !ok {
		t.Fatal("a forge with PullRequestEvents does not satisfy PullRequestFeed")
	}
}

func TestFeedRefusedAtOpenReturnsNoChannel(t *testing.T) {
	signals, err := feedForge{}.PullRequestEvents(context.Background(), PullRequestRef{})
	if !errors.Is(err, ErrFeedRefused) {
		t.Fatalf("PullRequestEvents error = %v, want ErrFeedRefused", err)
	}
	if signals != nil {
		t.Fatal("a refused feed returned a channel")
	}
}

func TestFeedStreamFailureSendsErrorSignalBeforeClose(t *testing.T) {
	signals, err := streamFailingFeed{}.PullRequestEvents(context.Background(), PullRequestRef{})
	if err != nil {
		t.Fatalf("PullRequestEvents error = %v, want a channel", err)
	}
	first, open := receiveSignal(t, signals)
	if !open || first.Kind != PullRequestNewCommit || first.Err != nil {
		t.Fatalf("first signal = %+v open %v, want the new commit kind", first, open)
	}
	failure, open := receiveSignal(t, signals)
	if !open {
		t.Fatal("channel closed before the error signal, want the error signal first")
	}
	if !errors.Is(failure.Err, ErrFeedRefused) || failure.Kind != 0 {
		t.Fatalf("error signal = %+v, want Err matching ErrFeedRefused and a zero Kind", failure)
	}
	if _, open := receiveSignal(t, signals); open {
		t.Fatal("channel still open after the error signal, want it closed")
	}
}

func TestFeedClosesOnCancelWhileSignalsFlow(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	signals, err := floodingFeed{}.PullRequestEvents(ctx, PullRequestRef{})
	if err != nil {
		t.Fatalf("PullRequestEvents error = %v, want a channel", err)
	}
	if first, open := receiveSignal(t, signals); !open || first.Kind != PullRequestChecksChanged {
		t.Fatalf("first signal = %+v open %v, want the checks kind", first, open)
	}
	cancel()
	for {
		signal, open := receiveSignal(t, signals)
		if !open {
			return
		}
		if signal.Err != nil {
			t.Fatalf("signal after cancel carries %v, want no error", signal.Err)
		}
	}
}

func TestZeroPullRequestReadsAsUnknownChecksAndNoReviews(t *testing.T) {
	var pull PullRequest
	if pull.Checks != ChecksUnknown || pull.LatestReview != "" || pull.Merge != MergeStateUnknown {
		t.Fatalf("zero PullRequest = checks %v, latest review %q, merge %v, want unknown, empty, unknown", pull.Checks, pull.LatestReview, pull.Merge)
	}
}
