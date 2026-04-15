package httpapi

import (
	"encoding/json"
	"net/http"
	"time"
)

type ProblemDetail struct {
	Type       string         `json:"type,omitempty"`
	Title      string         `json:"title"`
	Status     int            `json:"status"`
	Detail     string         `json:"detail,omitempty"`
	Instance   string         `json:"instance,omitempty"`
	Extensions map[string]any `json:"extensions,omitempty"`
}

type CursorPage struct {
	Data       []any  `json:"data"`
	NextCursor string `json:"next_cursor,omitempty"`
	HasMore    bool   `json:"has_more"`
}

func respondJSON(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(data)
}

func respondProblem(w http.ResponseWriter, status int, title, detail string) {
	w.Header().Set("Content-Type", "application/problem+json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(ProblemDetail{
		Type:     "about:blank",
		Title:    title,
		Status:   status,
		Detail:   detail,
		Instance: "",
	})
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondProblem(w, status, http.StatusText(status), message)
}

func respondValidationError(w http.ResponseWriter, fieldErrors map[string]string) {
	w.Header().Set("Content-Type", "application/problem+json; charset=utf-8")
	w.WriteHeader(http.StatusUnprocessableEntity)
	json.NewEncoder(w).Encode(ProblemDetail{
		Type:       "validation-error",
		Title:      "Validation Error",
		Status:     422,
		Detail:     "One or more fields failed validation",
		Extensions: map[string]any{"fields": fieldErrors},
	})
}

func respondCreated(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(data)
}

func respondNoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

func formatTimestamp(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}
