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
		{"action required fails", ConclusionActionRequired, ChecksFailed},
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

func TestSummarizeChecksRanksFailedOverPendingOverPassed(t *testing.T) {
	queued := Check{Status: CheckQueued}
	running := Check{Status: CheckInProgress}
	passed := completedCheck(ConclusionSuccess)
	failed := completedCheck(ConclusionFailure)
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
		{"failed outranks pending", []Check{running, failed}, ChecksFailed},
		{"failed outranks pending when listed first", []Check{failed, queued}, ChecksFailed},
		{"failed outranks passed", []Check{passed, failed}, ChecksFailed},
		{"failed outranks every other check", []Check{passed, running, failed, passed}, ChecksFailed},
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
