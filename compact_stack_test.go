package contracts

import (
	"context"
	"errors"
	"slices"
	"testing"
)

type compactFunc func(context.Context, CompactRequest, func(BackendEvent)) (CompactResult, error)

func (f compactFunc) Compact(ctx context.Context, req CompactRequest, onEvent func(BackendEvent)) (CompactResult, error) {
	return f(ctx, req, onEvent)
}

var errStageFailed = errors.New("stage failed")

func ignoreEvents(BackendEvent) {}

func recordingStage(name string, result CompactResult, err error, seen *[]CompactRequest, ran *[]string) CompactStage {
	return CompactStage{
		Name: name,
		Compactor: compactFunc(func(_ context.Context, req CompactRequest, _ func(BackendEvent)) (CompactResult, error) {
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
	result, err := RunCompactStack(t.Context(), stages, CompactRequest{}, ignoreEvents)
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
	if _, err := RunCompactStack(t.Context(), stages, CompactRequest{TokensUsed: 1000, TokenLimit: 8000}, ignoreEvents); err != nil {
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
		recordingStage("first", CompactResult{TokensAfter: 400, TokensMeasured: false}, nil, &seen, &ran),
		recordingStage("second", CompactResult{}, nil, &seen, &ran),
	}
	if _, err := RunCompactStack(t.Context(), stages, CompactRequest{TokensUsed: 1000}, ignoreEvents); err != nil {
		t.Fatalf("RunCompactStack() error = %v", err)
	}
	if seen[1].TokensUsed != 1000 {
		t.Fatalf("second stage saw usage %d, want 1000: an unmeasured TokensAfter is not a count", seen[1].TokensUsed)
	}
}

func TestRunCompactStackMarksUsageMeasuredOnceAStageMeasuresIt(t *testing.T) {
	var seen []CompactRequest
	var ran []string
	stages := []CompactStage{
		recordingStage("unmeasured", CompactResult{}, nil, &seen, &ran),
		recordingStage("measured", CompactResult{TokensAfter: 300, TokensMeasured: true}, nil, &seen, &ran),
		recordingStage("after", CompactResult{}, nil, &seen, &ran),
	}
	if _, err := RunCompactStack(t.Context(), stages, CompactRequest{TokenLimit: 8000}, ignoreEvents); err != nil {
		t.Fatalf("RunCompactStack() error = %v", err)
	}
	if got := []bool{seen[0].UsageMeasured, seen[1].UsageMeasured, seen[2].UsageMeasured}; !slices.Equal(got, []bool{false, false, true}) {
		t.Fatalf("stages saw UsageMeasured %v, want [false false true]", got)
	}
}

func TestRunCompactStackStopsAtADoneStage(t *testing.T) {
	var seen []CompactRequest
	var ran []string
	stages := []CompactStage{
		recordingStage("enough", CompactResult{TokensAfter: 100, TokensMeasured: true, Note: "compacted", Done: true}, nil, &seen, &ran),
		recordingStage("never", CompactResult{}, nil, &seen, &ran),
	}
	result, err := RunCompactStack(t.Context(), stages, CompactRequest{}, ignoreEvents)
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
	_, err := RunCompactStack(t.Context(), stages, CompactRequest{}, ignoreEvents)
	if !errors.Is(err, errStageFailed) {
		t.Fatalf("RunCompactStack() error = %v, want the stage's cause", err)
	}
	if want := []string{"broken"}; !slices.Equal(ran, want) {
		t.Fatalf("stages ran %v, want %v", ran, want)
	}
}

func TestRunCompactStackRefusesAnEmptyStack(t *testing.T) {
	_, err := RunCompactStack(t.Context(), nil, CompactRequest{}, ignoreEvents)
	if !errors.Is(err, ErrNoCompactStage) {
		t.Fatalf("RunCompactStack(nil) error = %v, want ErrNoCompactStage", err)
	}
}

func TestRunCompactStackRefusesANilCompactorAndStopsThere(t *testing.T) {
	var seen []CompactRequest
	var ran []string
	stages := []CompactStage{
		recordingStage("first", CompactResult{}, nil, &seen, &ran),
		{Name: "unwired"},
		recordingStage("never", CompactResult{}, nil, &seen, &ran),
	}
	_, err := RunCompactStack(t.Context(), stages, CompactRequest{}, ignoreEvents)
	if !errors.Is(err, ErrNilCompactor) {
		t.Fatalf("RunCompactStack() error = %v, want ErrNilCompactor", err)
	}
	if want := []string{"first"}; !slices.Equal(ran, want) {
		t.Fatalf("stages ran %v, want %v", ran, want)
	}
}

func TestRunCompactStackStopsBeforeAStageWhenTheContextIsCancelled(t *testing.T) {
	var ran []string
	ctx, cancel := context.WithCancel(t.Context())
	stages := []CompactStage{
		{Name: "first", Compactor: compactFunc(func(context.Context, CompactRequest, func(BackendEvent)) (CompactResult, error) {
			ran = append(ran, "first")
			cancel()
			return CompactResult{}, nil
		})},
		{Name: "second", Compactor: compactFunc(func(context.Context, CompactRequest, func(BackendEvent)) (CompactResult, error) {
			ran = append(ran, "second")
			return CompactResult{}, nil
		})},
	}
	_, err := RunCompactStack(ctx, stages, CompactRequest{}, ignoreEvents)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("RunCompactStack() error = %v, want context.Canceled", err)
	}
	if want := []string{"first"}; !slices.Equal(ran, want) {
		t.Fatalf("stages ran %v, want %v", ran, want)
	}
}

func TestRunCompactStackForwardsProgressToEveryStage(t *testing.T) {
	emitting := func(name string) CompactStage {
		return CompactStage{Name: name, Compactor: compactFunc(func(_ context.Context, _ CompactRequest, onEvent func(BackendEvent)) (CompactResult, error) {
			onEvent(BackendEvent{Kind: name})
			return CompactResult{}, nil
		})}
	}
	var kinds []string
	stages := []CompactStage{emitting("first"), emitting("second")}
	_, err := RunCompactStack(t.Context(), stages, CompactRequest{}, func(event BackendEvent) {
		kinds = append(kinds, event.Kind)
	})
	if err != nil {
		t.Fatalf("RunCompactStack() error = %v", err)
	}
	if want := []string{"first", "second"}; !slices.Equal(kinds, want) {
		t.Fatalf("progress events %v, want %v", kinds, want)
	}
}

func TestRunCompactStackDoesNotMarkUsageMeasuredWithoutATokenLimit(t *testing.T) {
	var seen []CompactRequest
	var ran []string
	stages := []CompactStage{
		recordingStage("measured", CompactResult{TokensAfter: 500, TokensMeasured: true}, nil, &seen, &ran),
		recordingStage("after", CompactResult{}, nil, &seen, &ran),
	}
	if _, err := RunCompactStack(t.Context(), stages, CompactRequest{}, ignoreEvents); err != nil {
		t.Fatalf("RunCompactStack() error = %v", err)
	}
	if seen[1].UsageMeasured {
		t.Fatalf("second stage saw UsageMeasured = true with TokenLimit 0, want false: the usage has no window to be read against")
	}
}

func TestRunCompactStackKeepsMeasuredUsageThroughAnUnmeasuredStage(t *testing.T) {
	var seen []CompactRequest
	var ran []string
	stages := []CompactStage{
		recordingStage("unmeasured", CompactResult{}, nil, &seen, &ran),
		recordingStage("after", CompactResult{}, nil, &seen, &ran),
	}
	request := CompactRequest{UsageMeasured: true, TokensUsed: 1000, TokenLimit: 8000}
	if _, err := RunCompactStack(t.Context(), stages, request, ignoreEvents); err != nil {
		t.Fatalf("RunCompactStack() error = %v", err)
	}
	if !seen[1].UsageMeasured || seen[1].TokensUsed != 1000 {
		t.Fatalf("second stage saw UsageMeasured %v and usage %d, want true and 1000", seen[1].UsageMeasured, seen[1].TokensUsed)
	}
}

func TestRunCompactStackToleratesANilEventHandler(t *testing.T) {
	emitting := CompactStage{Name: "emitting", Compactor: compactFunc(func(_ context.Context, _ CompactRequest, onEvent func(BackendEvent)) (CompactResult, error) {
		onEvent(BackendEvent{Kind: "progress"})
		return CompactResult{}, nil
	})}
	if _, err := RunCompactStack(t.Context(), []CompactStage{emitting}, CompactRequest{}, nil); err != nil {
		t.Fatalf("RunCompactStack() error = %v", err)
	}
}

func TestRunCompactStackDropsAnInitialUsageMeasuredWithoutATokenLimit(t *testing.T) {
	var seen []CompactRequest
	var ran []string
	stages := []CompactStage{
		recordingStage("unmeasured", CompactResult{}, nil, &seen, &ran),
		recordingStage("after", CompactResult{}, nil, &seen, &ran),
	}
	request := CompactRequest{UsageMeasured: true, TokensUsed: 1000, TokenLimit: 0}
	if _, err := RunCompactStack(t.Context(), stages, request, ignoreEvents); err != nil {
		t.Fatalf("RunCompactStack() error = %v", err)
	}
	if seen[1].UsageMeasured {
		t.Fatalf("second stage saw UsageMeasured = true with TokenLimit 0, want false: the request's flag is not trusted without a window")
	}
}

func TestRunCompactStackCollectsTheStatsOfEachStage(t *testing.T) {
	cases := map[string]struct {
		results []CompactResult
		want    []CompactStats
	}{
		"every stage reports": {
			results: []CompactResult{
				{Stats: &CompactStats{Outcome: CompactShort, Calls: 3, Pruned: 1}},
				{Stats: &CompactStats{Outcome: CompactDryRun, Calls: 5, Requests: 2}},
			},
			want: []CompactStats{
				{Stage: "first", Outcome: CompactShort, Calls: 3, Pruned: 1},
				{Stage: "second", Outcome: CompactDryRun, Calls: 5, Requests: 2},
			},
		},
		"a silent stage adds nothing": {
			results: []CompactResult{
				{Stats: &CompactStats{Outcome: CompactShort, Kept: 4}},
				{},
			},
			want: []CompactStats{{Stage: "first", Outcome: CompactShort, Kept: 4}},
		},
		"a stage's own name is kept": {
			results: []CompactResult{
				{Stats: &CompactStats{Stage: "custom", Outcome: CompactSkipped}},
				{},
			},
			want: []CompactStats{{Stage: "custom", Outcome: CompactSkipped}},
		},
		"a done stage ends the collection": {
			results: []CompactResult{
				{Stats: &CompactStats{Outcome: CompactDone, Truncated: 2}, Done: true},
				{Stats: &CompactStats{Outcome: CompactShort}},
			},
			want: []CompactStats{{Stage: "first", Outcome: CompactDone, Truncated: 2}},
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			var seen []CompactRequest
			var ran []string
			stages := []CompactStage{
				recordingStage("first", tc.results[0], nil, &seen, &ran),
				recordingStage("second", tc.results[1], nil, &seen, &ran),
			}
			result, err := RunCompactStack(t.Context(), stages, CompactRequest{}, ignoreEvents)
			if err != nil {
				t.Fatalf("RunCompactStack() error = %v", err)
			}
			if !slices.Equal(result.StageStats, tc.want) {
				t.Fatalf("StageStats = %+v, want %+v", result.StageStats, tc.want)
			}
		})
	}
}

func TestRunCompactStackKeepsTheStatsOfTheStagesBeforeAFailure(t *testing.T) {
	var seen []CompactRequest
	var ran []string
	stages := []CompactStage{
		recordingStage("first", CompactResult{Stats: &CompactStats{Outcome: CompactShort, Calls: 2}}, nil, &seen, &ran),
		recordingStage("broken", CompactResult{Stats: &CompactStats{Outcome: CompactFailed}}, errStageFailed, &seen, &ran),
		recordingStage("never", CompactResult{Stats: &CompactStats{Outcome: CompactDone}}, nil, &seen, &ran),
	}
	result, err := RunCompactStack(t.Context(), stages, CompactRequest{}, ignoreEvents)
	if !errors.Is(err, errStageFailed) {
		t.Fatalf("RunCompactStack() error = %v, want the stage's cause", err)
	}
	if len(result.StageStats) != 2 || result.StageStats[0].Stage != "first" || result.StageStats[1].Stage != "broken" {
		t.Fatalf("StageStats = %+v, want the stats of first and broken", result.StageStats)
	}
	if result.StageStats[0].Calls != 2 || result.StageStats[1].Outcome != CompactFailed {
		t.Fatalf("StageStats = %+v, want first's calls and broken's failure", result.StageStats)
	}
}
