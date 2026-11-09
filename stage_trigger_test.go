package contracts

import (
	"context"
	"errors"
	"testing"
)

type stageWithTrigger struct {
	percent int
}

func (stageWithTrigger) Compact(context.Context, CompactRequest, func(BackendEvent)) (CompactResult, error) {
	return CompactResult{}, nil
}

func (s stageWithTrigger) TriggerPercent() int {
	return s.percent
}

type stageWithoutTrigger struct{}

func (stageWithoutTrigger) Compact(context.Context, CompactRequest, func(BackendEvent)) (CompactResult, error) {
	return CompactResult{}, nil
}

func TestStageTriggerIsOptionalOnACompactor(t *testing.T) {
	var withTrigger Compactor = stageWithTrigger{percent: 50}
	trigger, ok := withTrigger.(StageTrigger)
	if !ok {
		t.Fatalf("stageWithTrigger does not satisfy StageTrigger")
	}
	if got := trigger.TriggerPercent(); got != 50 {
		t.Fatalf("TriggerPercent() = %d, want 50", got)
	}

	var withoutTrigger Compactor = stageWithoutTrigger{}
	if _, ok := withoutTrigger.(StageTrigger); ok {
		t.Fatalf("stageWithoutTrigger satisfies StageTrigger, want it optional")
	}
}

func TestValidateStageTriggerPercent(t *testing.T) {
	cases := []struct {
		name    string
		percent int
		wantErr bool
	}{
		{name: "zero is the host threshold", percent: 0},
		{name: "lowest percent", percent: 1},
		{name: "below the usual host threshold", percent: 50},
		{name: "highest percent", percent: 100},
		{name: "negative", percent: -1, wantErr: true},
		{name: "above the window", percent: 101, wantErr: true},
		{name: "far above the window", percent: 250, wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateStageTriggerPercent(tc.percent)
			if !tc.wantErr {
				if err != nil {
					t.Fatalf("ValidateStageTriggerPercent(%d) error = %v, want nil", tc.percent, err)
				}
				return
			}
			if !errors.Is(err, ErrStageTriggerPercentOutOfRange) {
				t.Fatalf("ValidateStageTriggerPercent(%d) error = %v, want ErrStageTriggerPercentOutOfRange", tc.percent, err)
			}
		})
	}
}
