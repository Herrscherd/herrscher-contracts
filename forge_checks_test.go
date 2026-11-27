package contracts

import (
	"testing"
)

func completedCheck(conclusion CheckConclusion) Check {
	return Check{Status: CheckCompleted, Conclusion: conclusion}
}

func TestSummarizeChecksMapsEachConclusion(t *testing.T) {
	tests := []struct {
		name       string
		conclusion CheckConclusion
		want       ChecksState
	}{
		{"success passes", ConclusionSuccess, ChecksPassed},
		{"neutral passes", ConclusionNeutral, ChecksPassed},
		{"skipped passes", ConclusionSkipped, ChecksPassed},
		{"failure fails", ConclusionFailure, ChecksFailed},
		{"cancelled fails", ConclusionCancelled, ChecksFailed},
		{"timed out fails", ConclusionTimedOut, ChecksFailed},
		{"action required waits for action", ConclusionActionRequired, ChecksActionRequired},
		{"other fails", ConclusionOther, ChecksFailed},
		{"completed without conclusion fails", ConclusionNone, ChecksFailed},
		{"unnamed conclusion fails", CheckConclusion(42), ChecksFailed},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SummarizeChecks([]Check{completedCheck(tt.conclusion)}); got != tt.want {
				t.Fatalf("SummarizeChecks(%v) = %v, want %v", tt.conclusion, got, tt.want)
			}
		})
	}
}

func TestSummarizeChecksRanksFailedOverActionRequiredOverPendingOverPassed(t *testing.T) {
	queued := Check{Status: CheckQueued}
	running := Check{Status: CheckInProgress}
	passed := completedCheck(ConclusionSuccess)
	failed := completedCheck(ConclusionFailure)
	actionRequired := completedCheck(ConclusionActionRequired)
	tests := []struct {
		name   string
		checks []Check
		want   ChecksState
	}{
		{"no check is none", nil, ChecksNone},
		{"empty list is none", []Check{}, ChecksNone},
		{"all passed is passed", []Check{passed, passed}, ChecksPassed},
		{"queued is pending", []Check{queued}, ChecksPending},
		{"running is pending", []Check{running}, ChecksPending},
		{"pending outranks passed", []Check{passed, running}, ChecksPending},
		{"passed does not hide pending", []Check{running, passed}, ChecksPending},
		{"action required is action required", []Check{actionRequired}, ChecksActionRequired},
		{"action required outranks passed", []Check{passed, actionRequired}, ChecksActionRequired},
		{"action required outranks pending", []Check{running, actionRequired}, ChecksActionRequired},
		{"pending does not hide action required", []Check{actionRequired, running}, ChecksActionRequired},
		{"action required outranks pending and passed", []Check{passed, running, actionRequired}, ChecksActionRequired},
		{"failed outranks pending", []Check{running, failed}, ChecksFailed},
		{"failed outranks pending when listed first", []Check{failed, queued}, ChecksFailed},
		{"failed outranks passed", []Check{passed, failed}, ChecksFailed},
		{"failed outranks action required", []Check{actionRequired, failed}, ChecksFailed},
		{"failed outranks action required when listed first", []Check{failed, actionRequired}, ChecksFailed},
		{"failed outranks every other check", []Check{passed, running, failed, actionRequired, passed}, ChecksFailed},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SummarizeChecks(tt.checks); got != tt.want {
				t.Fatalf("SummarizeChecks = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSummarizeChecksNeverReturnsUnknown(t *testing.T) {
	if got := SummarizeChecks(nil); got == ChecksUnknown {
		t.Fatal("SummarizeChecks(nil) = unknown, want none")
	}
}

func namedCompletedCheck(name string, conclusion CheckConclusion) Check {
	check := completedCheck(conclusion)
	check.Name = name
	return check
}

func TestSummarizeRequiredChecksWithoutRequiredNamesIsSummarizeChecks(t *testing.T) {
	tests := []struct {
		name   string
		checks []Check
	}{
		{"no check", nil},
		{"passed", []Check{namedCompletedCheck("build", ConclusionSuccess)}},
		{"pending", []Check{{Name: "build", Status: CheckQueued}}},
		{"failed", []Check{namedCompletedCheck("build", ConclusionFailure)}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got, want := SummarizeRequiredChecks(tt.checks, nil), SummarizeChecks(tt.checks); got != want {
				t.Fatalf("SummarizeRequiredChecks(checks, nil) = %v, want SummarizeChecks = %v", got, want)
			}
		})
	}
}

func TestSummarizeRequiredChecksIsPendingWhileARequiredCheckIsMissing(t *testing.T) {
	passed := namedCompletedCheck("build", ConclusionSuccess)
	tests := []struct {
		name     string
		checks   []Check
		required []string
	}{
		{"no check yet for a required name", nil, []string{"lint"}},
		{"passed check does not stand for a missing required name", []Check{passed}, []string{"build", "lint"}},
		{"a check with another name does not stand for a required one", []Check{passed}, []string{"lint"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SummarizeRequiredChecks(tt.checks, tt.required); got != ChecksPending {
				t.Fatalf("SummarizeRequiredChecks = %v, want %v", got, ChecksPending)
			}
		})
	}
}

func TestSummarizeRequiredChecksPassesWhenEveryRequiredCheckPassed(t *testing.T) {
	checks := []Check{
		namedCompletedCheck("build", ConclusionSuccess),
		namedCompletedCheck("lint", ConclusionSkipped),
	}
	if got := SummarizeRequiredChecks(checks, []string{"build", "lint"}); got != ChecksPassed {
		t.Fatalf("SummarizeRequiredChecks = %v, want %v", got, ChecksPassed)
	}
}

func TestSummarizeRequiredChecksKeepsFailedAndActionRequiredOverAMissingCheck(t *testing.T) {
	failed := namedCompletedCheck("build", ConclusionFailure)
	actionRequired := namedCompletedCheck("deploy", ConclusionActionRequired)
	tests := []struct {
		name   string
		checks []Check
		want   ChecksState
	}{
		{"failed outranks a missing required check", []Check{failed}, ChecksFailed},
		{"action required outranks a missing required check", []Check{actionRequired}, ChecksActionRequired},
		{"failed outranks action required beside a missing one", []Check{actionRequired, failed}, ChecksFailed},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SummarizeRequiredChecks(tt.checks, []string{"build", "lint"}); got != tt.want {
				t.Fatalf("SummarizeRequiredChecks = %v, want %v", got, tt.want)
			}
		})
	}
}
