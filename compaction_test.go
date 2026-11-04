package contracts

import (
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
