package contracts

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/Herrscherd/herrscher-contracts/echotest"
)

func TestArgMarshalJSONRefusesALongValueWithoutEchoingTheName(t *testing.T) {
	arg := Text(receivedProbe, strings.Repeat("a", maxNoticeArgBytes+1))
	_, err := json.Marshal(arg)
	if !errors.Is(err, ErrNoticeLimit) {
		t.Fatalf("error = %v, want ErrNoticeLimit", err)
	}
	echotest.RequireNoSecretFragment(t, err.Error(), receivedProbe)
}
