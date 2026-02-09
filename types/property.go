package types

type Property struct {
	Name        string    `json:"name"`
	Type        ProtoType `json:"type"`
	IsRepeated  bool      `json:"is_repeated"`
	Description string    `json:"description,omitempty"`
}
