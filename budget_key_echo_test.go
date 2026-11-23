package contracts

import (
	"strings"
	"testing"

	"github.com/Herrscherd/herrscher-contracts/echotest"
)

func TestBudgetErrorDoesNotRepeatTheKey(t *testing.T) {
	err := EnforceBudget(receivedProbe, strings.Repeat("a", 10), 5)
	if err == nil {
		t.Fatal("EnforceBudget accepted a body over its budget")
	}
	echotest.RequireNoSecretFragment(t, err.Error(), receivedProbe)
}
