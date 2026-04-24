package utils

import "testing"

func TestValidatePassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{"valid password", "SecurePass1!", false},
		{"valid complex", "MyP@ssw0rd", false},
		{"too short", "Ab1!", true},
		{"no uppercase", "securepass1!", true},
		{"no lowercase", "SECUREPASS1!", true},
		{"no digit", "SecurePass!", true},
		{"no special", "SecurePass1", true},
		{"empty", "", true},
		{"exactly 8 chars valid", "Abcde1!a", false},
		{"exactly 7 chars", "Abcde1!", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := ValidatePassword(tt.password)
			if tt.wantErr && msg == "" {
				t.Errorf("expected validation error, got none")
			}
			if !tt.wantErr && msg != "" {
				t.Errorf("expected no error, got: %s", msg)
			}
		})
	}
}
