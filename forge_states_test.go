package contracts

import (
	"errors"
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
		{"conclusion other", ConclusionOther, "other"},
		{"merge method merge", MergeMethodMerge, "merge"},
		{"merge method squash", MergeMethodSquash, "squash"},
		{"merge method rebase", MergeMethodRebase, "rebase"},
		{"merge state unknown", MergeStateUnknown, "unknown"},
		{"merge state clean", MergeStateClean, "clean"},
		{"merge state conflicting", MergeStateConflicting, "conflicting"},
		{"merge state blocked", MergeStateBlocked, "blocked"},
		{"merge state behind", MergeStateBehind, "behind"},
		{"merge state draft", MergeStateDraft, "draft"},
		{"checks unknown", ChecksUnknown, "unknown"},
		{"checks none", ChecksNone, "none"},
		{"checks pending", ChecksPending, "pending"},
		{"checks passed", ChecksPassed, "passed"},
		{"checks failed", ChecksFailed, "failed"},
		{"checks action required", ChecksActionRequired, "action_required"},
		{"event review submitted", PullRequestReviewSubmitted, "review_submitted"},
		{"event new commit", PullRequestNewCommit, "new_commit"},
		{"event state changed", PullRequestStateChanged, "state_changed"},
		{"event merge conflict", PullRequestMergeConflict, "merge_conflict"},
		{"event checks changed", PullRequestChecksChanged, "checks_changed"},
		{"association none", AssociationNone, "none"},
		{"association owner", AssociationOwner, "owner"},
		{"association member", AssociationMember, "member"},
		{"association collaborator", AssociationCollaborator, "collaborator"},
		{"association contributor", AssociationContributor, "contributor"},
		{"association first timer", AssociationFirstTimer, "first_timer"},
		{"association other", AssociationOther, "other"},
		{"issue state unknown", IssueStateUnknown, "unknown"},
		{"issue open", IssueOpen, "open"},
		{"issue closed", IssueClosed, "closed"},
		{"permission none", PermissionNone, "none"},
		{"permission read", PermissionRead, "read"},
		{"permission triage", PermissionTriage, "triage"},
		{"permission write", PermissionWrite, "write"},
		{"permission maintain", PermissionMaintain, "maintain"},
		{"permission admin", PermissionAdmin, "admin"},
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
		{PullRequestState(9), "PullRequestState(9)"},
		{CheckStatus(9), "CheckStatus(9)"},
		{CheckConclusion(9), "CheckConclusion(9)"},
		{MergeMethod(9), "MergeMethod(9)"},
		{MergeState(9), "MergeState(9)"},
		{ChecksState(9), "ChecksState(9)"},
		{PullRequestEventKind(9), "PullRequestEventKind(9)"},
		{AuthorAssociation(9), "AuthorAssociation(9)"},
		{IssueState(9), "IssueState(9)"},
		{RepoPermission(9), "RepoPermission(9)"},
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

func TestZeroIssueIsUnknownAndNotOpen(t *testing.T) {
	var issue Issue
	if issue.State != IssueStateUnknown {
		t.Fatalf("zero Issue state = %v, want IssueStateUnknown", issue.State)
	}
	if issue.State == IssueOpen {
		t.Fatalf("zero Issue reads as open, want unknown, which consumers treat as not open")
	}
}

func TestIssueQueryListsOpenIssuesByDefault(t *testing.T) {
	if listed := (IssueQuery{}).Listed(); listed != IssueOpen {
		t.Fatalf("zero IssueQuery lists %v, want IssueOpen", listed)
	}
	if listed := (IssueQuery{State: IssueOpen}).Listed(); listed != IssueOpen {
		t.Fatalf("IssueQuery{State: IssueOpen} lists %v, want IssueOpen", listed)
	}
	if listed := (IssueQuery{State: IssueClosed}).Listed(); listed != IssueClosed {
		t.Fatalf("IssueQuery{State: IssueClosed} lists %v, want IssueClosed", listed)
	}
}

func TestCheckConcludedReturnsTheConclusionOnlyWhenCompleted(t *testing.T) {
	tests := []struct {
		name       string
		check      Check
		want       CheckConclusion
		wantFinish bool
	}{
		{"queued", Check{Status: CheckQueued, Conclusion: ConclusionFailure}, ConclusionNone, false},
		{"in progress", Check{Status: CheckInProgress}, ConclusionNone, false},
		{"completed success", Check{Status: CheckCompleted, Conclusion: ConclusionSuccess}, ConclusionSuccess, true},
		{"completed unmapped", Check{Status: CheckCompleted, Conclusion: ConclusionOther}, ConclusionOther, true},
		{"zero check", Check{}, ConclusionNone, false},
	}
	for _, tt := range tests {
		got, finished := tt.check.Concluded()
		if got != tt.want || finished != tt.wantFinish {
			t.Errorf("%s: Concluded() = %v, %v, want %v, %v", tt.name, got, finished, tt.want, tt.wantFinish)
		}
	}
}

func TestConclusionOtherIsNotNone(t *testing.T) {
	if ConclusionOther == ConclusionNone {
		t.Fatalf("ConclusionOther equals ConclusionNone, want a distinct conclusion")
	}
}

func TestZeroMergeStateAndIssueStateAreTheUnmarkedValues(t *testing.T) {
	var merge MergeState
	if merge != MergeStateUnknown || merge.String() != "unknown" {
		t.Fatalf("zero MergeState = %v, want MergeStateUnknown", merge)
	}
	var issue IssueState
	if issue != IssueStateUnknown || issue.String() != "unknown" {
		t.Fatalf("zero IssueState = %v, want IssueStateUnknown", issue)
	}
	var association AuthorAssociation
	if association != AssociationNone || association.String() != "none" {
		t.Fatalf("zero AuthorAssociation = %v, want AssociationNone", association)
	}
	var permission RepoPermission
	if permission != PermissionNone || permission.String() != "none" {
		t.Fatalf("zero RepoPermission = %v, want PermissionNone", permission)
	}
}

func TestParsePullRequestStateReadsEveryStateBack(t *testing.T) {
	for _, want := range PullRequestStates() {
		got, err := ParsePullRequestState(want.String())
		if err != nil || got != want {
			t.Fatalf("ParsePullRequestState(%q) = %v, %v, want %v", want.String(), got, err, want)
		}
	}
}

func TestParsePullRequestStateRefusesUnknownText(t *testing.T) {
	for _, text := range []string{"", "Open", "draft", "PullRequestState(0)"} {
		got, err := ParsePullRequestState(text)
		if !errors.Is(err, ErrUnknownPullRequestState) {
			t.Errorf("ParsePullRequestState(%q) error = %v, want ErrUnknownPullRequestState", text, err)
		}
		if got != 0 {
			t.Errorf("ParsePullRequestState(%q) = %v, want the zero value on error", text, got)
		}
	}
}

func TestPullRequestStateRefusesToMarshalAnUnknownValue(t *testing.T) {
	for _, state := range []PullRequestState{0, 9} {
		if _, err := state.MarshalText(); !errors.Is(err, ErrUnknownPullRequestState) {
			t.Errorf("MarshalText(%d) error = %v, want ErrUnknownPullRequestState", int(state), err)
		}
	}
}

func TestParseChecksStateReadsEveryStateBack(t *testing.T) {
	for _, want := range checksStates() {
		got, err := ParseChecksState(want.String())
		if err != nil || got != want {
			t.Fatalf("ParseChecksState(%q) = %v, %v, want %v", want.String(), got, err, want)
		}
	}
}

func TestParseChecksStateReadsEveryDeclaredStateBack(t *testing.T) {
	tests := []struct {
		text string
		want ChecksState
	}{
		{"unknown", ChecksUnknown},
		{"none", ChecksNone},
		{"pending", ChecksPending},
		{"passed", ChecksPassed},
		{"failed", ChecksFailed},
		{"action_required", ChecksActionRequired},
	}
	if got, want := len(checksStates()), len(tests); got != want {
		t.Fatalf("checksStates() lists %d states but this table names %d: add the new state to this table", got, want)
	}
	for _, tt := range tests {
		got, err := ParseChecksState(tt.text)
		if err != nil || got != tt.want {
			t.Errorf("ParseChecksState(%q) = %v, %v, want %v", tt.text, got, err, tt.want)
		}
	}
}

func TestParsePullRequestStateReadsEveryDeclaredStateBack(t *testing.T) {
	tests := []struct {
		text string
		want PullRequestState
	}{
		{"open", PullRequestOpen},
		{"closed", PullRequestClosed},
		{"merged", PullRequestMerged},
	}
	if got, want := len(PullRequestStates()), len(tests); got != want {
		t.Fatalf("PullRequestStates() lists %d states but this table names %d: add the new state to this table", got, want)
	}
	for _, tt := range tests {
		got, err := ParsePullRequestState(tt.text)
		if err != nil || got != tt.want {
			t.Errorf("ParsePullRequestState(%q) = %v, %v, want %v", tt.text, got, err, tt.want)
		}
	}
}

func TestParsePullRequestEventKindReadsEveryDeclaredKindBack(t *testing.T) {
	tests := []struct {
		text string
		want PullRequestEventKind
	}{
		{"review_submitted", PullRequestReviewSubmitted},
		{"new_commit", PullRequestNewCommit},
		{"state_changed", PullRequestStateChanged},
		{"merge_conflict", PullRequestMergeConflict},
		{"checks_changed", PullRequestChecksChanged},
	}
	if got, want := len(pullRequestEventKinds()), len(tests); got != want {
		t.Fatalf("pullRequestEventKinds() lists %d kinds but this table names %d: add the new kind to this table", got, want)
	}
	for _, tt := range tests {
		got, err := ParsePullRequestEventKind(tt.text)
		if err != nil || got != tt.want {
			t.Errorf("ParsePullRequestEventKind(%q) = %v, %v, want %v", tt.text, got, err, tt.want)
		}
	}
}

func TestParseChecksStateRefusesUnknownText(t *testing.T) {
	for _, text := range []string{"", "Passed", "success", "action-required", "ChecksState(0)"} {
		got, err := ParseChecksState(text)
		if !errors.Is(err, ErrUnknownChecksState) {
			t.Errorf("ParseChecksState(%q) error = %v, want ErrUnknownChecksState", text, err)
		}
		if got != 0 {
			t.Errorf("ParseChecksState(%q) = %v, want the zero value on error", text, got)
		}
	}
}

func TestZeroChecksStateIsUnknown(t *testing.T) {
	var state ChecksState
	if state != ChecksUnknown || state.String() != "unknown" {
		t.Fatalf("zero ChecksState = %v, want ChecksUnknown", state)
	}
}

func TestParsePullRequestEventKindReadsEveryKindBack(t *testing.T) {
	for _, want := range pullRequestEventKinds() {
		got, err := ParsePullRequestEventKind(want.String())
		if err != nil || got != want {
			t.Fatalf("ParsePullRequestEventKind(%q) = %v, %v, want %v", want.String(), got, err, want)
		}
	}
}

func TestParsePullRequestEventKindRefusesUnknownText(t *testing.T) {
	for _, text := range []string{"", "Review_Submitted", "review", "PullRequestEventKind(0)"} {
		got, err := ParsePullRequestEventKind(text)
		if !errors.Is(err, ErrUnknownPullRequestEventKind) {
			t.Errorf("ParsePullRequestEventKind(%q) error = %v, want ErrUnknownPullRequestEventKind", text, err)
		}
		if got != 0 {
			t.Errorf("ParsePullRequestEventKind(%q) = %v, want the zero value on error", text, got)
		}
	}
}

func TestZeroPullRequestEventKindIsNoKind(t *testing.T) {
	var kind PullRequestEventKind
	if _, err := ParsePullRequestEventKind(kind.String()); !errors.Is(err, ErrUnknownPullRequestEventKind) {
		t.Fatalf("zero PullRequestEventKind parses as a kind: error = %v", err)
	}
}
