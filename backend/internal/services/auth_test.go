package services

import (
	"testing"

	"github.com/harem-brasil/backend/internal/domain"
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
