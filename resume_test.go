package contracts

import (
	"encoding/json"
	"testing"
)

type resumableStub struct{ tok string }

func (r resumableStub) ResumeToken() string { return r.tok }

var _ ResumeAware = resumableStub{tok: "abc"}

func TestEventResumeJSON(t *testing.T) {
	b, err := json.Marshal(Event{T: "reply", Done: true, Resume: "sid-1"})
	if err != nil {
		t.Fatal(err)
	}
	if got := string(b); got != `{"t":"reply","done":true,"resume":"sid-1"}` {
		t.Fatalf("marshal: got %s", got)
	}
	b, err = json.Marshal(Event{T: "reply"})
	if err != nil {
		t.Fatal(err)
	}
	if got := string(b); got != `{"t":"reply"}` {
		t.Fatalf("omitempty: got %s", got)
	}
}
