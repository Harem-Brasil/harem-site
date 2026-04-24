package domain

import "testing"

func TestRegisterRequestValidate(t *testing.T) {
	tests := []struct {
		name       string
		req        RegisterRequest
		wantFields []string
		wantOk     bool
	}{
		{"valid", RegisterRequest{Email: "a@b.com", Username: "user", Password: "Secure1!"}, nil, true},
		{"missing email", RegisterRequest{Username: "user", Password: "Secure1!"}, []string{"email"}, false},
		{"invalid email", RegisterRequest{Email: "nope", Username: "user", Password: "Secure1!"}, []string{"email"}, false},
		{"missing username", RegisterRequest{Email: "a@b.com", Password: "Secure1!"}, []string{"username"}, false},
		{"missing password", RegisterRequest{Email: "a@b.com", Username: "user"}, []string{"password"}, false},
		{"weak password", RegisterRequest{Email: "a@b.com", Username: "user", Password: "abc"}, []string{"password"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs, ok := tt.req.Validate()
			if ok != tt.wantOk {
				t.Errorf("Validate() ok = %v, want %v", ok, tt.wantOk)
			}
			for _, f := range tt.wantFields {
				if _, exists := errs[f]; !exists {
					t.Errorf("expected error for field %q, got none", f)
				}
			}
		})
	}
}

func TestLoginRequestValidate(t *testing.T) {
	tests := []struct {
		name       string
		req        LoginRequest
		wantFields []string
		wantOk     bool
	}{
		{"valid", LoginRequest{Email: "a@b.com", Password: "Secure1!"}, nil, true},
		{"missing email", LoginRequest{Password: "Secure1!"}, []string{"email"}, false},
		{"invalid email", LoginRequest{Email: "nope", Password: "Secure1!"}, []string{"email"}, false},
		{"missing password", LoginRequest{Email: "a@b.com"}, []string{"password"}, false},
		{"weak password", LoginRequest{Email: "a@b.com", Password: "abc"}, []string{"password"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs, ok := tt.req.Validate()
			if ok != tt.wantOk {
				t.Errorf("Validate() ok = %v, want %v", ok, tt.wantOk)
			}
			for _, f := range tt.wantFields {
				if _, exists := errs[f]; !exists {
					t.Errorf("expected error for field %q, got none", f)
				}
			}
		})
	}
}

func TestValidatePassword(t *testing.T) {
	tests := []struct {
		password string
		wantErr  bool
	}{
		{"SecurePass1!", false},
		{"MyP@ssw0rd", false},
		{"Abcde1!a", false}, // exactly 8 chars
		{"", true},
		{"abc", true},
		{"nouppercase1!", true},
		{"NOLOWERCASE1!", true},
		{"NoDigits!", true},
		{"NoSpecial1", true},
	}
	for _, tt := range tests {
		t.Run(tt.password, func(t *testing.T) {
			msg := validatePassword(tt.password)
			if tt.wantErr && msg == "" {
				t.Error("expected error, got none")
			}
			if !tt.wantErr && msg != "" {
				t.Errorf("unexpected error: %s", msg)
			}
		})
	}
}
