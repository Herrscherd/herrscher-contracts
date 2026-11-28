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

func TestSummarizeRequiredChecksWithOneRunPerCheckMatchesSummarizeChecks(t *testing.T) {
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
			got := summarizeRequired(tt.checks, nil)
			if got != SummarizeChecks(tt.checks) {
				t.Fatalf("summarizeRequired(checks, nil) = %v, want SummarizeChecks = %v", got, SummarizeChecks(tt.checks))
			}
		})
	}
}

func TestSummarizeRequiredChecksIgnoresAFailedCheckThatIsNotRequired(t *testing.T) {
	checks := []Check{
		namedCompletedCheck("build", ConclusionSuccess),
		namedCompletedCheck("lint", ConclusionFailure),
	}
	got := summarizeRequired(checks, []RequiredCheck{requiredCheck(t, "build", 0)})
	if got != ChecksPassed {
		t.Fatalf("SummarizeRequiredChecks = %v, want %v", got, ChecksPassed)
	}
}

func TestSummarizeRequiredChecksIsPendingWhileARequiredCheckIsMissing(t *testing.T) {
	passed := namedCompletedCheck("build", ConclusionSuccess)
	tests := []struct {
		name     string
		checks   []Check
		required []RequiredCheck
	}{
		{"no check yet for a required name", nil, []RequiredCheck{requiredCheck(t, "lint", 0)}},
		{"passed check does not stand for a missing required name", []Check{passed}, []RequiredCheck{requiredCheck(t, "build", 0), requiredCheck(t, "lint", 0)}},
		{"a check with another name does not stand for a required one", []Check{passed}, []RequiredCheck{requiredCheck(t, "lint", 0)}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := summarizeRequired(tt.checks, tt.required)
			if got != ChecksPending {
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
	got := summarizeRequired(checks, []RequiredCheck{requiredCheck(t, "build", 0), requiredCheck(t, "lint", 0)})
	if got != ChecksPassed {
		t.Fatalf("SummarizeRequiredChecks = %v, want %v", got, ChecksPassed)
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
	required := []RequiredCheck{requiredCheck(t, "build", 0), requiredCheck(t, "lint", 0)}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := summarizeRequired(tt.checks, required)
			if got != tt.want {
				t.Fatalf("SummarizeRequiredChecks = %v, want %v", got, tt.want)
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
			got := summarizeRequired(tt.checks, []RequiredCheck{requiredCheck(t, "build", 0)})
			if got != tt.want {
				t.Fatalf("SummarizeRequiredChecks = %v, want %v", got, tt.want)
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
	required := []RequiredCheck{requiredCheck(t, "build", 1)}
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
			got := summarizeRequired([]Check{check}, required)
			if got != tt.want {
				t.Fatalf("SummarizeRequiredChecks = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSummarizeRequiredChecksAcceptsAnySourceWhenTheRequirementNamesNone(t *testing.T) {
	check := namedCompletedCheck("build", ConclusionSuccess)
	check.AppID = 7
	got := summarizeRequired([]Check{check}, []RequiredCheck{requiredCheck(t, "build", 0)})
	if got != ChecksPassed {
		t.Fatalf("SummarizeRequiredChecks = %v, want %v", got, ChecksPassed)
	}
}

func TestSummarizeRequiredChecksReadsAnUnknownRequirementAsPendingWithTheCause(t *testing.T) {
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
			summary, err := SummarizeRequiredChecks(tt.checks, nil, unknown)
			if err != nil || summary.State != tt.want || !errors.Is(summary.Cause, ErrRequiredChecksUnknown) {
				t.Fatalf("SummarizeRequiredChecks = %+v, %v, want state %v with the cause", summary, err, tt.want)
			}
		})
	}
}

func TestSummarizeRequiredChecksReturnsAnotherReadErrorAsCauseWithUnknownState(t *testing.T) {
	readErr := errors.New("connection reset")
	summary, err := SummarizeRequiredChecks([]Check{namedCompletedCheck("build", ConclusionSuccess)}, nil, readErr)
	if err != nil || !errors.Is(summary.Cause, readErr) || summary.State != ChecksUnknown {
		t.Fatalf("SummarizeRequiredChecks = %+v, %v, want %v with the read error as cause", summary, err, ChecksUnknown)
	}
}

func TestSummarizeRequiredChecksRefusesARequirementWithoutAName(t *testing.T) {
	_, err := SummarizeRequiredChecks([]Check{namedCompletedCheck("build", ConclusionSuccess)}, []RequiredCheck{{}}, nil)
	if !errors.Is(err, ErrEmptyRequiredCheckName) {
		t.Fatalf("SummarizeRequiredChecks with a zero requirement error = %v, want %v", err, ErrEmptyRequiredCheckName)
	}
}

func TestSummarizeRequiredChecksWithoutRequirementReadsTheLatestAttempt(t *testing.T) {
	failedFirstRun := attempt(namedCompletedCheck("build", ConclusionFailure), 1)
	passedRerun := attempt(namedCompletedCheck("build", ConclusionSuccess), 2)
	got := summarizeRequired([]Check{failedFirstRun, passedRerun}, nil)
	if got != ChecksPassed {
		t.Fatalf("SummarizeRequiredChecks without requirement = %v, want %v: a re-run supersedes the run it repeats", got, ChecksPassed)
	}
	if SummarizeChecks([]Check{failedFirstRun, passedRerun}) != ChecksFailed {
		t.Fatalf("SummarizeChecks of the same runs must still count the failure")
	}
}

func requiredCheck(t *testing.T, name string, appID int64) RequiredCheck {
	t.Helper()
	required, err := NewRequiredCheck(name, appID)
	if err != nil {
		t.Fatal(err)
	}
	return required
}

func TestNewRequiredCheckRefusesAnEmptyName(t *testing.T) {
	if _, err := NewRequiredCheck("", 1); !errors.Is(err, ErrEmptyRequiredCheckName) {
		t.Fatalf("NewRequiredCheck(\"\") error = %v, want %v", err, ErrEmptyRequiredCheckName)
	}
}

func TestSummarizeChecksReadsAFailureOverASuccessWithoutAttempt(t *testing.T) {
	failed := namedCompletedCheck("build", ConclusionFailure)
	passed := namedCompletedCheck("build", ConclusionSuccess)
	tests := []struct {
		name   string
		checks []Check
	}{
		{"failure listed first", []Check{failed, passed}},
		{"success listed first", []Check{passed, failed}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SummarizeChecks(tt.checks); got != ChecksFailed {
				t.Fatalf("SummarizeChecks = %v, want %v: a run without attempt must not hide a failure", got, ChecksFailed)
			}
		})
	}
}

func TestSummarizeRequiredChecksReadsAFailureOverASuccessWithoutAttempt(t *testing.T) {
	failed := namedCompletedCheck("build", ConclusionFailure)
	passed := namedCompletedCheck("build", ConclusionSuccess)
	required := []RequiredCheck{requiredCheck(t, "build", 0)}
	tests := []struct {
		name   string
		checks []Check
	}{
		{"failure listed first", []Check{failed, passed}},
		{"success listed first", []Check{passed, failed}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := summarizeRequired(tt.checks, required)
			if got != ChecksFailed {
				t.Fatalf("SummarizeRequiredChecks = %v, want %v: a run without attempt must not hide a failure", got, ChecksFailed)
			}
		})
	}
}

func TestSummarizeChecksCountsEveryRunOfARepeatedCheck(t *testing.T) {
	failedFirstRun := attempt(namedCompletedCheck("build", ConclusionFailure), 1)
	passedRerun := attempt(namedCompletedCheck("build", ConclusionSuccess), 2)
	if got := SummarizeChecks([]Check{failedFirstRun, passedRerun}); got != ChecksFailed {
		t.Fatalf("SummarizeChecks = %v, want %v: SummarizeChecks keeps every run as published", got, ChecksFailed)
	}
	if got := SummarizeChecks([]Check{passedRerun, failedFirstRun}); got != ChecksFailed {
		t.Fatalf("SummarizeChecks listed in reverse = %v, want %v", got, ChecksFailed)
	}
}

func summarizeRequired(checks []Check, required []RequiredCheck) ChecksState {
	summary, err := SummarizeRequiredChecks(checks, required, nil)
	if err != nil {
		return ChecksUnknown
	}
	return summary.State
}
