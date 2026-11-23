package approvalsock

import (
	"encoding/json"
	"errors"
	"net"
	"strings"
	"testing"

	contracts "github.com/Herrscherd/herrscher-contracts"
)

const answeredSecret = "zq8Vk2wXr7Lm"

const echoFragmentBytes = 4

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
		answer := contracts.ApprovalAnswer{ID: answeredSecret, Decision: contracts.DecisionAllow}
		if err := json.NewEncoder(server).Encode(answer); err != nil {
			t.Errorf("encode answer: %v", err)
		}
	}()
	_, err := roundTrip(client, []byte(`{"ID":"req-1"}`+"\n"), "req-1")
	if !errors.Is(err, ErrAnswerMismatch) {
		t.Fatalf("error = %v, want ErrAnswerMismatch", err)
	}
	requireNoSecretFragment(t, err.Error(), answeredSecret)
}

func requireNoSecretFragment(tb testing.TB, msg, secret string) {
	tb.Helper()
	for start := 0; start+echoFragmentBytes <= len(secret); start++ {
		fragment := secret[start : start+echoFragmentBytes]
		if strings.Contains(msg, fragment) {
			tb.Fatalf("message %q echoes %q from the answer ID", msg, fragment)
		}
	}
}
