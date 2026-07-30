package contracts

import (
	"testing"
	"time"
)

func TestNextState(t *testing.T) {
	now := time.Date(2026, 7, 30, 12, 0, 0, 0, time.UTC)
	const stale = 30 * 24 * time.Hour
	const archive = 90 * 24 * time.Hour

	cases := []struct {
		name        string
		lastSeen    time.Time
		stale, arch time.Duration
		want        string
	}{
		{"fresh", now.Add(-time.Hour), stale, archive, StateActive},
		{"just before stale", now.Add(-(stale - time.Minute)), stale, archive, StateActive},
		{"exactly at stale", now.Add(-stale), stale, archive, StateStale},
		{"between stale and archive", now.Add(-60 * 24 * time.Hour), stale, archive, StateStale},
		{"exactly at archive", now.Add(-archive), stale, archive, StateArchived},
		{"well past archive", now.Add(-365 * 24 * time.Hour), stale, archive, StateArchived},
		{"stale disabled", now.Add(-60 * 24 * time.Hour), 0, archive, StateActive},
		{"archive disabled stays stale", now.Add(-365 * 24 * time.Hour), stale, 0, StateStale},
		{"both disabled", now.Add(-365 * 24 * time.Hour), 0, 0, StateActive},
		{"reactivation: recent lastSeen", now.Add(-time.Minute), stale, archive, StateActive},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := NextState(c.lastSeen, now, c.stale, c.arch); got != c.want {
				t.Fatalf("NextState = %q, want %q", got, c.want)
			}
		})
	}
}
