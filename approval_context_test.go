package contracts

import (
	"context"
	"testing"
)

func TestApprovalAskerRoundTrips(t *testing.T) {
	asker := func(_ context.Context, req ApprovalRequest) ApprovalAnswer {
		return ApprovalAnswer{ID: req.ID, Decision: DecisionAllow}
	}
	ctx := WithApprovalAsker(context.Background(), asker)
	got, ok := ApprovalAskerFrom(ctx)
	if !ok {
		t.Fatal("ApprovalAskerFrom found no asker on a context that carries one")
	}
	if answer := got(ctx, ApprovalRequest{ID: "7"}); answer.ID != "7" || !answer.Allowed() {
		t.Fatalf("round-tripped asker answered %+v, want the allowed answer for ID 7", answer)
	}
}

func TestApprovalAskerAbsentFromBareContext(t *testing.T) {
	if _, ok := ApprovalAskerFrom(context.Background()); ok {
		t.Fatal("ApprovalAskerFrom found an asker on a bare context")
	}
}

func TestApprovalAskerNilIsAbsent(t *testing.T) {
	ctx := WithApprovalAsker(context.Background(), nil)
	if _, ok := ApprovalAskerFrom(ctx); ok {
		t.Fatal("ApprovalAskerFrom returned a nil asker as present")
	}
}
