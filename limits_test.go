package contracts

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestLimitWindowStringNamesEachWindow(t *testing.T) {
	want := map[LimitWindow]string{LimitFiveHour: "five-hour", LimitWeekly: "weekly", LimitMonthly: "monthly"}
	for window, text := range want {
		if got := window.String(); got != text {
			t.Errorf("%d.String() = %q, want %q", int(window), got, text)
		}
	}
	if len(LimitWindows()) != len(want) {
		t.Fatalf("LimitWindows() = %v, want the %d windows", LimitWindows(), len(want))
	}
}

func TestParseLimitWindowReadsEveryWindowBack(t *testing.T) {
	for _, want := range LimitWindows() {
		got, err := ParseLimitWindow(want.String())
		if err != nil || got != want {
			t.Fatalf("ParseLimitWindow(%q) = %v, %v, want %v", want.String(), got, err, want)
		}
	}
}

func TestParseLimitWindowRefusesUnknownText(t *testing.T) {
	for _, text := range []string{"", "Weekly", "five_hour", "LimitWindow(0)"} {
		got, err := ParseLimitWindow(text)
		if !errors.Is(err, ErrUnknownLimitWindow) {
			t.Errorf("ParseLimitWindow(%q) error = %v, want ErrUnknownLimitWindow", text, err)
		}
		if got != 0 {
			t.Errorf("ParseLimitWindow(%q) = %v, want the zero value on error", text, got)
		}
	}
}

func TestLimitWindowRoundTripsThroughJSON(t *testing.T) {
	type envelope struct {
		Window LimitWindow `json:"window"`
	}
	raw, err := json.Marshal(envelope{Window: LimitFiveHour})
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	if string(raw) != `{"window":"five-hour"}` {
		t.Fatalf("wire JSON = %s, want the text form", raw)
	}
	var back envelope
	if err := json.Unmarshal(raw, &back); err != nil || back.Window != LimitFiveHour {
		t.Fatalf("json.Unmarshal() = %+v, %v, want LimitFiveHour", back, err)
	}
}

func TestLimitWindowRefusesToMarshalAnUnknownValue(t *testing.T) {
	for _, window := range []LimitWindow{0, 9} {
		if _, err := window.MarshalText(); !errors.Is(err, ErrUnknownLimitWindow) {
			t.Errorf("MarshalText(%d) error = %v, want ErrUnknownLimitWindow", int(window), err)
		}
	}
}

func TestLimitWindowUnmarshalRefusesUnknownText(t *testing.T) {
	var window LimitWindow
	if err := json.Unmarshal([]byte(`"daily"`), &window); !errors.Is(err, ErrUnknownLimitWindow) {
		t.Fatalf("json.Unmarshal(\"daily\") error = %v, want ErrUnknownLimitWindow", err)
	}
}

func TestEventCarriesModelUsageOnTheWire(t *testing.T) {
	want := Event{
		T: "reply", Done: true, Cost: 0.03,
		Models: []ModelUsage{
			{Model: "claude-opus-5-5", In: 1200, Out: 300, CacheRead: 9000, CacheCreate: 400, Cost: 0.025},
			{Model: "claude-haiku-5-5", In: 800, Out: 90, Cost: 0.005},
		},
	}
	raw, err := json.Marshal(want)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	const wire = `{"t":"reply","done":true,"cost":0.03,"models":[` +
		`{"model":"claude-opus-5-5","in":1200,"out":300,"cache_read":9000,"cache_create":400,"cost":0.025},` +
		`{"model":"claude-haiku-5-5","in":800,"out":90,"cost":0.005}]}`
	if string(raw) != wire {
		t.Fatalf("wire JSON:\n got %s\nwant %s", raw, wire)
	}
	var got Event
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if len(got.Models) != 2 || got.Models[0] != want.Models[0] || got.Models[1] != want.Models[1] {
		t.Fatalf("decoded models = %+v, want %+v", got.Models, want.Models)
	}
}

func TestEventWithoutModelUsageKeepsItsWire(t *testing.T) {
	raw, err := json.Marshal(Event{T: "reply", Done: true})
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	if string(raw) != `{"t":"reply","done":true}` {
		t.Fatalf("wire JSON = %s, want no models key", raw)
	}
}
