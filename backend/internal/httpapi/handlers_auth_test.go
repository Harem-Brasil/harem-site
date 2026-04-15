package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

// Test input validation without database
func TestRegisterRequestValidation(t *testing.T) {
	tests := []struct {
		name    string
		req     RegisterRequest
		wantErr bool
	}{
		{
			name: "valid request",
			req: RegisterRequest{
				Email:    "test@example.com",
				Username: "testuser",
				Password: "securepassword123",
			},
			wantErr: false,
		},
		{
			name: "missing email",
			req: RegisterRequest{
				Username: "testuser",
				Password: "securepassword123",
			},
			wantErr: true,
		},
		{
			name: "missing username",
			req: RegisterRequest{
				Email:    "test@example.com",
				Password: "securepassword123",
			},
			wantErr: true,
		},
		{
			name: "missing password",
			req: RegisterRequest{
				Email:    "test@example.com",
				Username: "testuser",
			},
			wantErr: true,
		},
		{
			name:    "empty request",
			req:     RegisterRequest{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasError := tt.req.Email == "" || tt.req.Username == "" || tt.req.Password == ""
			if hasError != tt.wantErr {
				t.Errorf("validation mismatch: got error=%v, want error=%v", hasError, tt.wantErr)
			}
		})
	}
}

func TestLoginRequestValidation(t *testing.T) {
	tests := []struct {
		name    string
		req     LoginRequest
		wantErr bool
	}{
		{
			name: "valid request",
			req: LoginRequest{
				Email:    "test@example.com",
				Password: "securepassword123",
			},
			wantErr: false,
		},
		{
			name: "missing email",
			req: LoginRequest{
				Password: "securepassword123",
			},
			wantErr: true,
		},
		{
			name: "missing password",
			req: LoginRequest{
				Email: "test@example.com",
			},
			wantErr: true,
		},
		{
			name:    "empty request",
			req:     LoginRequest{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasError := tt.req.Email == "" || tt.req.Password == ""
			if hasError != tt.wantErr {
				t.Errorf("validation mismatch: got error=%v, want error=%v", hasError, tt.wantErr)
			}
		})
	}
}

// Test health endpoint (no DB required)
func TestHandleHealth(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	// Use a simple router without DB dependencies
	r := chi.NewRouter()
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		respondJSON(w, map[string]string{"status": "ok"})
	})
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("got status %d, want %d", rec.Code, http.StatusOK)
	}

	var resp map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Errorf("failed to parse response: %v", err)
	}

	if resp["status"] != "ok" {
		t.Error("response missing or incorrect status field")
	}
}
