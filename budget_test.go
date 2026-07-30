package contracts_test

import (
	"errors"
	"strings"
	"testing"

	contracts "github.com/Herrscherd/herrscher-contracts"
)

func TestBudgetErrorMessageAndAs(t *testing.T) {
	err := error(&contracts.BudgetError{Key: "projects/x/fact", Runes: 2100, Limit: 2000})

	msg := err.Error()
	for _, want := range []string{"projects/x/fact", "2100", "2000"} {
		if !strings.Contains(msg, want) {
			t.Fatalf("message %q missing %q", msg, want)
		}
	}

	var be *contracts.BudgetError
	if !errors.As(err, &be) {
		t.Fatal("errors.As failed to match *BudgetError")
	}
	if be.Runes != 2100 || be.Limit != 2000 {
		t.Fatalf("unexpected fields: %+v", be)
	}
}

func TestEnforceBudgetOverReturnsBudgetError(t *testing.T) {
	// "é" is 2 bytes / 1 rune; 100 of them = 100 runes, 200 bytes.
	body := strings.Repeat("é", 100)
	err := contracts.EnforceBudget("user:alice", body, 50)
	var be *contracts.BudgetError
	if !errors.As(err, &be) {
		t.Fatalf("want *BudgetError, got %T (%v)", err, err)
	}
	if be.Runes != 100 || be.Limit != 50 || be.Key != "user:alice" {
		t.Fatalf("got Key=%q Runes=%d Limit=%d", be.Key, be.Runes, be.Limit)
	}
}

func TestEnforceBudgetUnderReturnsNil(t *testing.T) {
	if err := contracts.EnforceBudget("k", strings.Repeat("é", 40), 50); err != nil {
		t.Fatalf("want nil, got %v", err)
	}
}

func TestEnforceBudgetDisabledByNonPositiveLimit(t *testing.T) {
	for _, limit := range []int{0, -1} {
		if err := contracts.EnforceBudget("k", strings.Repeat("x", 999), limit); err != nil {
			t.Fatalf("limit %d should disable, got %v", limit, err)
		}
	}
}
