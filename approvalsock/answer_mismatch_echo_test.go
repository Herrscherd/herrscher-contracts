package approvalsock

import (
	"encoding/json"
	"errors"
	"net"
	"testing"

	contracts "github.com/Herrscherd/herrscher-contracts"
	"github.com/Herrscherd/herrscher-contracts/echotest"
)

func TestRoundTripRefusesAnAnswerForAnotherRequestWithoutEcho(t *testing.T) {
	client, server := net.Pipe()
	t.Cleanup(func() {
		if err := client.Close(); err != nil {
			t.Errorf("close client: %v", err)
		}
		if err := server.Close(); err != nil {
			t.Errorf("close server: %v", err)
		}
	})
	go func() {
		if _, err := readLine(server); err != nil {
			t.Errorf("read request: %v", err)
			return
		}
		answer := contracts.ApprovalAnswer{ID: echotest.Secret, Decision: contracts.DecisionAllow}
		if err := json.NewEncoder(server).Encode(answer); err != nil {
			t.Errorf("encode answer: %v", err)
		}
	}()
	_, err := roundTrip(client, []byte(`{"ID":"req-1"}`+"\n"), "req-1")
	if !errors.Is(err, ErrAnswerMismatch) {
		t.Fatalf("error = %v, want ErrAnswerMismatch", err)
	}
	echotest.RequireNoSecretFragment(t, err.Error(), echotest.Secret)
}
