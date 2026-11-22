package contracts

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestDecodeNoticeArgsNamesACutStreamWithTheSentinel(t *testing.T) {
	tests := []struct {
		name string
		raw  string
	}{
		{name: "blank", raw: " "},
		{name: "list cut inside an argument", raw: "[{"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := decodeNoticeArgs(json.RawMessage(tt.raw))
			if !errors.Is(err, ErrNoticeArgs) {
				t.Fatalf("decodeNoticeArgs(%q) = %v, want ErrNoticeArgs", tt.raw, err)
			}
			if !strings.Contains(err.Error(), noticeWireArgs) {
				t.Fatalf("message %q does not name %q", err.Error(), noticeWireArgs)
			}
		})
	}
}

func TestDecodeNoticeArgsKeepsTheLimitSentinelAlone(t *testing.T) {
	raw := "[" + strings.Repeat(`{"name":"n","kind":0,"value":null},`, maxNoticeArgs) + `{"name":"n","kind":0,"value":null}]`
	_, err := decodeNoticeArgs(json.RawMessage(raw))
	if !errors.Is(err, ErrNoticeLimit) {
		t.Fatalf("decodeNoticeArgs = %v, want ErrNoticeLimit", err)
	}
	if errors.Is(err, ErrNoticeArgs) {
		t.Fatalf("decodeNoticeArgs = %v, want the limit not to match ErrNoticeArgs", err)
	}
}
