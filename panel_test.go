package contracts

import (
	"encoding/json"
	"testing"
)

func TestTodoItemRoundTrips(t *testing.T) {
	in := []TodoItem{{Text: "lire le contrat", State: "done"}, {Text: "ecrire le test", State: "active"}}
	b, err := json.Marshal(in)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != `[{"text":"lire le contrat","state":"done"},{"text":"ecrire le test","state":"active"}]` {
		t.Fatalf("todo encoding = %s", b)
	}
	var out []TodoItem
	if err := json.Unmarshal(b, &out); err != nil || len(out) != 2 || out[1].State != "active" {
		t.Fatalf("todo decode = %+v (%v)", out, err)
	}
}

func TestSubagentOmitsWhatItDoesNotCarry(t *testing.T) {
	b, err := json.Marshal(Subagent{ID: "toolu_1", State: "done"})
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != `{"id":"toolu_1","state":"done"}` {
		t.Fatalf("subagent encoding = %s", b)
	}
}

func TestEventCarriesPanels(t *testing.T) {
	b, err := json.Marshal(Event{T: "todos", Todos: []TodoItem{{Text: "x", State: "pending"}}})
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != `{"t":"todos","todos":[{"text":"x","state":"pending"}]}` {
		t.Fatalf("event encoding = %s", b)
	}
}
