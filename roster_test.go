package contracts

import "testing"

type fakeRoster struct{ agents []AgentInfo }

func (f fakeRoster) Agents() []AgentInfo { return f.agents }

func TestRosterProviderSatisfied(t *testing.T) {
	var _ RosterProvider = fakeRoster{}
	r := fakeRoster{agents: []AgentInfo{{Name: "codex", Backend: "codex", Tags: []string{"refactor"}}}}
	got := r.Agents()
	if len(got) != 1 || got[0].Name != "codex" || got[0].Backend != "codex" {
		t.Fatalf("unexpected roster projection: %+v", got)
	}
}
