package contracts

import (
	"context"
	"errors"
	"slices"
	"testing"
)

type compactFunc func(context.Context, CompactRequest) (CompactResult, error)

func (f compactFunc) Compact(ctx context.Context, req CompactRequest) (CompactResult, error) {
	return f(ctx, req)
}

var errStageFailed = errors.New("stage failed")

func recordingStage(name string, result CompactResult, err error, seen *[]CompactRequest, ran *[]string) CompactStage {
	return CompactStage{
		Name: name,
		Compactor: compactFunc(func(_ context.Context, req CompactRequest) (CompactResult, error) {
			*seen = append(*seen, req)
			*ran = append(*ran, name)
			return result, err
		}),
	}
}

func TestRunCompactStackRunsStagesInOrderAndReturnsTheLastResult(t *testing.T) {
	var seen []CompactRequest
	var ran []string
	stages := []CompactStage{
		recordingStage("first", CompactResult{Note: "one"}, nil, &seen, &ran),
		recordingStage("second", CompactResult{Note: "two"}, nil, &seen, &ran),
		recordingStage("third", CompactResult{Note: "three"}, nil, &seen, &ran),
	}
	result, err := RunCompactStack(t.Context(), stages, CompactRequest{})
	if err != nil {
		t.Fatalf("RunCompactStack() error = %v", err)
	}
	if want := []string{"first", "second", "third"}; !slices.Equal(ran, want) {
		t.Fatalf("stages ran %v, want %v", ran, want)
	}
	if result.Note != "three" {
		t.Fatalf("result = %+v, want the last stage's result", result)
	}
}

func TestRunCompactStackHandsEachStageTheUsageLeftByThePrevious(t *testing.T) {
	var seen []CompactRequest
	var ran []string
	stages := []CompactStage{
		recordingStage("first", CompactResult{TokensAfter: 500, TokensMeasured: true}, nil, &seen, &ran),
		recordingStage("second", CompactResult{TokensAfter: 300, TokensMeasured: true}, nil, &seen, &ran),
		recordingStage("third", CompactResult{}, nil, &seen, &ran),
	}
	if _, err := RunCompactStack(t.Context(), stages, CompactRequest{TokensUsed: 1000, TokenLimit: 8000}); err != nil {
		t.Fatalf("RunCompactStack() error = %v", err)
	}
	if got := []int{seen[0].TokensUsed, seen[1].TokensUsed, seen[2].TokensUsed}; !slices.Equal(got, []int{1000, 500, 300}) {
		t.Fatalf("stages saw usage %v, want [1000 500 300]", got)
	}
	if seen[2].TokenLimit != 8000 {
		t.Fatalf("TokenLimit = %d, want 8000 carried through the stack", seen[2].TokenLimit)
	}
}

func TestRunCompactStackKeepsTheUsageWhenPreviousIsUnmeasured(t *testing.T) {
	var seen []CompactRequest
	var ran []string
	stages := []CompactStage{
		recordingStage("first", CompactResult{}, nil, &seen, &ran),
		recordingStage("second", CompactResult{}, nil, &seen, &ran),
	}
	if _, err := RunCompactStack(t.Context(), stages, CompactRequest{TokensUsed: 1000}); err != nil {
		t.Fatalf("RunCompactStack() error = %v", err)
	}
	if seen[1].TokensUsed != 1000 {
		t.Fatalf("second stage saw usage %d, want 1000: an unmeasured zero is not a count", seen[1].TokensUsed)
	}
}

func TestRunCompactStackStopsAtADoneStage(t *testing.T) {
	var seen []CompactRequest
	var ran []string
	stages := []CompactStage{
		recordingStage("enough", CompactResult{TokensAfter: 100, TokensMeasured: true, Note: "compacted", Done: true}, nil, &seen, &ran),
		recordingStage("never", CompactResult{}, nil, &seen, &ran),
	}
	result, err := RunCompactStack(t.Context(), stages, CompactRequest{})
	if err != nil {
		t.Fatalf("RunCompactStack() error = %v", err)
	}
	if want := []string{"enough"}; !slices.Equal(ran, want) {
		t.Fatalf("stages ran %v, want %v", ran, want)
	}
	if result.Note != "compacted" {
		t.Fatalf("result = %+v, want the Done stage's result", result)
	}
}

func TestRunCompactStackStopsOnAFailureAndWrapsTheCause(t *testing.T) {
	var seen []CompactRequest
	var ran []string
	stages := []CompactStage{
		recordingStage("broken", CompactResult{}, errStageFailed, &seen, &ran),
		recordingStage("never", CompactResult{}, nil, &seen, &ran),
	}
	_, err := RunCompactStack(t.Context(), stages, CompactRequest{})
	if !errors.Is(err, errStageFailed) {
		t.Fatalf("RunCompactStack() error = %v, want the stage's cause", err)
	}
	if want := []string{"broken"}; !slices.Equal(ran, want) {
		t.Fatalf("stages ran %v, want %v", ran, want)
	}
}

func TestRunCompactStackRefusesAnEmptyStack(t *testing.T) {
	_, err := RunCompactStack(t.Context(), nil, CompactRequest{})
	if !errors.Is(err, ErrNoCompactStage) {
		t.Fatalf("RunCompactStack(nil) error = %v, want ErrNoCompactStage", err)
	}
}
