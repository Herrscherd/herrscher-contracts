package contracts

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestCompactTriggerStringNamesEachTrigger(t *testing.T) {
	if got := TriggerManual.String(); got != "manual" {
		t.Fatalf("TriggerManual.String() = %q, want %q", got, "manual")
	}
	if got := TriggerAutomatic.String(); got != "automatic" {
		t.Fatalf("TriggerAutomatic.String() = %q, want %q", got, "automatic")
	}
}

func TestCompactTriggerZeroValueIsNotAName(t *testing.T) {
	var unset CompactTrigger
	if got := unset.String(); got != "CompactTrigger(0)" {
		t.Fatalf("zero CompactTrigger.String() = %q, want the raw number", got)
	}
	if _, err := ParseCompactTrigger(unset.String()); !errors.Is(err, ErrUnknownCompactTrigger) {
		t.Fatalf("ParseCompactTrigger accepted the zero value's text: %v", err)
	}
}

func TestParseCompactTriggerReadsEveryTriggerBack(t *testing.T) {
	for _, want := range CompactTriggers() {
		got, err := ParseCompactTrigger(want.String())
		if err != nil || got != want {
			t.Fatalf("ParseCompactTrigger(%q) = %v, %v, want %v", want.String(), got, err, want)
		}
	}
}

func TestParseCompactTriggerRefusesUnknownText(t *testing.T) {
	for _, text := range []string{"", "Manual", "auto", "CompactTrigger(0)"} {
		got, err := ParseCompactTrigger(text)
		if !errors.Is(err, ErrUnknownCompactTrigger) {
			t.Errorf("ParseCompactTrigger(%q) error = %v, want ErrUnknownCompactTrigger", text, err)
		}
		if got != 0 {
			t.Errorf("ParseCompactTrigger(%q) = %v, want the zero value on error", text, got)
		}
	}
}

func TestCompactTriggerMarshalsItsTextForm(t *testing.T) {
	type envelope struct {
		Trigger CompactTrigger `json:"Trigger"`
	}
	raw, err := json.Marshal(envelope{Trigger: TriggerAutomatic})
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	if string(raw) != `{"Trigger":"automatic"}` {
		t.Fatalf("wire JSON = %s, want the text form", raw)
	}
	var back envelope
	if err := json.Unmarshal(raw, &back); err != nil || back.Trigger != TriggerAutomatic {
		t.Fatalf("json.Unmarshal() = %+v, %v, want TriggerAutomatic", back, err)
	}
}

func TestCompactTriggerRefusesToMarshalAnUnknownValue(t *testing.T) {
	_, err := CompactTrigger(9).MarshalText()
	if !errors.Is(err, ErrUnknownCompactTrigger) {
		t.Fatalf("MarshalText(9) error = %v, want ErrUnknownCompactTrigger", err)
	}
}

func TestCompactTriggerUnmarshalRefusesUnknownText(t *testing.T) {
	var trigger CompactTrigger
	err := json.Unmarshal([]byte(`"soon"`), &trigger)
	if !errors.Is(err, ErrUnknownCompactTrigger) {
		t.Fatalf("json.Unmarshal(\"soon\") error = %v, want ErrUnknownCompactTrigger", err)
	}
}

func TestValidateAutoCompactPercentAcceptsZeroAndTheRange(t *testing.T) {
	for _, percent := range []int{0, 1, 92, 100} {
		if err := ValidateAutoCompactPercent(percent); err != nil {
			t.Fatalf("ValidateAutoCompactPercent(%d) error = %v, want nil", percent, err)
		}
	}
}

func TestValidateAutoCompactPercentRefusesOutOfRange(t *testing.T) {
	for _, percent := range []int{-1, 101, 250} {
		if err := ValidateAutoCompactPercent(percent); !errors.Is(err, ErrAutoCompactPercentOutOfRange) {
			t.Fatalf("ValidateAutoCompactPercent(%d) error = %v, want ErrAutoCompactPercentOutOfRange", percent, err)
		}
	}
}
