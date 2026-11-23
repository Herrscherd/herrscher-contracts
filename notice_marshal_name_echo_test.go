package contracts

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestArgMarshalJSONRefusesALongValueWithoutEchoingTheName(t *testing.T) {
	arg := Text(receivedSecret, strings.Repeat("a", maxNoticeArgBytes+1))
	_, err := json.Marshal(arg)
	if !errors.Is(err, ErrNoticeLimit) {
		t.Fatalf("error = %v, want ErrNoticeLimit", err)
	}
	if strings.Contains(err.Error(), receivedSecret) {
		t.Fatalf("message %q echoes the argument name", err)
	}
}
