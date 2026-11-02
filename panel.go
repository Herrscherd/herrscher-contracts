package contracts

// TodoItem is one entry of a session's task list, with its text and its state.
type TodoItem struct {
	Text  string `json:"text"`
	State string `json:"state"`
}

// Subagent is a child agent working for a session. Name and Kind are optional labels,
// and State is its current state.
type Subagent struct {
	ID    string `json:"id"`
	Name  string `json:"name,omitempty"`
	Kind  string `json:"kind,omitempty"`
	State string `json:"state"`
}
