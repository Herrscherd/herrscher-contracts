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
