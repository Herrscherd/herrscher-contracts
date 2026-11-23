package contracts

import (
	"errors"
	"strings"
	"testing"

	"github.com/Herrscherd/herrscher-contracts/echotest"
)

func TestValidateHistoryEditsRefusesEachEditByRankWithoutEchoingTheCall(t *testing.T) {
	keep := HistoryEdit{CallID: "call-1", Action: PruneResult}
	tests := []struct {
		name  string
		edits []HistoryEdit
		want  string
	}{
		{
			name:  "no call",
			edits: []HistoryEdit{keep, {Action: PruneResult}},
			want:  "edit 2 names no call",
		},
		{
			name:  "call edited twice",
			edits: []HistoryEdit{keep, {CallID: receivedProbe, Action: PruneResult}, {CallID: receivedProbe, Action: PruneResult}},
			want:  "edit 3 edits a call twice",
		},
		{
			name:  "negative KeepChars",
			edits: []HistoryEdit{{CallID: receivedProbe, Action: PruneResult, KeepChars: -1}},
			want:  "edit 1 keeps -1 runes",
		},
		{
			name:  "negative KeepTail",
			edits: []HistoryEdit{keep, {CallID: receivedProbe, Action: PruneResult, KeepTail: -2}},
			want:  "edit 2 keeps -2 runes from its end",
		},
		{
			name:  "tail on a pruned call",
			edits: []HistoryEdit{{CallID: receivedProbe, Action: PruneCall, KeepTail: 3}},
			want:  "edit 1 keeps a tail of a pruned call",
		},
		{
			name:  "unknown action",
			edits: []HistoryEdit{keep, {CallID: receivedProbe, Action: PruneAction(99)}},
			want:  "edit 2: unknown prune action: 99",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateHistoryEdits(tt.edits)
			if !errors.Is(err, ErrInvalidHistoryEdit) {
				t.Fatalf("error = %v, want ErrInvalidHistoryEdit", err)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("message %q does not name %q", err, tt.want)
			}
			echotest.RequireNoSecretFragment(t, err.Error(), receivedProbe)
		})
	}
}
