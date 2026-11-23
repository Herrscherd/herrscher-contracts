package contracts_test

import (
	"strings"
	"testing"

	contracts "github.com/Herrscherd/herrscher-contracts"
)

func TestBudgetErrorDoesNotRepeatTheKey(t *testing.T) {
	const received = "s3cret-received-text"
	err := contracts.EnforceBudget("user:"+received, strings.Repeat("a", 10), 5)
	if err == nil {
		t.Fatal("EnforceBudget accepted a body over its budget")
	}
	if strings.Contains(err.Error(), received) {
		t.Fatalf("message %q echoes the node key", err)
	}
}
