package contracts

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestHistoryRoleRoundTripsThroughJSON(t *testing.T) {
	for _, want := range HistoryRoles() {
		raw, err := json.Marshal(want)
		if err != nil {
			t.Fatalf("Marshal(%v): %v", want, err)
		}
		var got HistoryRole
		if err := json.Unmarshal(raw, &got); err != nil || got != want {
			t.Fatalf("Unmarshal(%s) = %v, %v, want %v", raw, got, err, want)
		}
	}
}

func TestHistoryRoleZeroValueDoesNotMarshal(t *testing.T) {
	var unset HistoryRole
	if _, err := json.Marshal(unset); !errors.Is(err, ErrUnknownHistoryRole) {
		t.Fatalf("Marshal(zero role) error = %v, want ErrUnknownHistoryRole", err)
	}
	if _, err := ParseHistoryRole("system"); !errors.Is(err, ErrUnknownHistoryRole) {
		t.Fatalf("ParseHistoryRole(system) error = %v, want ErrUnknownHistoryRole", err)
	}
}

func TestPruneActionRoundTripsThroughJSON(t *testing.T) {
	for _, want := range PruneActions() {
		raw, err := json.Marshal(want)
		if err != nil {
			t.Fatalf("Marshal(%v): %v", want, err)
		}
		var got PruneAction
		if err := json.Unmarshal(raw, &got); err != nil || got != want {
			t.Fatalf("Unmarshal(%s) = %v, %v, want %v", raw, got, err, want)
		}
	}
}

func TestPruneActionRefusesUnknownText(t *testing.T) {
	var action PruneAction
	if err := action.UnmarshalText([]byte("drop")); !errors.Is(err, ErrUnknownPruneAction) {
		t.Fatalf("UnmarshalText(drop) error = %v, want ErrUnknownPruneAction", err)
	}
	if _, err := PruneAction(0).MarshalText(); !errors.Is(err, ErrUnknownPruneAction) {
		t.Fatalf("MarshalText(zero action) error = %v, want ErrUnknownPruneAction", err)
	}
}

func TestValidateHistoryEditsAcceptsDistinctKnownEdits(t *testing.T) {
	edits := []HistoryEdit{
		{CallID: "t1", Action: PruneResult, KeepChars: 300},
		{CallID: "t2", Action: PruneCall},
		{CallID: "t3", Action: PruneResult, KeepChars: 200, KeepTail: 100},
	}
	if err := ValidateHistoryEdits(edits); err != nil {
		t.Fatalf("ValidateHistoryEdits(%+v) = %v, want nil", edits, err)
	}
	if err := ValidateHistoryEdits(nil); err != nil {
		t.Fatalf("ValidateHistoryEdits(nil) = %v, want nil", err)
	}
}

func TestValidateHistoryEditsRefusesEachMalformedEdit(t *testing.T) {
	cases := map[string][]HistoryEdit{
		"no call":        {{Action: PruneCall}},
		"twice":          {{CallID: "t1", Action: PruneCall}, {CallID: "t1", Action: PruneResult}},
		"unknown action": {{CallID: "t1"}},
		"negative keep":  {{CallID: "t1", Action: PruneResult, KeepChars: -1}},
		"negative tail":  {{CallID: "t1", Action: PruneResult, KeepTail: -1}},
		"tail on a call": {{CallID: "t1", Action: PruneCall, KeepTail: 20}},
	}
	for name, edits := range cases {
		if err := ValidateHistoryEdits(edits); !errors.Is(err, ErrInvalidHistoryEdit) {
			t.Fatalf("%s: ValidateHistoryEdits error = %v, want ErrInvalidHistoryEdit", name, err)
		}
	}
}

func TestHistoryCallsListsCallsInMessageOrder(t *testing.T) {
	history := History{Messages: []HistoryMessage{
		{Role: RoleUser, Text: "fix it"},
		{Role: RoleAssistant, Calls: []HistoryCall{{ID: "t1"}, {ID: "t2"}}},
		{Role: RoleAssistant, Calls: []HistoryCall{{ID: "t3"}}},
	}}
	calls := history.Calls()
	if len(calls) != 3 || calls[0].ID != "t1" || calls[2].ID != "t3" {
		t.Fatalf("Calls() = %+v, want t1, t2, t3", calls)
	}
}
