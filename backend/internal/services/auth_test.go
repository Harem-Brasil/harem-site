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
			req:        domain.RegisterRequest{Email: "test@example.com", Username: "testuser", Password: "SecurePass1!"},
			wantFields: nil,
		},
		{
			name:       "weak password - too short",
			req:        domain.RegisterRequest{Email: "test@example.com", Username: "testuser", Password: "Ab1!"},
			wantFields: []string{"password"},
		},
		{
			name:       "weak password - no uppercase",
			req:        domain.RegisterRequest{Email: "test@example.com", Username: "testuser", Password: "securepass1!"},
			wantFields: []string{"password"},
		},
		{
			name:       "weak password - no special char",
			req:        domain.RegisterRequest{Email: "test@example.com", Username: "testuser", Password: "SecurePass1"},
			wantFields: []string{"password"},
		},
		{
			name:       "missing email",
			req:        domain.RegisterRequest{Email: "", Username: "testuser", Password: "SecurePass1!"},
			wantFields: []string{"email"},
		},
		{
			name:       "invalid email format",
			req:        domain.RegisterRequest{Email: "not-an-email", Username: "testuser", Password: "SecurePass1!"},
			wantFields: []string{"email"},
		},
		{
			name:       "missing username",
			req:        domain.RegisterRequest{Email: "test@example.com", Username: "", Password: "SecurePass1!"},
			wantFields: []string{"username"},
		},
		{
			name:       "missing password",
			req:        domain.RegisterRequest{Email: "test@example.com", Username: "testuser", Password: ""},
			wantFields: []string{"password"},
		},
		{
			name:       "all fields missing",
			req:        domain.RegisterRequest{},
			wantFields: []string{"email", "username", "password"},
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
			req:        domain.LoginRequest{Email: "test@example.com", Password: "SecurePass1!"},
			wantFields: nil,
		},
		{
			name:       "weak password - too short",
			req:        domain.LoginRequest{Email: "test@example.com", Password: "Ab1!"},
			wantFields: []string{"password"},
		},
		{
			name:       "weak password - no digit",
			req:        domain.LoginRequest{Email: "test@example.com", Password: "SecurePass!"},
			wantFields: []string{"password"},
		},
		{
			name:       "missing email",
			req:        domain.LoginRequest{Email: "", Password: "SecurePass1!"},
			wantFields: []string{"email"},
		},
		{
			name:       "invalid email format",
			req:        domain.LoginRequest{Email: "not-an-email", Password: "SecurePass1!"},
			wantFields: []string{"email"},
		},
		{
			name:       "missing password",
			req:        domain.LoginRequest{Email: "test@example.com", Password: ""},
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
