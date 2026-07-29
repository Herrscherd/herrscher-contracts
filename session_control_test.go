package contracts

import "testing"

func TestSessionInfoCarriesIncarnation(t *testing.T) {
	const want = "session-42-v3"
	info := SessionInfo{Incarnation: want}
	if info.Incarnation != want {
		t.Fatalf("Incarnation = %q, want %q", info.Incarnation, want)
	}
}
