package contracts

import (
	"encoding/json"
	"testing"
)

// resumableStub proves a type can satisfy ResumeAware.
type resumableStub struct{ tok string }

func (r resumableStub) ResumeToken() string { return r.tok }

func TestResumeAwareSatisfied(t *testing.T) {
	var _ ResumeAware = resumableStub{tok: "abc"}
}

func TestEventResumeJSON(t *testing.T) {
	b, err := json.Marshal(Event{T: "reply", Done: true, Resume: "sid-1"})
	if err != nil {
		t.Fatal(err)
	}
	if got := string(b); got != `{"t":"reply","done":true,"resume":"sid-1"}` {
		t.Fatalf("marshal: got %s", got)
	}
	// Empty Resume must be omitted.
	b, _ = json.Marshal(Event{T: "reply"})
	if got := string(b); got != `{"t":"reply"}` {
		t.Fatalf("omitempty: got %s", got)
	}
}
