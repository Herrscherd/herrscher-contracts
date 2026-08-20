package contracts

import (
	"encoding/json"
	"strings"
	"testing"
)

// A settled project rides home on the terminal reply, the same way Resume does.
func TestEventCarriesTheSettledProject(t *testing.T) {
	b, err := json.Marshal(Event{T: "reply", Done: true, Project: "neublox"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `"project":"neublox"`) {
		t.Fatalf("reply did not carry the project: %s", b)
	}
	var back Event
	if err := json.Unmarshal(b, &back); err != nil {
		t.Fatal(err)
	}
	if back.Project != "neublox" {
		t.Fatalf("Project = %q, want %q", back.Project, "neublox")
	}
}

// Every event that settles nothing must stay byte-identical to what it is today,
// because every gateway and every recorded transcript reads this wire.
func TestEventWithoutAProjectSaysNothing(t *testing.T) {
	b, err := json.Marshal(Event{T: "reply", Done: true})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), "project") {
		t.Fatalf("an unsettled reply mentioned a project: %s", b)
	}
}
