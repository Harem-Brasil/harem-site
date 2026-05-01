package domain

import "encoding/json"

// ProblemDetail segue RFC 7807 (Problem Details for HTTP APIs).
type ProblemDetail struct {
	Type       string         `json:"type,omitempty"`
	Title      string         `json:"title"`
	Status     int            `json:"status"`
	Detail     string         `json:"detail,omitempty"`
	Instance   string         `json:"instance,omitempty"`
	Extensions map[string]any `json:"extensions,omitempty"`
}

// CursorPage paginação por cursor opaca.
type CursorPage struct {
	Data       []any  `json:"data"`
	NextCursor string `json:"next_cursor,omitempty"`
	HasMore    bool   `json:"has_more"`
}

// MarshalJSON serializa data sempre como array JSON ([]), nunca null.
func (p CursorPage) MarshalJSON() ([]byte, error) {
	type out struct {
		Data       []any  `json:"data"`
		NextCursor string `json:"next_cursor,omitempty"`
		HasMore    bool   `json:"has_more"`
	}
	data := p.Data
	if data == nil {
		data = []any{}
	}
	return json.Marshal(out{
		Data:       data,
		NextCursor: p.NextCursor,
		HasMore:    p.HasMore,
	})
}
