package contracts

import (
	"math"
	"strings"
	"time"
)

// Ranking weights. Constants for now; a later PR may make them configurable.
const (
	weightTF       = 1.0 // per query-term occurrence in Title+Body
	weightTitleHit = 3.0 // per distinct query term found in Title
	weightRecency  = 2.0 // scaled by recencyScore in [0,1]
	weightKind     = 1.0 // scaled by kindBoost in [0,1]
	weightProx     = 1.5 // scaled by proximityBoost in (0,1]; applied by RecallRelevant

	// recencyHalfLifeDays sets how fast recency decays: a node captured this
	// long ago scores 0.5 recency; older decays toward 0, newer toward 1.
	recencyHalfLifeDays = 30.0
	// recencyNeutral is the recency subscore for a node with no capturedAt —
	// neutral, so an undated node is never penalized below a moderately old one.
	recencyNeutral = 0.5
)

// tokenize lowercases s and splits on any non-alphanumeric run, dropping empties.
// Shared by query parsing and node scanning so both sides tokenize identically.
func tokenize(s string) []string {
	return strings.FieldsFunc(strings.ToLower(s), func(r rune) bool {
		return !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9')
	})
}

// ranker scores nodes against a fixed query. now is the reference time for
// recency decay; a zero now disables recency (subscore stays neutral for all),
// which keeps text-only tests deterministic.
type ranker struct {
	terms map[string]bool
	now   time.Time
}

func newRanker(text string, now time.Time) ranker {
	terms := map[string]bool{}
	for _, t := range tokenize(text) {
		terms[t] = true
	}
	return ranker{terms: terms, now: now}
}

// Score exposes the lexical relevance score of n against text for out-of-package
// rankers (e.g. the Obsidian Search implementation). now drives recency decay;
// pass a zero Time to disable it. The bool reports whether any query term matched
// Title or Body, letting callers exclude non-matches.
func Score(text string, now time.Time, n Node) (float64, bool) {
	return newRanker(text, now).score(n)
}

// score returns a node's relevance score and whether it matched any query term.
// It combines term frequency, a title-hit boost, recency decay, and a per-kind
// boost. Proximity is NOT included here — RecallRelevant adds it, since only the
// graph walk knows a node's distance from a root. textHit is false when no query
// term appears in Title or Body, letting callers exclude non-matches.
func (r ranker) score(n Node) (total float64, textHit bool) {
	if len(r.terms) == 0 {
		return 0, false
	}
	var tf float64
	for _, tok := range tokenize(n.Title + "\n" + n.Body) {
		if r.terms[tok] {
			tf++
		}
	}
	if tf == 0 {
		return 0, false
	}
	var titleHits float64
	seen := map[string]bool{}
	for _, tok := range tokenize(n.Title) {
		if r.terms[tok] && !seen[tok] {
			seen[tok] = true
			titleHits++
		}
	}
	total = weightTF*tf + weightTitleHit*titleHits
	total += weightRecency * r.recencyScore(n)
	total += weightKind * kindBoost(n.Kind)
	return total, true
}

// recencyScore maps a node's Meta["capturedAt"] to [0,1] via exponential decay.
// A missing/unparseable date, or a zero reference now, yields the neutral score.
func (r ranker) recencyScore(n Node) float64 {
	if r.now.IsZero() {
		return recencyNeutral
	}
	raw := n.Meta["capturedAt"]
	if raw == "" {
		return recencyNeutral
	}
	at, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return recencyNeutral
	}
	ageDays := r.now.Sub(at).Hours() / 24
	if ageDays < 0 {
		ageDays = 0
	}
	return math.Exp(-math.Ln2 * ageDays / recencyHalfLifeDays)
}

// kindBoost weights durable knowledge above transient session logs, in [0,1].
func kindBoost(k NodeKind) float64 {
	switch k {
	case KindDecision, KindUser, KindArchitecture, KindProduction:
		return 1.0
	case KindProject, KindRepo, KindServer, KindOrganization, KindDomain, KindAgent:
		return 0.6
	case KindSession:
		return 0.2
	default:
		return 0.4
	}
}

// proximityBoost decays with graph distance from a scope root: depth 0 → 1.
func proximityBoost(depth int) float64 {
	if depth < 0 {
		return 0
	}
	return 1.0 / float64(1+depth)
}
