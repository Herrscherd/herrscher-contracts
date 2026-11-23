package contracts

import (
	"strings"
	"testing"
)

func TestBudgetErrorDoesNotRepeatTheKey(t *testing.T) {
	err := EnforceBudget(receivedSecret, strings.Repeat("a", 10), 5)
	if err == nil {
		t.Fatal("EnforceBudget accepted a body over its budget")
	}
	requireNoSecretFragment(t, err.Error(), receivedSecret)
}
