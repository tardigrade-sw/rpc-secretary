package types

type Documentation struct {
	Services []Service          `json:"services"`
	Messages map[string]Message `json:"messages"`
	Enums    map[string]Enum    `json:"enums"`
}
