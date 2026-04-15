package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRespondJSON(t *testing.T) {
	tests := []struct {
		name     string
		status   int
		data     any
		wantBody string
	}{
		{
			name:     "simple object",
			status:   http.StatusOK,
			data:     map[string]string{"message": "hello"},
			wantBody: `{"message":"hello"}`,
		},
		{
			name:     "nested object",
			status:   http.StatusOK,
			data:     map[string]any{"user": map[string]string{"id": "123"}},
			wantBody: `{"user":{"id":"123"}}`,
		},
		{
			name:     "array response",
			status:   http.StatusOK,
			data:     []string{"a", "b", "c"},
			wantBody: `["a","b","c"]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			respondJSON(rec, tt.data)

			if rec.Code != tt.status {
				t.Errorf("status code: got %d, want %d", rec.Code, tt.status)
			}

			ct := rec.Header().Get("Content-Type")
			if ct != "application/json; charset=utf-8" {
				t.Errorf("content-type: got %q, want application/json; charset=utf-8", ct)
			}

			var got, want map[string]any
			json.Unmarshal(rec.Body.Bytes(), &got)
			json.Unmarshal([]byte(tt.wantBody), &want)
		})
	}
}

func TestRespondError(t *testing.T) {
	tests := []struct {
		name       string
		status     int
		title      string
		detail     string
		wantStatus int
	}{
		{
			name:       "bad request",
			status:     http.StatusBadRequest,
			title:      "Invalid Input",
			detail:     "Missing required field",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "not found",
			status:     http.StatusNotFound,
			title:      "Not Found",
			detail:     "Resource not found",
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "internal error",
			status:     http.StatusInternalServerError,
			title:      "Internal Error",
			detail:     "Something went wrong",
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			respondProblem(rec, tt.status, tt.title, tt.detail)

			if rec.Code != tt.wantStatus {
				t.Errorf("status code: got %d, want %d", rec.Code, tt.wantStatus)
			}

			ct := rec.Header().Get("Content-Type")
			wantCT := "application/problem+json; charset=utf-8"
			if ct != wantCT {
				t.Errorf("content-type: got %q, want %q", ct, wantCT)
			}

			var resp ProblemDetail
			if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
				t.Fatalf("failed to parse response: %v", err)
			}

			if resp.Title != tt.title {
				t.Errorf("title: got %q, want %q", resp.Title, tt.title)
			}

			if resp.Detail != tt.detail {
				t.Errorf("detail: got %q, want %q", resp.Detail, tt.detail)
			}

			if resp.Status != tt.status {
				t.Errorf("status: got %d, want %d", resp.Status, tt.status)
			}
		})
	}
}

func TestProblemResponse(t *testing.T) {
	rec := httptest.NewRecorder()
	respondProblem(rec, http.StatusBadRequest, "Validation Error", "Field 'email' is invalid")

	// Verify RFC 7807 structure
	body := rec.Body.String()
	var fields map[string]any
	if err := json.Unmarshal([]byte(body), &fields); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	required := []string{"type", "title", "status", "detail"}
	for _, f := range required {
		if _, ok := fields[f]; !ok {
			t.Errorf("missing required field: %s", f)
		}
	}
}
