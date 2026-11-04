package contracts

import (
	"context"
	"errors"
	"fmt"
)

// CompactStageBackend is the reserved stage name for the session's own backend
// compaction. The host resolves it to the backend of the session, so a
// configuration may place it anywhere in a stack.
const CompactStageBackend = "backend"

// CompactStage is one step of a compaction stack: the name it is configured
// under, and the Compactor that runs it.
type CompactStage struct {
	Name      string
	Compactor Compactor
}

// ErrNoCompactStage is returned by [RunCompactStack] when it is given no stage.
// Match it with errors.Is.
var ErrNoCompactStage = errors.New("no compaction stage")

// ErrNilCompactor is returned by [RunCompactStack] when a stage has no Compactor.
// Match it with errors.Is.
var ErrNilCompactor = errors.New("nil compactor")

// RunCompactStack runs stages in order against req, and returns the result of the
// last stage that ran.
//
// Each stage receives the usage the previous one left: its TokensAfter when that
// was measured, and the usage it was given otherwise. The stack stops at the first
// stage whose result is Done, and returns that result. A stage that fails stops
// the stack, and its error is returned wrapped with the stage's Name, so
// errors.Is still matches the cause. A stage whose Compactor is nil fails with
// [ErrNilCompactor] in the same way. The stages before the failing one have
// already run, and their effect on the conversation is not undone. An empty stack
// returns [ErrNoCompactStage].
func RunCompactStack(ctx context.Context, stages []CompactStage, req CompactRequest) (CompactResult, error) {
	if len(stages) == 0 {
		return CompactResult{}, ErrNoCompactStage
	}
	var result CompactResult
	for _, stage := range stages {
		var err error
		result, err = runCompactStage(ctx, stage, req)
		if err != nil {
			return CompactResult{}, fmt.Errorf("compaction stage %q: %w", stage.Name, err)
		}
		if result.Done {
			return result, nil
		}
		if result.TokensMeasured {
			req.TokensUsed = result.TokensAfter
		}
	}
	return result, nil
}

func runCompactStage(ctx context.Context, stage CompactStage, req CompactRequest) (CompactResult, error) {
	if stage.Compactor == nil {
		return CompactResult{}, ErrNilCompactor
	}
	return stage.Compactor.Compact(ctx, req)
}
