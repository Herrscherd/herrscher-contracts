package contracts

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

func richEvent() Event {
	return Event{
		T: "reply", Who: "alice", Text: "done", Value: "2", Done: true, Cost: 0.0042,
		Tokens: 55, TokensIn: 30, CacheRead: 12, CacheCreate: 3,
		Models: []ModelUsage{{Model: "opus", In: 30, Out: 55, Cost: 0.004}},
		Resume: "resume-1", Project: "neublox", Attachments: []string{"shot.png"},
		SessionIncarnation: "session-42-v3", TurnID: "turn-8f31", Agent: "reviewer",
		Coordination: &CoordinationEvent{Kind: "delegated", SourceSession: "lead", Agent: "roblox-scripter"},
		Todos:        []TodoItem{{Text: "write the test", State: "doing"}},
		Subagent:     &Subagent{ID: "sub-1", Name: "scout", Kind: "explore", State: "active"},
		ToolCall:     &ToolCall{ID: "call-1", Tool: "bash", Detail: "go test", State: ToolCallStarted, At: time.Date(2026, time.November, 22, 10, 30, 15, 250000000, time.FixedZone("CET", 3600))},
	}
}

func TestEventRoundTripKeepsEveryField(t *testing.T) {
	encoded, err := json.Marshal(richEvent())
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	got, err := DecodeEvent(encoded)
	if err != nil {
		t.Fatalf("DecodeEvent(%s) error = %v", encoded, err)
	}
	again, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("Marshal(decoded) error = %v", err)
	}
	if !bytes.Equal(again, encoded) {
		t.Fatalf("round trip changed the event:\n got %s\nwant %s", again, encoded)
	}
}

type eventEnvelope struct {
	Event

	Session string `json:"session"`
}

func TestEmbeddingEventKeepsTheOuterDecoding(t *testing.T) {
	var envelope eventEnvelope
	if err := json.Unmarshal([]byte(`{"t":"reply","session":"s1"}`), &envelope); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if envelope.Session != "s1" {
		t.Fatalf("Session = %q, want %q: the embedded Event must not capture the decoding", envelope.Session, "s1")
	}
	if envelope.T != "reply" {
		t.Fatalf("T = %q, want %q", envelope.T, "reply")
	}
}

func TestDecodeEventRefusesAWrongJSONTypeByItsWireName(t *testing.T) {
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
			_, err := DecodeEvent([]byte(tt.input))
			if !errors.Is(err, ErrInvalidEvent) {
				t.Fatalf("DecodeEvent(%s) = %v, want ErrInvalidEvent", tt.input, err)
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

func TestDecodeEventNamesANonObjectAsEvent(t *testing.T) {
	_, err := DecodeEvent([]byte(`"reply"`))
	if !errors.Is(err, ErrInvalidEvent) {
		t.Fatalf("DecodeEvent() = %v, want ErrInvalidEvent", err)
	}
	if msg := err.Error(); msg != "invalid event: event has the wrong JSON type" {
		t.Fatalf("message = %q, want the event wire name", msg)
	}
}

func TestDecodeEventKeepsAToolCallRefusal(t *testing.T) {
	_, err := DecodeEvent([]byte(`{"t":"reply","tool_call":{"id":5}}`))
	if !errors.Is(err, ErrInvalidToolCall) {
		t.Fatalf("DecodeEvent() = %v, want ErrInvalidToolCall", err)
	}
	if errors.Is(err, ErrInvalidEvent) {
		t.Fatalf("DecodeEvent() = %v, a tool call refusal must not become ErrInvalidEvent", err)
	}
	if !strings.Contains(err.Error(), "id has the wrong JSON type") {
		t.Fatalf("message %q does not name the call field", err)
	}
}

func TestDecodeEventNamesAToolCallThatIsNotAnObject(t *testing.T) {
	_, err := DecodeEvent([]byte(`{"t":"reply","tool_call":"s3cret-received-text"}`))
	if !errors.Is(err, ErrInvalidToolCall) {
		t.Fatalf("DecodeEvent() = %v, want ErrInvalidToolCall", err)
	}
	msg := err.Error()
	if !strings.Contains(msg, "tool_call has the wrong JSON type") {
		t.Fatalf("message %q does not name tool_call", msg)
	}
	if strings.Contains(msg, "s3cret-received-text") {
		t.Fatalf("message %q echoes the received text", msg)
	}
}

func TestDecodeEventLeavesASyntaxErrorUnwrapped(t *testing.T) {
	_, err := DecodeEvent([]byte(`{"t":"reply"`))
	if _, ok := errors.AsType[*json.SyntaxError](err); !ok {
		t.Fatalf("DecodeEvent() = %v, want a *json.SyntaxError", err)
	}
	if errors.Is(err, ErrInvalidEvent) {
		t.Fatalf("DecodeEvent() = %v, a syntax error must not match ErrInvalidEvent", err)
	}
}

func TestDecodeEventNullIsTheZeroEvent(t *testing.T) {
	got, err := DecodeEvent([]byte(`null`))
	if err != nil {
		t.Fatalf("DecodeEvent(null) error = %v", err)
	}
	if got.T != "" || got.ToolCall != nil || len(got.Models) != 0 {
		t.Fatalf("DecodeEvent(null) = %+v, want the zero Event", got)
	}
}

func TestUnmarshalIntoEventKeepsTheGoTypeError(t *testing.T) {
	var e Event
	err := json.Unmarshal([]byte(`{"t":5}`), &e)
	if _, ok := errors.AsType[*json.UnmarshalTypeError](err); !ok {
		t.Fatalf("json.Unmarshal() = %v, want the *json.UnmarshalTypeError of encoding/json", err)
	}
	if errors.Is(err, ErrInvalidEvent) {
		t.Fatalf("json.Unmarshal() = %v, must not match ErrInvalidEvent: only DecodeEvent names the field", err)
	}
}

func TestDecodeEventNamesEveryWireFieldOfAWrongJSONType(t *testing.T) {
	encoded, err := json.Marshal(richEvent())
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	var fields map[string]json.RawMessage
	if err = json.Unmarshal(encoded, &fields); err != nil {
		t.Fatalf("Unmarshal(fields) error = %v", err)
	}
	for key, value := range fields {
		t.Run(key, func(t *testing.T) {
			input := fmt.Sprintf("{%q:%s}", key, wrongJSONValueFor(value))
			_, err := DecodeEvent([]byte(input))
			wantSentinel := ErrInvalidEvent
			if key == "tool_call" {
				wantSentinel = ErrInvalidToolCall
			}
			if !errors.Is(err, wantSentinel) {
				t.Fatalf("DecodeEvent(%s) = %v, want %v", input, err, wantSentinel)
			}
			if want := key + " has the wrong JSON type"; !strings.Contains(err.Error(), want) {
				t.Fatalf("DecodeEvent(%s) = %q, want it to contain %q", input, err, want)
			}
		})
	}
}

func wrongJSONValueFor(value json.RawMessage) string {
	switch value[0] {
	case '"':
		return "5"
	case '[':
		return "{}"
	default:
		return `"x"`
	}
}
