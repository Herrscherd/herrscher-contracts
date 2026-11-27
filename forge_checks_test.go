package contracts

import (
	"errors"
	"fmt"
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

func TestSummarizeRequiredChecksWithoutRequirementIsSummarizeChecks(t *testing.T) {
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
			got, err := SummarizeRequiredChecks(tt.checks, nil, nil)
			if err != nil || got != SummarizeChecks(tt.checks) {
				t.Fatalf("SummarizeRequiredChecks(checks, nil, nil) = %v, %v, want SummarizeChecks = %v", got, err, SummarizeChecks(tt.checks))
			}
		})
	}
}

func TestSummarizeRequiredChecksIgnoresAFailedCheckThatIsNotRequired(t *testing.T) {
	checks := []Check{
		namedCompletedCheck("build", ConclusionSuccess),
		namedCompletedCheck("lint", ConclusionFailure),
	}
	got, err := SummarizeRequiredChecks(checks, []RequiredCheck{{Name: "build"}}, nil)
	if err != nil || got != ChecksPassed {
		t.Fatalf("SummarizeRequiredChecks = %v, %v, want %v", got, err, ChecksPassed)
	}
}

func TestSummarizeRequiredChecksIsPendingWhileARequiredCheckIsMissing(t *testing.T) {
	passed := namedCompletedCheck("build", ConclusionSuccess)
	tests := []struct {
		name     string
		checks   []Check
		required []RequiredCheck
	}{
		{"no check yet for a required name", nil, []RequiredCheck{{Name: "lint"}}},
		{"passed check does not stand for a missing required name", []Check{passed}, []RequiredCheck{{Name: "build"}, {Name: "lint"}}},
		{"a check with another name does not stand for a required one", []Check{passed}, []RequiredCheck{{Name: "lint"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SummarizeRequiredChecks(tt.checks, tt.required, nil)
			if err != nil || got != ChecksPending {
				t.Fatalf("SummarizeRequiredChecks = %v, %v, want %v", got, err, ChecksPending)
			}
		})
	}
}

func TestSummarizeRequiredChecksPassesWhenEveryRequiredCheckPassed(t *testing.T) {
	checks := []Check{
		namedCompletedCheck("build", ConclusionSuccess),
		namedCompletedCheck("lint", ConclusionSkipped),
	}
	got, err := SummarizeRequiredChecks(checks, []RequiredCheck{{Name: "build"}, {Name: "lint"}}, nil)
	if err != nil || got != ChecksPassed {
		t.Fatalf("SummarizeRequiredChecks = %v, %v, want %v", got, err, ChecksPassed)
	}
}

func TestSummarizeRequiredChecksKeepsFailedAndActionRequiredOverAMissingCheck(t *testing.T) {
	failed := namedCompletedCheck("build", ConclusionFailure)
	actionRequired := namedCompletedCheck("lint", ConclusionActionRequired)
	tests := []struct {
		name   string
		checks []Check
		want   ChecksState
	}{
		{"failed outranks a missing required check", []Check{failed}, ChecksFailed},
		{"action required outranks a missing required check", []Check{actionRequired}, ChecksActionRequired},
		{"failed outranks action required beside a missing one", []Check{actionRequired, failed}, ChecksFailed},
	}
	required := []RequiredCheck{{Name: "build"}, {Name: "lint"}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SummarizeRequiredChecks(tt.checks, required, nil)
			if err != nil || got != tt.want {
				t.Fatalf("SummarizeRequiredChecks = %v, %v, want %v", got, err, tt.want)
			}
		})
	}
}

func attempt(check Check, number int64) Check {
	check.Attempt = number
	return check
}

func TestSummarizeChecksReadsTheLatestAttemptOfARerun(t *testing.T) {
	failed := namedCompletedCheck("build", ConclusionFailure)
	passed := namedCompletedCheck("build", ConclusionSuccess)
	queued := Check{Name: "build", Status: CheckQueued}
	tests := []struct {
		name   string
		checks []Check
		want   ChecksState
	}{
		{"a re-run that passes supersedes its failure", []Check{attempt(failed, 1), attempt(passed, 2)}, ChecksPassed},
		{"the latest attempt is read even when listed first", []Check{attempt(passed, 2), attempt(failed, 1)}, ChecksPassed},
		{"a latest failure outranks an earlier pass", []Check{attempt(passed, 1), attempt(failed, 2)}, ChecksFailed},
		{"a re-run in progress is pending", []Check{attempt(failed, 1), attempt(queued, 2)}, ChecksPending},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SummarizeRequiredChecks(tt.checks, []RequiredCheck{{Name: "build"}}, nil)
			if err != nil || got != tt.want {
				t.Fatalf("SummarizeRequiredChecks = %v, %v, want %v", got, err, tt.want)
			}
		})
	}
}

func TestSummarizeChecksKeepsSameNamedChecksOfTwoAppsApart(t *testing.T) {
	failedByOneApp := namedCompletedCheck("build", ConclusionFailure)
	failedByOneApp.AppID = 1
	passedByOtherApp := namedCompletedCheck("build", ConclusionSuccess)
	passedByOtherApp.AppID = 2
	if got := SummarizeChecks([]Check{failedByOneApp, passedByOtherApp}); got != ChecksFailed {
		t.Fatalf("SummarizeChecks = %v, want %v: a pass of another app must not replace a failure", got, ChecksFailed)
	}
}

func TestSummarizeRequiredChecksRefusesARequiredCheckFromAnotherSource(t *testing.T) {
	required := []RequiredCheck{{Name: "build", AppID: 1}}
	tests := []struct {
		name  string
		appID int64
		want  ChecksState
	}{
		{"a check of the same name from another app is pending", 2, ChecksPending},
		{"a check of the same name with no reported source is pending", 0, ChecksPending},
		{"a check of the same name from the required app passes", 1, ChecksPassed},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			check := namedCompletedCheck("build", ConclusionSuccess)
			check.AppID = tt.appID
			got, err := SummarizeRequiredChecks([]Check{check}, required, nil)
			if err != nil || got != tt.want {
				t.Fatalf("SummarizeRequiredChecks = %v, %v, want %v", got, err, tt.want)
			}
		})
	}
}

func TestSummarizeRequiredChecksAcceptsAnySourceWhenTheRequirementNamesNone(t *testing.T) {
	check := namedCompletedCheck("build", ConclusionSuccess)
	check.AppID = 7
	got, err := SummarizeRequiredChecks([]Check{check}, []RequiredCheck{{Name: "build"}}, nil)
	if err != nil || got != ChecksPassed {
		t.Fatalf("SummarizeRequiredChecks = %v, %v, want %v", got, err, ChecksPassed)
	}
}

func TestSummarizeRequiredChecksReadsAnUnknownRequirementAsPending(t *testing.T) {
	unknown := fmt.Errorf("read rulesets: %w", ErrRequiredChecksUnknown)
	passed := namedCompletedCheck("build", ConclusionSuccess)
	failed := namedCompletedCheck("build", ConclusionFailure)
	tests := []struct {
		name   string
		checks []Check
		want   ChecksState
	}{
		{"no check reads as pending", nil, ChecksPending},
		{"a passing check does not prove the requirements", []Check{passed}, ChecksPending},
		{"a failed check still outranks the unknown requirements", []Check{failed}, ChecksFailed},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SummarizeRequiredChecks(tt.checks, nil, unknown)
			if err != nil || got != tt.want {
				t.Fatalf("SummarizeRequiredChecks = %v, %v, want %v", got, err, tt.want)
			}
		})
	}
}

func TestSummarizeRequiredChecksReturnsAnotherReadErrorInsteadOfAState(t *testing.T) {
	readErr := errors.New("connection reset")
	got, err := SummarizeRequiredChecks([]Check{namedCompletedCheck("build", ConclusionSuccess)}, nil, readErr)
	if !errors.Is(err, readErr) || got != ChecksUnknown {
		t.Fatalf("SummarizeRequiredChecks = %v, %v, want %v and the read error", got, err, ChecksUnknown)
	}
}
