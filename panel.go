package contracts

type TodoItem struct {
	Text  string `json:"text"`
	State string `json:"state"`
}

type Subagent struct {
	ID    string `json:"id"`
	Name  string `json:"name,omitempty"`
	Kind  string `json:"kind,omitempty"`
	State string `json:"state"`
}
