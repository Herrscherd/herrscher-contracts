package approvalsock

import (
	"errors"
	"os"
	"testing"
)

func TestCheckSameDirectoryRefusesReplacedDirectory(t *testing.T) {
	first := t.TempDir()
	second := t.TempDir()
	before, err := os.Stat(first)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if err := checkSameDirectory(first, before); err != nil {
		t.Fatalf("same directory refused: %v", err)
	}
	if err := checkSameDirectory(second, before); !errors.Is(err, ErrDirectoryChanged) {
		t.Fatalf("replaced directory error = %v, want ErrDirectoryChanged", err)
	}
}

func TestNextAcceptDelayGrowsAndCaps(t *testing.T) {
	delay := nextAcceptDelay(0)
	if delay != minAcceptDelay {
		t.Fatalf("first delay = %v, want %v", delay, minAcceptDelay)
	}
	for range 20 {
		delay = nextAcceptDelay(delay)
	}
	if delay != maxAcceptDelay {
		t.Fatalf("capped delay = %v, want %v", delay, maxAcceptDelay)
	}
}
