package types

type Message struct {
	Name        string     `json:"name"`
	Properties  []Property `json:"properties"`
	Description string     `json:"description,omitempty"`
}
