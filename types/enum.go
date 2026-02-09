package types

type Enum struct {
	Name        string           `json:"name"`
	Values      map[string]int32 `json:"values"`
	Description string           `json:"description,omitempty"`
}
