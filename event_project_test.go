package contracts

import (
	"encoding/json"
	"strings"
	"testing"
)

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

func TestEventWithoutAProjectStaysByteIdentical(t *testing.T) {
	b, err := json.Marshal(Event{T: "reply", Done: true})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(b), `{"t":"reply","done":true}`; got != want {
		t.Fatalf("an unsettled reply marshalled to %s, want %s", got, want)
	}
}
