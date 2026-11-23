package contracts

import (
	"fmt"
	"testing"
)

func TestClosedTypeStringsNameEachValue(t *testing.T) {
	tests := []struct {
		name string
		got  fmt.Stringer
		want string
	}{
		{"token source saved", TokenSourceSaved, "saved"},
		{"token source app", TokenSourceApp, "app"},
		{"token source env", TokenSourceEnv, "env"},
		{"token source cli", TokenSourceCLI, "cli"},
		{"pull request open", PullRequestOpen, "open"},
		{"pull request closed", PullRequestClosed, "closed"},
		{"pull request merged", PullRequestMerged, "merged"},
		{"check queued", CheckQueued, "queued"},
		{"check in progress", CheckInProgress, "in_progress"},
		{"check completed", CheckCompleted, "completed"},
		{"conclusion none", ConclusionNone, "none"},
		{"conclusion success", ConclusionSuccess, "success"},
		{"conclusion failure", ConclusionFailure, "failure"},
		{"conclusion neutral", ConclusionNeutral, "neutral"},
		{"conclusion cancelled", ConclusionCancelled, "cancelled"},
		{"conclusion skipped", ConclusionSkipped, "skipped"},
		{"conclusion timed out", ConclusionTimedOut, "timed_out"},
		{"conclusion action required", ConclusionActionRequired, "action_required"},
		{"merge method merge", MergeMethodMerge, "merge"},
		{"merge method squash", MergeMethodSquash, "squash"},
		{"merge method rebase", MergeMethodRebase, "rebase"},
	}
	for _, tt := range tests {
		if got := tt.got.String(); got != tt.want {
			t.Errorf("%s: String() = %q, want %q", tt.name, got, tt.want)
		}
	}
}

func TestClosedTypeStringsPrintTheRawNumberOfAnUnknownValue(t *testing.T) {
	tests := []struct {
		got  fmt.Stringer
		want string
	}{
		{TokenSource(9), "TokenSource(9)"},
		{PullRequestState(9), "PullRequestState(9)"},
		{CheckStatus(9), "CheckStatus(9)"},
		{CheckConclusion(9), "CheckConclusion(9)"},
		{MergeMethod(9), "MergeMethod(9)"},
	}
	for _, tt := range tests {
		if got := tt.got.String(); got != tt.want {
			t.Errorf("String() = %q, want %q", got, tt.want)
		}
	}
}

func TestZeroCheckConclusionIsNoneNotAnUnknownValue(t *testing.T) {
	var conclusion CheckConclusion
	if conclusion != ConclusionNone || conclusion.String() != "none" {
		t.Fatalf("zero CheckConclusion = %v, want ConclusionNone", conclusion)
	}
}
