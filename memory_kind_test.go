package contracts

import "testing"

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
