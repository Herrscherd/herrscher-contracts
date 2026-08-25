package contracts

import "testing"

// TestKindSkillIsDistinct locks the wire value and guards against a copy-paste
// collision with another kind: the learned-skill projection and the promote
// guard both switch on it, so a value equal to an existing kind would silently
// widen both.
func TestKindSkillIsDistinct(t *testing.T) {
	if KindSkill != "skill" {
		t.Fatalf("KindSkill = %q, want %q", KindSkill, "skill")
	}
	others := []NodeKind{
		KindOrganization, KindProject, KindRepo, KindServer,
		KindArchitecture, KindProduction, KindSession, KindDecision,
		KindUser, KindAgent, KindDomain, KindTranscript,
	}
	for _, k := range others {
		if k == KindSkill {
			t.Errorf("KindSkill collides with %q", k)
		}
	}
}
