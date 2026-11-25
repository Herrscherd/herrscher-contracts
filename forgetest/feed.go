package forgetest

import (
	"context"
	"errors"
	"testing"
	"time"

	contracts "github.com/Herrscherd/herrscher-contracts"
)

// CheckPullRequestFeed opens the event feed of ref on feed and fails the test
// unless its channel closes once ctx is cancelled, and no send blocks past that
// cancellation. A feed that refuses at open must return an error matching
// [contracts.ErrFeedRefused], which passes the check. Each signal received after
// the feed opens must be a valid kind, or carry an Err with a zero Kind, and an
// Err must be the last signal before the close. The check cannot make a feed
// fail mid-stream, so a plugin proves that path with its own fake. A channel
// still open ten seconds after the cancellation fails the check.
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
