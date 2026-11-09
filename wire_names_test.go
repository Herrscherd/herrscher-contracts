package contracts

import (
	"encoding/json"
	"testing"
	"time"

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

func TestPromptJSONKeepsLanguageAsItsNumber(t *testing.T) {
	want := Prompt{Content: "bonjour", Author: "shan", ChannelID: "chan-9", Language: i18n.French, Outbox: "/w/out"}
	const wantJSON = `{"Content":"bonjour","Context":"","Author":"shan","MessageID":"","ChannelID":"chan-9","Attachments":null,"Language":2,"Outbox":"/w/out"}`
	assertJSONRoundTrip(t, want, wantJSON, func(raw []byte) (any, error) {
		var got Prompt
		err := json.Unmarshal(raw, &got)
		if err == nil && got.Language != want.Language {
			t.Fatalf("Prompt decoded Language %v, want %v", got.Language, want.Language)
		}
		return got, err
	})
}

func TestPromptWithoutLanguageDecodesToUnknown(t *testing.T) {
	const oldHostJSON = `{"Content":"bonjour","Author":"shan"}`
	var got Prompt
	if err := json.Unmarshal([]byte(oldHostJSON), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Language != 0 {
		t.Fatalf("Language = %v from a message without the field, want the zero value", got.Language)
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

func TestOutboundFileJSONKeepsGoFieldNames(t *testing.T) {
	want := OutboundFile{Path: "/w/out/pixel.png", Name: "pixel.png", ContentType: "image/png"}
	const wantJSON = `{"Path":"/w/out/pixel.png","Name":"pixel.png","ContentType":"image/png"}`
	assertJSONRoundTrip(t, want, wantJSON, func(raw []byte) (any, error) {
		var got OutboundFile
		err := json.Unmarshal(raw, &got)
		if err == nil && got != want {
			t.Fatalf("OutboundFile decoded to %+v, want %+v", got, want)
		}
		return got, err
	})
}

func TestCompactStatsJSONKeepsGoFieldNames(t *testing.T) {
	want := CompactStats{
		Stage:         "prune",
		Outcome:       CompactShort,
		Calls:         4,
		Kept:          2,
		Truncated:     1,
		Pruned:        1,
		Pinned:        0,
		CharsBefore:   900,
		CharsAfter:    300,
		TokensRemoved: 150,
		Requests:      0,
		RequestTokens: 0,
		Duration:      2 * time.Second,
	}
	const wantJSON = `{"Stage":"prune","Outcome":"short","Calls":4,"Kept":2,"Truncated":1,"Pruned":1,"Pinned":0,"CharsBefore":900,"CharsAfter":300,"TokensRemoved":150,"Requests":0,"RequestTokens":0,"Duration":2000000000}`
	assertJSONRoundTrip(t, want, wantJSON, func(raw []byte) (any, error) {
		var got CompactStats
		err := json.Unmarshal(raw, &got)
		if err == nil && got != want {
			t.Fatalf("CompactStats decoded to %+v, want %+v", got, want)
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
