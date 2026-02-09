package types

type Service struct {
	Name        string   `json:"name"`
	Methods     []Method `json:"methods"`
	Description string   `json:"description,omitempty"`
}
