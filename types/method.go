package types

type Method struct {
	Name            string    `json:"name"`
	InputType       ProtoType `json:"input_type"`
	OutputType      ProtoType `json:"output_type"`
	ClientStreaming bool      `json:"client_streaming,omitempty"`
	ServerStreaming bool      `json:"server_streaming,omitempty"`
	Description     string    `json:"description,omitempty"`
}
