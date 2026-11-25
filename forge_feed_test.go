package contracts

import (
	"context"
	"testing"
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

func TestZeroPullRequestReadsAsUnknownChecksAndNoReviews(t *testing.T) {
	var pull PullRequest
	if pull.Checks != ChecksUnknown || pull.LatestReview != "" || pull.Merge != MergeStateUnknown {
		t.Fatalf("zero PullRequest = checks %v, latest review %q, merge %v, want unknown, empty, unknown", pull.Checks, pull.LatestReview, pull.Merge)
	}
}
