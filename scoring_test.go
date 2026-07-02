package contracts

import (
	"testing"
	"time"
)

func TestTokenize_LowercasesAndSplitsOnNonAlnum(t *testing.T) {
	got := tokenize("NATS, transport-layer!")
	want := []string{"nats", "transport", "layer"}
	if len(got) != len(want) {
		t.Fatalf("want %v, got %v", want, got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("token %d: want %q, got %q", i, want[i], got[i])
		}
	}
}

func TestScore_TwoTermBeatsOneTerm(t *testing.T) {
	r := newRanker("nats transport", time.Time{})
	both := Node{Title: "x", Body: "we use nats for transport"}
	one := Node{Title: "x", Body: "we use nats only"}
	sBoth, hitBoth := r.score(both)
	sOne, _ := r.score(one)
	if !hitBoth || sBoth <= sOne {
		t.Fatalf("two-term (%.3f) must outrank one-term (%.3f)", sBoth, sOne)
	}
}

func TestScore_TitleHitBeatsBodyOnly(t *testing.T) {
	r := newRanker("nats", time.Time{})
	inTitle := Node{Title: "nats decision", Body: "z"}
	inBody := Node{Title: "z", Body: "nats decision"}
	sT, _ := r.score(inTitle)
	sB, _ := r.score(inBody)
	if sT <= sB {
		t.Fatalf("title hit (%.3f) must outrank body-only (%.3f)", sT, sB)
	}
}

func TestScore_RecentBeatsStaleAtEqualText(t *testing.T) {
	now := time.Date(2026, 7, 2, 0, 0, 0, 0, time.UTC)
	r := newRanker("nats", now)
	recent := Node{Title: "nats", Meta: map[string]string{"capturedAt": now.Add(-24 * time.Hour).Format(time.RFC3339)}}
	stale := Node{Title: "nats", Meta: map[string]string{"capturedAt": now.Add(-365 * 24 * time.Hour).Format(time.RFC3339)}}
	sR, _ := r.score(recent)
	sS, _ := r.score(stale)
	if sR <= sS {
		t.Fatalf("recent (%.3f) must outrank stale (%.3f)", sR, sS)
	}
}

func TestScore_MissingDateIsNeutralNotZero(t *testing.T) {
	now := time.Date(2026, 7, 2, 0, 0, 0, 0, time.UTC)
	r := newRanker("nats", now)
	noDate := Node{Title: "nats"}
	veryStale := Node{Title: "nats", Meta: map[string]string{"capturedAt": now.Add(-3650 * 24 * time.Hour).Format(time.RFC3339)}}
	sNo, _ := r.score(noDate)
	sStale, _ := r.score(veryStale)
	if sNo <= sStale {
		t.Fatalf("missing date (%.3f) should be neutral, above very stale (%.3f)", sNo, sStale)
	}
}

func TestScore_NoTextMatchHasNoTextHit(t *testing.T) {
	r := newRanker("nats", time.Time{})
	_, hit := r.score(Node{Title: "redis", Body: "cache"})
	if hit {
		t.Fatal("node with no query term must report textHit=false")
	}
}

func TestScore_KindBoostOrdersDurableAboveSession(t *testing.T) {
	r := newRanker("nats", time.Time{})
	dec := Node{Title: "nats", Kind: KindDecision}
	sess := Node{Title: "nats", Kind: KindSession}
	sD, _ := r.score(dec)
	sS, _ := r.score(sess)
	if sD <= sS {
		t.Fatalf("decision (%.3f) must outrank session (%.3f) at equal text", sD, sS)
	}
}

func TestScore_ExportedMatchesInternal(t *testing.T) {
	now := time.Date(2026, 7, 2, 0, 0, 0, 0, time.UTC)
	n := Node{Title: "nats", Kind: KindDecision}
	got, hit := Score("nats", now, n)
	want, wantHit := newRanker("nats", now).score(n)
	if got != want || hit != wantHit {
		t.Fatalf("Score() must match ranker.score(): (%.3f,%v) vs (%.3f,%v)", got, hit, want, wantHit)
	}
}
