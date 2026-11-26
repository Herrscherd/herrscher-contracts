package forgetest

import (
	"context"
	"errors"
	"testing"
	"time"

	contracts "github.com/Herrscherd/herrscher-contracts"
)

// CheckPullRequestFeed opens the event feed of ref, cancels it, and fails the
// test unless the channel closes within ten seconds of the cancellation. A feed
// that refuses at open with an error matching [contracts.ErrFeedRefused] passes.
// Each signal received after the feed opens must name a valid kind or carry an
// Err with a zero Kind, and an Err must be the last signal before the close.
// A valid signal sent after the cancellation is accepted, since the check cannot
// tell it from one sent just before the cancellation. A feed that keeps the
// channel open past the deadline fails, but one that sends a bounded number of
// signals after the cancellation and then closes passes. The check cannot make a
// feed fail mid-stream, so a plugin proves that path with its own fake.
// [contracts.PullRequestFeed] obliges a plugin to send no new signal once it
// has observed the cancellation and never to block a send past ctx; the check
// does not prove either.
func CheckPullRequestFeed(tb testing.TB, feed contracts.PullRequestFeed, ref contracts.PullRequestRef) {
	tb.Helper()
	checkPullRequestFeed(tb, feed, ref, callTimeout)
}

func checkPullRequestFeed(report reporter, feed contracts.PullRequestFeed, ref contracts.PullRequestRef, timeout time.Duration) {
	report.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	signals, err := feed.PullRequestEvents(ctx, ref)
	if err != nil {
		if !errors.Is(err, contracts.ErrFeedRefused) {
			report.Errorf("PullRequestEvents: error = %v, want a channel or ErrFeedRefused", err)
		}
		return
	}
	if signals == nil {
		report.Errorf("PullRequestEvents returned a nil channel, want a channel or ErrFeedRefused")
		return
	}
	cancel()
	drainCancelledFeed(report, signals, timeout)
}

func drainCancelledFeed(report reporter, signals <-chan contracts.PullRequestSignal, timeout time.Duration) {
	report.Helper()
	deadline := time.NewTimer(timeout)
	defer deadline.Stop()
	failed := false
	for {
		select {
		case signal, open := <-signals:
			if !open {
				return
			}
			if failed {
				report.Errorf("PullRequestEvents sent a signal after the one with an error, want the channel closed")
				return
			}
			failed = checkFeedSignal(report, signal)
		case <-deadline.C:
			report.Errorf("PullRequestEvents channel still open %v after ctx was cancelled, want it closed", timeout)
			return
		}
	}
}

func checkFeedSignal(report reporter, signal contracts.PullRequestSignal) bool {
	if signal.Err != nil {
		if signal.Kind != 0 {
			report.Errorf("PullRequestEvents signal with Err %v has Kind %v, want a zero Kind", signal.Err, signal.Kind)
		}
		return true
	}
	if _, err := contracts.ParsePullRequestEventKind(signal.Kind.String()); err != nil {
		report.Errorf("PullRequestEvents signal Kind %v names no pull request event kind", signal.Kind)
	}
	return false
}
