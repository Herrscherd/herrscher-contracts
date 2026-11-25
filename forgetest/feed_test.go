package forgetest

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	contracts "github.com/Herrscherd/herrscher-contracts"
)

type scriptedFeed struct {
	run func(ctx context.Context, signals chan<- contracts.PullRequestSignal)
}

func (f scriptedFeed) PullRequestEvents(ctx context.Context, _ contracts.PullRequestRef) (<-chan contracts.PullRequestSignal, error) {
	signals := make(chan contracts.PullRequestSignal)
	go func() {
		defer close(signals)
		f.run(ctx, signals)
	}()
	return signals, nil
}

type refusingFeed struct{}

func (refusingFeed) PullRequestEvents(context.Context, contracts.PullRequestRef) (<-chan contracts.PullRequestSignal, error) {
	return nil, fmt.Errorf("feed refused: %w", contracts.ErrFeedRefused)
}

type brokenOpenFeed struct{}

func (brokenOpenFeed) PullRequestEvents(context.Context, contracts.PullRequestRef) (<-chan contracts.PullRequestSignal, error) {
	return nil, errors.New("connection reset")
}

type nilChannelFeed struct{}

func (nilChannelFeed) PullRequestEvents(context.Context, contracts.PullRequestRef) (<-chan contracts.PullRequestSignal, error) {
	var none <-chan contracts.PullRequestSignal
	return none, nil
}

type openForeverFeed struct{}

func (openForeverFeed) PullRequestEvents(context.Context, contracts.PullRequestRef) (<-chan contracts.PullRequestSignal, error) {
	return make(chan contracts.PullRequestSignal), nil
}

func checkScripted(run func(ctx context.Context, signals chan<- contracts.PullRequestSignal)) *recorder {
	rec := &recorder{}
	checkPullRequestFeed(rec, scriptedFeed{run: run}, contracts.PullRequestRef{}, callTimeout)
	return rec
}

func expectFailures(t *testing.T, rec *recorder, want int) {
	t.Helper()
	if len(rec.failures) != want {
		t.Fatalf("reported %d failures, want %d: %q", len(rec.failures), want, rec.failures)
	}
}

func TestCheckPullRequestFeedPassesAFeedThatClosesOnCancel(t *testing.T) {
	rec := checkScripted(func(ctx context.Context, _ chan<- contracts.PullRequestSignal) {
		<-ctx.Done()
	})
	expectFailures(t, rec, 0)
}

func TestCheckPullRequestFeedPassesARefusalAtOpen(t *testing.T) {
	rec := &recorder{}
	checkPullRequestFeed(rec, refusingFeed{}, contracts.PullRequestRef{}, callTimeout)
	expectFailures(t, rec, 0)
}

func TestCheckPullRequestFeedFailsAnOpenErrorThatIsNotARefusal(t *testing.T) {
	rec := &recorder{}
	checkPullRequestFeed(rec, brokenOpenFeed{}, contracts.PullRequestRef{}, callTimeout)
	expectFailures(t, rec, 1)
}

func TestCheckPullRequestFeedFailsANilChannel(t *testing.T) {
	rec := &recorder{}
	checkPullRequestFeed(rec, nilChannelFeed{}, contracts.PullRequestRef{}, callTimeout)
	expectFailures(t, rec, 1)
}

func TestCheckPullRequestFeedFailsAChannelOpenAfterCancel(t *testing.T) {
	rec := &recorder{}
	checkPullRequestFeed(rec, openForeverFeed{}, contracts.PullRequestRef{}, 10*time.Millisecond)
	expectFailures(t, rec, 1)
}

func TestCheckPullRequestFeedPassesAnErrorSignalBeforeClose(t *testing.T) {
	rec := checkScripted(func(_ context.Context, signals chan<- contracts.PullRequestSignal) {
		signals <- contracts.PullRequestSignal{Err: fmt.Errorf("stream dropped: %w", contracts.ErrFeedRefused)}
	})
	expectFailures(t, rec, 0)
}

func TestCheckPullRequestFeedFailsASignalAfterAnErrorSignal(t *testing.T) {
	rec := checkScripted(func(ctx context.Context, signals chan<- contracts.PullRequestSignal) {
		<-ctx.Done()
		signals <- contracts.PullRequestSignal{Err: contracts.ErrFeedRefused}
		signals <- contracts.PullRequestSignal{Kind: contracts.PullRequestNewCommit}
	})
	expectFailures(t, rec, 1)
}

func TestCheckPullRequestFeedFailsAnErrorSignalWithAKind(t *testing.T) {
	rec := checkScripted(func(ctx context.Context, signals chan<- contracts.PullRequestSignal) {
		<-ctx.Done()
		signals <- contracts.PullRequestSignal{Kind: contracts.PullRequestNewCommit, Err: contracts.ErrFeedRefused}
	})
	expectFailures(t, rec, 1)
}

func TestCheckPullRequestFeedFailsAZeroKindSignal(t *testing.T) {
	rec := checkScripted(func(ctx context.Context, signals chan<- contracts.PullRequestSignal) {
		<-ctx.Done()
		signals <- contracts.PullRequestSignal{}
	})
	expectFailures(t, rec, 1)
}
