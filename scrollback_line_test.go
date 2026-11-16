package contracts

import (
	"encoding/json"
	"testing"
)

func TestScrollbackLineWithoutKindKeepsOldWireFormat(t *testing.T) {
	want := ScrollbackLine{Role: "assistant", Text: "bonjour"}
	const wantJSON = `{"Role":"assistant","Text":"bonjour"}`
	assertJSONRoundTrip(t, want, wantJSON, func(raw []byte) (any, error) {
		var got ScrollbackLine
		err := json.Unmarshal(raw, &got)
		if err == nil && got != want {
			t.Fatalf("ScrollbackLine decoded to %+v, want %+v", got, want)
		}
		return got, err
	})
}

func TestScrollbackLineNoticeCarriesTurnInterruptedKind(t *testing.T) {
	want := ScrollbackLine{Role: "notice", Text: "tour interrompu", Kind: ScrollbackKindTurnInterrupted}
	const wantJSON = `{"Role":"notice","Text":"tour interrompu","Kind":"turn_interrupted"}`
	assertJSONRoundTrip(t, want, wantJSON, func(raw []byte) (any, error) {
		var got ScrollbackLine
		err := json.Unmarshal(raw, &got)
		if err == nil && got != want {
			t.Fatalf("ScrollbackLine decoded to %+v, want %+v", got, want)
		}
		return got, err
	})
}
