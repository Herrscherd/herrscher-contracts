package contracts

import (
	"encoding/json"
	"testing"

	"github.com/Herrscherd/herrscher-contracts/i18n"
)

func TestSessionInfoJSONKeepsGoFieldNames(t *testing.T) {
	want := SessionInfo{
		Name:               "lead",
		Incarnation:        "inc-1",
		ChannelID:          "chan-9",
		Type:               "terminal",
		Gateways:           []string{"discord", "terminal"},
		Vendor:             "claude",
		Model:              "claude-opus-5",
		Effort:             "high",
		Project:            "herrscher",
		Dir:                "/w/lead",
		Archived:           true,
		Resumable:          true,
		LastTs:             "2026-11-03T10:00:00Z",
		ContextTokens:      4200,
		AutoCompactPercent: 92,
		ContextWindow:      400000,
		Language:           i18n.French,
	}
	const wantJSON = `{"Name":"lead","Incarnation":"inc-1","ChannelID":"chan-9","Type":"terminal","Gateways":["discord","terminal"],"Vendor":"claude","Model":"claude-opus-5","Effort":"high","Project":"herrscher","Dir":"/w/lead","Archived":true,"Resumable":true,"LastTs":"2026-11-03T10:00:00Z","ContextTokens":4200,"AutoCompactPercent":92,"ContextWindow":400000,"Language":2}`
	assertJSONRoundTrip(t, want, wantJSON, func(raw []byte) (any, error) {
		var got SessionInfo
		err := json.Unmarshal(raw, &got)
		if err == nil && (got.Name != want.Name || got.ChannelID != want.ChannelID || len(got.Gateways) != len(want.Gateways) || got.ContextTokens != want.ContextTokens || got.ContextWindow != want.ContextWindow) {
			t.Fatalf("SessionInfo decoded to %+v, want %+v", got, want)
		}
		return got, err
	})
}

func TestSessionInfoWithoutContextWindowDecodesToZero(t *testing.T) {
	const oldDaemonJSON = `{"Name":"lead","Vendor":"codex","ContextTokens":4200,"AutoCompactPercent":92}`
	var got SessionInfo
	if err := json.Unmarshal([]byte(oldDaemonJSON), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.ContextWindow != 0 {
		t.Fatalf("ContextWindow = %d from a message without the field, want 0", got.ContextWindow)
	}
	if got.ContextTokens != 4200 || got.AutoCompactPercent != 92 {
		t.Fatalf("the other fields decoded to %+v", got)
	}
}

func TestSessionInfoWithoutLanguageDecodesToUnknown(t *testing.T) {
	const oldDaemonJSON = `{"Name":"lead","Vendor":"claude","ContextWindow":400000}`
	var got SessionInfo
	if err := json.Unmarshal([]byte(oldDaemonJSON), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Language != 0 {
		t.Fatalf("Language = %v from a message without the field, want the zero value", got.Language)
	}
	if got.ContextWindow != 400000 {
		t.Fatalf("the other fields decoded to %+v", got)
	}
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
