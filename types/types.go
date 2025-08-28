package types

type Res struct {
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
	Count   int    `json:"count,omitempty"`
}
