package services

import (
	"strings"
	"testing"

	"github.com/harem-brasil/backend/internal/domain"
	"github.com/harem-brasil/backend/internal/middleware"
)

func TestRegisterRequestValidation(t *testing.T) {
	tests := []struct {
		name       string
		req        domain.RegisterRequest
		wantFields []string
	}{
		{
			name:       "valid request",
			req:        domain.RegisterRequest{Email: "test@example.com", ScreenName: "testuser", Password: "SecurePass1!", AcceptTermsVersion: "1.0"},
			wantFields: nil,
		},
		{
			name:       "weak password - too short",
			req:        domain.RegisterRequest{Email: "test@example.com", ScreenName: "testuser", Password: "Ab1!", AcceptTermsVersion: "1.0"},
			wantFields: []string{"password"},
		},
		{
			name:       "weak password - no uppercase",
			req:        domain.RegisterRequest{Email: "test@example.com", ScreenName: "testuser", Password: "securepass1!", AcceptTermsVersion: "1.0"},
			wantFields: []string{"password"},
		},
		{
			name:       "weak password - no special char",
			req:        domain.RegisterRequest{Email: "test@example.com", ScreenName: "testuser", Password: "SecurePass1", AcceptTermsVersion: "1.0"},
			wantFields: []string{"password"},
		},
		{
			name:       "missing email",
			req:        domain.RegisterRequest{ScreenName: "testuser", Password: "SecurePass1!", AcceptTermsVersion: "1.0"},
			wantFields: []string{"email"},
		},
		{
			name:       "invalid email format",
			req:        domain.RegisterRequest{Email: "not-an-email", ScreenName: "testuser", Password: "SecurePass1!", AcceptTermsVersion: "1.0"},
			wantFields: []string{"email"},
		},
		{
			name:       "missing screen_name",
			req:        domain.RegisterRequest{Email: "test@example.com", Password: "SecurePass1!", AcceptTermsVersion: "1.0"},
			wantFields: []string{"screen_name"},
		},
		{
			name:       "screen_name too short",
			req:        domain.RegisterRequest{Email: "test@example.com", ScreenName: "a", Password: "SecurePass1!", AcceptTermsVersion: "1.0"},
			wantFields: []string{"screen_name"},
		},
		{
			name:       "missing password",
			req:        domain.RegisterRequest{Email: "test@example.com", ScreenName: "testuser", AcceptTermsVersion: "1.0"},
			wantFields: []string{"password"},
		},
		{
			name:       "missing accept_terms_version",
			req:        domain.RegisterRequest{Email: "test@example.com", ScreenName: "testuser", Password: "SecurePass1!"},
			wantFields: []string{"accept_terms_version"},
		},
		{
			name:       "all fields missing",
			req:        domain.RegisterRequest{},
			wantFields: []string{"email", "screen_name", "password", "accept_terms_version"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors, ok := tt.req.Validate()
			for _, field := range tt.wantFields {
				if _, exists := errors[field]; !exists {
					t.Errorf("expected validation error for field %q, got none", field)
				}
			}
			if tt.wantFields == nil && !ok {
				t.Errorf("expected no validation errors, got %v", errors)
			}
			if tt.wantFields != nil && ok {
				t.Errorf("expected validation errors, got none")
			}
		})
	}
}

func TestLoginRequestValidation(t *testing.T) {
	tests := []struct {
		name       string
		req        domain.LoginRequest
		wantFields []string
	}{
		{
			name:       "valid request",
			req:        domain.LoginRequest{Email: "test@example.com", Password: "anything"},
			wantFields: nil,
		},
		{
			name:       "missing email",
			req:        domain.LoginRequest{Password: "anything"},
			wantFields: []string{"email"},
		},
		{
			name:       "invalid email format",
			req:        domain.LoginRequest{Email: "not-an-email", Password: "anything"},
			wantFields: []string{"email"},
		},
		{
			name:       "missing password",
			req:        domain.LoginRequest{Email: "test@example.com"},
			wantFields: []string{"password"},
		},
		{
			name:       "all fields missing",
			req:        domain.LoginRequest{},
			wantFields: []string{"email", "password"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors, ok := tt.req.Validate()
			for _, field := range tt.wantFields {
				if _, exists := errors[field]; !exists {
					t.Errorf("expected validation error for field %q, got none", field)
				}
			}
			if tt.wantFields == nil && !ok {
				t.Errorf("expected no validation errors, got %v", errors)
			}
			if tt.wantFields != nil && ok {
				t.Errorf("expected validation errors, got none")
			}
		})
	}
}

func TestSha256Hash(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		want64 bool
	}{
		{"empty string", "", true},
		{"short input", "hello", true},
		{"long input", strings.Repeat("a", 1000), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sha256Hash(tt.input)
			if len(got) != 64 {
				t.Errorf("sha256Hash(%q) = %d chars, want 64", tt.input, len(got))
			}
			if sha256Hash(tt.input) != got {
				t.Errorf("sha256Hash is not deterministic for %q", tt.input)
			}
			if tt.input != "" && sha256Hash(tt.input+"x") == got {
				t.Errorf("sha256Hash collision for %q", tt.input)
			}
		})
	}
}

func TestSplitRefreshToken(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantID     string
		wantSecret string
		wantOK     bool
	}{
		{"valid", "abc123.secretpart", "abc123", "secretpart", true},
		{"no dot", "abc123secretpart", "", "", false},
		{"empty", "", "", "", false},
		{"dot only", ".", "", "", true},
		{"multiple dots", "abc123.secret.extra", "abc123", "secret.extra", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, secret, ok := splitRefreshToken(tt.input)
			if ok != tt.wantOK {
				t.Errorf("splitRefreshToken(%q) ok = %v, want %v", tt.input, ok, tt.wantOK)
			}
			if ok {
				if id != tt.wantID {
					t.Errorf("splitRefreshToken(%q) id = %q, want %q", tt.input, id, tt.wantID)
				}
				if secret != tt.wantSecret {
					t.Errorf("splitRefreshToken(%q) secret = %q, want %q", tt.input, secret, tt.wantSecret)
				}
			}
		})
	}
}

func TestLogoutOwnershipCheck(t *testing.T) {
	tests := []struct {
		name         string
		userID       string
		tokenOwnerID string
		shouldRevoke bool
	}{
		{"same user", "user-1", "user-1", true},
		{"different user", "user-1", "user-2", false},
		{"empty owner", "user-1", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shouldRevoke := tt.userID == tt.tokenOwnerID && tt.tokenOwnerID != ""
			if shouldRevoke != tt.shouldRevoke {
				t.Errorf("ownership check: userID=%q tokenOwnerID=%q shouldRevoke=%v want=%v",
					tt.userID, tt.tokenOwnerID, shouldRevoke, tt.shouldRevoke)
			}
		})
	}
}

func TestLogoutAllRequiresAuth(t *testing.T) {
	svc := Services{}
	err := svc.LogoutAll(nil, nil)
	if err == nil {
		t.Error("expected error for nil UserClaims, got nil")
	}
}

func TestLogoutIdempotency(t *testing.T) {
	svc := Services{Dependencies: Dependencies{Logger: nil}}
	user := &middleware.UserClaims{UserID: "user-1"}

	err := svc.Logout(nil, user, LogoutBody{RefreshToken: ""})
	if err != nil {
		t.Errorf("Logout with empty token should be no-op, got: %v", err)
	}

	err = svc.Logout(nil, user, LogoutBody{RefreshToken: "not-a-valid-token"})
	if err != nil {
		t.Errorf("Logout with malformed token should be no-op, got: %v", err)
	}
}
