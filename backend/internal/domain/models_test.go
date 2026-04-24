package domain

import "testing"

func TestRegisterRequestValidate(t *testing.T) {
	tests := []struct {
		name       string
		req        RegisterRequest
		wantFields []string
		wantOk     bool
	}{
		{"valid", RegisterRequest{Email: "a@b.com", ScreenName: "user", Password: "Secure1!", AcceptTermsVersion: "1.0"}, nil, true},
		{"missing email", RegisterRequest{ScreenName: "user", Password: "Secure1!", AcceptTermsVersion: "1.0"}, []string{"email"}, false},
		{"invalid email", RegisterRequest{Email: "nope", ScreenName: "user", Password: "Secure1!", AcceptTermsVersion: "1.0"}, []string{"email"}, false},
		{"missing screen_name", RegisterRequest{Email: "a@b.com", Password: "Secure1!", AcceptTermsVersion: "1.0"}, []string{"screen_name"}, false},
		{"screen_name too short", RegisterRequest{Email: "a@b.com", ScreenName: "a", Password: "Secure1!", AcceptTermsVersion: "1.0"}, []string{"screen_name"}, false},
		{"missing password", RegisterRequest{Email: "a@b.com", ScreenName: "user", AcceptTermsVersion: "1.0"}, []string{"password"}, false},
		{"weak password", RegisterRequest{Email: "a@b.com", ScreenName: "user", Password: "abc", AcceptTermsVersion: "1.0"}, []string{"password"}, false},
		{"missing accept_terms_version", RegisterRequest{Email: "a@b.com", ScreenName: "user", Password: "Secure1!"}, []string{"accept_terms_version"}, false},
		{"all fields missing", RegisterRequest{}, []string{"email", "screen_name", "password", "accept_terms_version"}, false},
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
		{"valid", LoginRequest{Email: "a@b.com", Password: "anything"}, nil, true},
		{"missing email", LoginRequest{Password: "anything"}, []string{"email"}, false},
		{"invalid email", LoginRequest{Email: "nope", Password: "anything"}, []string{"email"}, false},
		{"missing password", LoginRequest{Email: "a@b.com"}, []string{"password"}, false},
		{"all fields missing", LoginRequest{}, []string{"email", "password"}, false},
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

func TestValidateScreenName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid", "user123", false},
		{"minimum length", "ab", false},
		{"too short", "a", true},
		{"control char", "user\x00name", true},
		{"del char", "user\x7Fname", true},
		{"zero-width space", "user\u200Bname", true},
		{"space", "user name", true},
		{"tab", "user\tname", true},
		{"leading space", " username", true},
		{"trailing space", "username ", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := validateScreenName(tt.input)
			if tt.wantErr && msg == "" {
				t.Error("expected error, got none")
			}
			if !tt.wantErr && msg != "" {
				t.Errorf("unexpected error: %s", msg)
			}
		})
	}
}
