package contracts

import (
	"context"
	"testing"
)

type reactor struct{ got string }

func (r *reactor) React(_ context.Context, reply string) string {
	r.got = reply
	return "clean"
}

func TestTurnReactorSatisfied(t *testing.T) {
	var tr TurnReactor = &reactor{}
	if out := tr.React(context.Background(), "raw <recall>x</recall>"); out != "clean" {
		t.Fatalf("React must return the cleaned reply, got %q", out)
	}
}
