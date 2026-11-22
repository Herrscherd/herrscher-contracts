package contracts

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestEventRoundTripKeepsEveryField(t *testing.T) {
	when := time.Date(2026, time.November, 22, 10, 30, 15, 250000000, time.FixedZone("CET", 3600))
	want := Event{
		T: "reply", Who: "alice", Text: "done", Value: "2", Done: true, Cost: 0.0042,
		Tokens: 55, TokensIn: 30, CacheRead: 12, CacheCreate: 3,
		Models: []ModelUsage{{Model: "opus", In: 30, Out: 55, Cost: 0.004}},
		Resume: "resume-1", Project: "neublox", Attachments: []string{"shot.png"},
		SessionIncarnation: "session-42-v3", TurnID: "turn-8f31", Agent: "reviewer",
		Coordination: &CoordinationEvent{Kind: "delegated", SourceSession: "lead", Agent: "roblox-scripter"},
		Todos:        []TodoItem{{Text: "write the test", State: "doing"}},
		Subagent:     &Subagent{ID: "sub-1", Name: "scout", Kind: "explore", State: "active"},
		ToolCall:     &ToolCall{ID: "call-1", Tool: "bash", Detail: "go test", State: ToolCallStarted, At: when},
	}
	encoded, err := json.Marshal(want)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	var got Event
	if err = json.Unmarshal(encoded, &got); err != nil {
		t.Fatalf("Unmarshal(%s) error = %v", encoded, err)
	}
	again, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("Marshal(decoded) error = %v", err)
	}
	if !bytes.Equal(again, encoded) {
		t.Fatalf("round trip changed the event:\n got %s\nwant %s", again, encoded)
	}
}

func TestEventRefusesAWrongJSONTypeByItsWireName(t *testing.T) {
	tests := []struct {
		name  string
		input string
		field string
	}{
		{name: "top level field", input: `{"t":5}`, field: "t"},
		{name: "misspelled key is canonical", input: `{"T":5}`, field: "t"},
		{name: "nested in coordination", input: `{"t":"reply","coordination":{"kind":5}}`, field: "coordination"},
		{name: "nested in todos", input: `{"t":"reply","todos":[{"text":5}]}`, field: "todos"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var e Event
			err := json.Unmarshal([]byte(tt.input), &e)
			if !errors.Is(err, ErrInvalidEvent) {
				t.Fatalf("Unmarshal(%s) = %v, want ErrInvalidEvent", tt.input, err)
			}
			msg := err.Error()
			if !strings.Contains(msg, tt.field+" has the wrong JSON type") {
				t.Fatalf("message %q does not name the wire field %q", msg, tt.field)
			}
			if strings.Contains(msg, "Event") || strings.Contains(msg, "contracts.") {
				t.Fatalf("message %q names a Go type", msg)
			}
		})
	}
}

func TestEventRefusesANonObjectByItsWireName(t *testing.T) {
	var e Event
	err := json.Unmarshal([]byte(`"reply"`), &e)
	if !errors.Is(err, ErrInvalidEvent) {
		t.Fatalf("Unmarshal() = %v, want ErrInvalidEvent", err)
	}
	if msg := err.Error(); strings.Contains(msg, "Event") || strings.Contains(msg, "contracts.") {
		t.Fatalf("message %q names a Go type", msg)
	}
}

func TestEventKeepsAToolCallRefusal(t *testing.T) {
	var e Event
	err := json.Unmarshal([]byte(`{"t":"reply","tool_call":{"id":5}}`), &e)
	if !errors.Is(err, ErrInvalidToolCall) {
		t.Fatalf("Unmarshal() = %v, want ErrInvalidToolCall", err)
	}
	if errors.Is(err, ErrInvalidEvent) {
		t.Fatalf("Unmarshal() = %v, a tool call refusal must not become ErrInvalidEvent", err)
	}
	if !strings.Contains(err.Error(), "id has the wrong JSON type") {
		t.Fatalf("message %q does not name the call field", err)
	}
}

func TestEventLeavesASyntaxErrorUnwrapped(t *testing.T) {
	var e Event
	err := json.Unmarshal([]byte(`{"t":"reply"`), &e)
	if _, ok := errors.AsType[*json.SyntaxError](err); !ok {
		t.Fatalf("Unmarshal() = %v, want a *json.SyntaxError", err)
	}
	if errors.Is(err, ErrInvalidEvent) {
		t.Fatalf("Unmarshal() = %v, a syntax error must not match ErrInvalidEvent", err)
	}
}

func TestEventNullLeavesItUnchanged(t *testing.T) {
	e := Event{T: "reply"}
	if err := json.Unmarshal([]byte(`null`), &e); err != nil {
		t.Fatalf("Unmarshal(null) error = %v", err)
	}
	if e.T != "reply" {
		t.Fatalf("Unmarshal(null) reset the event to T = %q", e.T)
	}
}
