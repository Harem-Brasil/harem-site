package services

import (
	"net/mail"
	"testing"

	"github.com/harem-brasil/backend/internal/utils"
)

func TestRegisterRequestValidation(t *testing.T) {
	tests := []struct {
		name       string
		email      string
		username   string
		password   string
		wantFields []string
	}{
		{
			name:       "valid request",
			email:      "test@example.com",
			username:   "testuser",
			password:   "SecurePass1!",
			wantFields: nil,
		},
		{
			name:       "weak password - too short",
			email:      "test@example.com",
			username:   "testuser",
			password:   "Ab1!",
			wantFields: []string{"password"},
		},
		{
			name:       "weak password - no uppercase",
			email:      "test@example.com",
			username:   "testuser",
			password:   "securepass1!",
			wantFields: []string{"password"},
		},
		{
			name:       "weak password - no special char",
			email:      "test@example.com",
			username:   "testuser",
			password:   "SecurePass1",
			wantFields: []string{"password"},
		},
		{
			name:       "missing email",
			email:      "",
			username:   "testuser",
			password:   "SecurePass1!",
			wantFields: []string{"email"},
		},
		{
			name:       "invalid email format",
			email:      "not-an-email",
			username:   "testuser",
			password:   "SecurePass1!",
			wantFields: []string{"email"},
		},
		{
			name:       "missing username",
			email:      "test@example.com",
			username:   "",
			password:   "SecurePass1!",
			wantFields: []string{"username"},
		},
		{
			name:       "missing password",
			email:      "test@example.com",
			username:   "testuser",
			password:   "",
			wantFields: []string{"password"},
		},
		{
			name:       "all fields missing",
			email:      "",
			username:   "",
			password:   "",
			wantFields: []string{"email", "username", "password"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := make(map[string]string)
			if tt.email == "" {
				errors["email"] = "Email is required"
			} else if _, err := mail.ParseAddress(tt.email); err != nil {
				errors["email"] = "Invalid email format"
			}
			if tt.username == "" {
				errors["username"] = "Username is required"
			}
			if tt.password == "" {
				errors["password"] = "Password is required"
			} else if msg := utils.ValidatePassword(tt.password); msg != "" {
				errors["password"] = msg
			}

			for _, field := range tt.wantFields {
				if _, ok := errors[field]; !ok {
					t.Errorf("expected validation error for field %q, got none", field)
				}
			}
			if tt.wantFields == nil && len(errors) > 0 {
				t.Errorf("expected no validation errors, got %v", errors)
			}
		})
	}
}

func TestLoginRequestValidation(t *testing.T) {
	tests := []struct {
		name       string
		email      string
		password   string
		wantFields []string
	}{
		{
			name:       "valid request",
			email:      "test@example.com",
			password:   "SecurePass1!",
			wantFields: nil,
		},
		{
			name:       "weak password - too short",
			email:      "test@example.com",
			password:   "Ab1!",
			wantFields: []string{"password"},
		},
		{
			name:       "weak password - no digit",
			email:      "test@example.com",
			password:   "SecurePass!",
			wantFields: []string{"password"},
		},
		{
			name:       "missing email",
			email:      "",
			password:   "SecurePass1!",
			wantFields: []string{"email"},
		},
		{
			name:       "invalid email format",
			email:      "not-an-email",
			password:   "SecurePass1!",
			wantFields: []string{"email"},
		},
		{
			name:       "missing password",
			email:      "test@example.com",
			password:   "",
			wantFields: []string{"password"},
		},
		{
			name:       "all fields missing",
			email:      "",
			password:   "",
			wantFields: []string{"email", "password"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := make(map[string]string)
			if tt.email == "" {
				errors["email"] = "Email is required"
			} else if _, err := mail.ParseAddress(tt.email); err != nil {
				errors["email"] = "Invalid email format"
			}
			if tt.password == "" {
				errors["password"] = "Password is required"
			} else if msg := utils.ValidatePassword(tt.password); msg != "" {
				errors["password"] = msg
			}

			for _, field := range tt.wantFields {
				if _, ok := errors[field]; !ok {
					t.Errorf("expected validation error for field %q, got none", field)
				}
			}
			if tt.wantFields == nil && len(errors) > 0 {
				t.Errorf("expected no validation errors, got %v", errors)
			}
		})
	}
}
