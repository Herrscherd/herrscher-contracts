package approvalsock

import (
	"net"
	"testing"

	contracts "github.com/Herrscherd/herrscher-contracts"
	"github.com/Herrscherd/herrscher-contracts/echotest"
)

func TestRoundTripRefusesAnUndecodableAnswerWithoutEcho(t *testing.T) {
	t.Parallel()
	for name, tc := range map[string]struct {
		reply  string
		secret string
	}{
		"not JSON":        {reply: echotest.Probe + "\n", secret: echotest.Probe},
		"wrong JSON type": {reply: `{"ID":4242,"Decision":"allow"}` + "\n", secret: "4242"},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			client, server := net.Pipe()
			t.Cleanup(func() { closePipes(t, client, server) })
			go replyToRequest(t, server, tc.reply)
			_, err := roundTrip(client, []byte(`{"ID":"req-1"}`+"\n"), "req-1")
			echotest.RequireRefusalWithoutEcho(t, err, contracts.ErrInvalidDecision, tc.secret)
		})
	}
}

func replyToRequest(t *testing.T, server net.Conn, reply string) {
	t.Helper()
	if _, err := readLine(server); err != nil {
		t.Errorf("read request: %v", err)
		return
	}
	if _, err := server.Write([]byte(reply)); err != nil {
		t.Errorf("write answer: %v", err)
	}
}

func closePipes(t *testing.T, client, server net.Conn) {
	t.Helper()
	if err := client.Close(); err != nil {
		t.Errorf("close client: %v", err)
	}
	if err := server.Close(); err != nil {
		t.Errorf("close server: %v", err)
	}
}
