package contracts

import (
	"encoding/json"
	"testing"
)

func TestSessionInfoJSONKeepsGoFieldNames(t *testing.T) {
	want := SessionInfo{
		Name:          "lead",
		Incarnation:   "inc-1",
		ChannelID:     "chan-9",
		Type:          "terminal",
		Gateways:      []string{"discord", "terminal"},
		Vendor:        "claude",
		Model:         "claude-opus-5",
		Effort:        "high",
		Project:       "herrscher",
		Dir:           "/w/lead",
		Archived:      true,
		Resumable:     true,
		LastTs:        "2026-11-03T10:00:00Z",
		ContextTokens: 4200,
	}
	const wantJSON = `{"Name":"lead","Incarnation":"inc-1","ChannelID":"chan-9","Type":"terminal","Gateways":["discord","terminal"],"Vendor":"claude","Model":"claude-opus-5","Effort":"high","Project":"herrscher","Dir":"/w/lead","Archived":true,"Resumable":true,"LastTs":"2026-11-03T10:00:00Z","ContextTokens":4200}`
	assertJSONRoundTrip(t, want, wantJSON, func(raw []byte) (any, error) {
		var got SessionInfo
		err := json.Unmarshal(raw, &got)
		if err == nil && (got.Name != want.Name || got.ChannelID != want.ChannelID || len(got.Gateways) != len(want.Gateways) || got.ContextTokens != want.ContextTokens) {
			t.Fatalf("SessionInfo decoded to %+v, want %+v", got, want)
		}
		return got, err
	})
}

func TestScrollbackLineJSONKeepsGoFieldNames(t *testing.T) {
	want := ScrollbackLine{Role: "user", Text: "hello"}
	const wantJSON = `{"Role":"user","Text":"hello"}`
	assertJSONRoundTrip(t, want, wantJSON, func(raw []byte) (any, error) {
		var got ScrollbackLine
		err := json.Unmarshal(raw, &got)
		if err == nil && got != want {
			t.Fatalf("ScrollbackLine decoded to %+v, want %+v", got, want)
		}
		return got, err
	})
}

func TestLocationJSONKeepsGoFieldNames(t *testing.T) {
	want := Location{Obsidian: "obsidian://open?vault=herrscher", File: "/vault/n.md"}
	const wantJSON = `{"Obsidian":"obsidian://open?vault=herrscher","File":"/vault/n.md"}`
	assertJSONRoundTrip(t, want, wantJSON, func(raw []byte) (any, error) {
		var got Location
		err := json.Unmarshal(raw, &got)
		if err == nil && got != want {
			t.Fatalf("Location decoded to %+v, want %+v", got, want)
		}
		return got, err
	})
}

func assertJSONRoundTrip(t *testing.T, value any, wantJSON string, decode func([]byte) (any, error)) {
	t.Helper()
	got, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(got) != wantJSON {
		t.Fatalf("wire JSON changed:\n got %s\nwant %s", got, wantJSON)
	}
	decoded, err := decode([]byte(wantJSON))
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	again, err := json.Marshal(decoded)
	if err != nil {
		t.Fatalf("re-marshal: %v", err)
	}
	if string(again) != wantJSON {
		t.Fatalf("round trip changed the wire JSON:\n got %s\nwant %s", again, wantJSON)
	}
}
