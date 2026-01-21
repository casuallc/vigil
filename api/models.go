package api

type WSMessage struct {
  Type string `json:"type"` // "input" | "resize"
  Data []byte `json:"data,omitempty"`

  Cols int `json:"cols,omitempty"`
  Rows int `json:"rows,omitempty"`
}
