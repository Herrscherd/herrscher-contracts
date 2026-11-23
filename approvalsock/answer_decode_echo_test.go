package approvalsock

import (
	"errors"
	"io"
	"net"
	"strings"
	"testing"

	contracts "github.com/Herrscherd/herrscher-contracts"
	"github.com/Herrscherd/herrscher-contracts/echotest"
)

func TestRoundTripRefusesAnAnswerWithoutEcho(t *testing.T) {
	t.Parallel()
	for name, tc := range map[string]struct {
		reply    string
		secret   string
		sentinel error
	}{
		"not JSON": {
			reply: echotest.Probe + "\n", secret: echotest.Probe, sentinel: ErrUnreadableAnswer,
		},
		"truncated JSON": {
			reply: `{"ID":"` + echotest.Probe + `"` + "\n", secret: echotest.Probe, sentinel: ErrUnreadableAnswer,
		},
		"wrong ID type": {
			reply: `{"ID":4242,"Decision":"allow"}` + "\n", secret: "4242", sentinel: ErrUnreadableAnswer,
		},
		"wrong Decision type": {
			reply: `{"ID":"req-1","Decision":5151}` + "\n", secret: "5151", sentinel: ErrUnreadableAnswer,
		},
		"wrong Refusal type": {
			reply: `{"ID":"req-1","Decision":"deny","Refusal":7373}` + "\n", secret: "7373", sentinel: ErrUnreadableAnswer,
		},
		"oversized": {
			reply:    `{"ID":"req-1","Decision":"allow","Refusal":"` + echotest.Probe + strings.Repeat("r", maxMessageBytes) + `"}` + "\n",
			secret:   echotest.Probe,
			sentinel: ErrUnreadableAnswer,
		},
		"unknown decision": {
			reply: `{"ID":"req-1","Decision":"` + echotest.Probe + `"}` + "\n", secret: echotest.Probe, sentinel: contracts.ErrInvalidDecision,
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			client, server := net.Pipe()
			t.Cleanup(func() { closePipes(t, client, server) })
			go replyToRequest(t, server, tc.reply)
			_, err := roundTrip(client, []byte(`{"ID":"req-1"}`+"\n"), "req-1")
			echotest.RequireRefusalWithoutEcho(t, err, tc.sentinel, tc.secret)
		})
	}
}

func replyToRequest(t *testing.T, server net.Conn, reply string) {
	t.Helper()
	if _, err := readLine(server); err != nil {
		t.Errorf("read request: %v", err)
		return
	}
	if _, err := server.Write([]byte(reply)); err != nil && !errors.Is(err, io.ErrClosedPipe) {
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
