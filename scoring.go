package contracts

import (
	"math"
	"strings"
	"time"
	"unicode"
)

const (
	weightTF       = 1.0
	weightTitleHit = 3.0
	weightRecency  = 2.0
	weightKind     = 1.0
	weightProx     = 1.5

	recencyHalfLifeDays = 30.0
	recencyNeutral      = 0.5
	hoursPerDay         = 24.0

	kindBoostCore    = 1.0
	kindBoostEntity  = 0.6
	kindBoostSession = 0.2
	kindBoostOther   = 0.4
)

func tokenize(s string) []string {
	return strings.FieldsFunc(strings.ToLower(s), func(r rune) bool {
		return !isWordRune(r)
	})
}

func isWordRune(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r)
}

type ranker struct {
	terms map[string]bool
	now   time.Time
}

func newRanker(text string, now time.Time) ranker {
	terms := make(map[string]bool)
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

func (r ranker) score(node Node) (float64, bool) {
	if len(r.terms) == 0 {
		return 0, false
	}
	var termFrequency, titleHits float64
	seen := make(map[string]bool)
	for _, tok := range tokenize(node.Title) {
		if r.terms[tok] {
			termFrequency++
			if !seen[tok] {
				seen[tok] = true
				titleHits++
			}
		}
	}
	for _, tok := range tokenize(node.Body) {
		if r.terms[tok] {
			termFrequency++
		}
	}
	if termFrequency == 0 {
		return 0, false
	}
	total := weightTF*termFrequency + weightTitleHit*titleHits
	total += weightRecency * r.recencyScore(node)
	total += weightKind * kindBoost(node.Kind)
	return total, true
}

func (r ranker) recencyScore(node Node) float64 {
	if r.now.IsZero() {
		return recencyNeutral
	}
	raw := node.Meta["capturedAt"]
	if raw == "" {
		return recencyNeutral
	}
	capturedAt, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return recencyNeutral
	}
	ageDays := r.now.Sub(capturedAt).Hours() / hoursPerDay
	if ageDays < 0 {
		ageDays = 0
	}
	return math.Exp(-math.Ln2 * ageDays / recencyHalfLifeDays)
}

func kindBoost(k NodeKind) float64 {
	switch k {
	case KindDecision, KindUser, KindArchitecture, KindProduction:
		return kindBoostCore
	case KindProject, KindRepo, KindServer, KindOrganization, KindDomain, KindAgent:
		return kindBoostEntity
	case KindSession:
		return kindBoostSession
	case KindTranscript, KindSkill:
		return kindBoostOther
	default:
		return kindBoostOther
	}
}

func proximityBoost(depth int) float64 {
	if depth < 0 {
		return 0
	}
	return 1.0 / float64(1+depth)
}
