package approvalsock

import (
	"encoding/json"
	"errors"
	"net"
	"strings"
	"testing"

	contracts "github.com/Herrscherd/herrscher-contracts"
)

const answeredSecret = "s3cret-answered-id"

func TestRoundTripRefusesAnAnswerForAnotherRequestWithoutEcho(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()
	go func() {
		_, _ = readLine(server)
		_ = json.NewEncoder(server).Encode(contracts.ApprovalAnswer{ID: answeredSecret, Decision: contracts.DecisionAllow})
	}()
	_, err := roundTrip(client, []byte(`{"ID":"req-1"}`+"\n"), "req-1")
	if !errors.Is(err, ErrAnswerMismatch) {
		t.Fatalf("error = %v, want ErrAnswerMismatch", err)
	}
	if strings.Contains(err.Error(), answeredSecret) {
		t.Fatalf("message %q echoes the answer ID", err)
	}
}
