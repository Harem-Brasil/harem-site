package utils

import "unicode"

const (
	minPasswordLength = 8
)

// ValidatePassword checks password complexity requirements:
// minimum 8 chars, at least 1 lowercase, 1 uppercase, 1 digit, 1 special char.
// Returns a human-readable error message, or empty string if valid.
func ValidatePassword(password string) string {
	if len(password) < minPasswordLength {
		return "Password must be at least 8 characters long"
	}

	var hasLower, hasUpper, hasDigit, hasSpecial bool
	for _, r := range password {
		switch {
		case unicode.IsLower(r):
			hasLower = true
		case unicode.IsUpper(r):
			hasUpper = true
		case unicode.IsDigit(r):
			hasDigit = true
		case unicode.IsPunct(r) || unicode.IsSymbol(r):
			hasSpecial = true
		}
	}

	if !hasLower {
		return "Password must contain at least one lowercase letter"
	}
	if !hasUpper {
		return "Password must contain at least one uppercase letter"
	}
	if !hasDigit {
		return "Password must contain at least one number"
	}
	if !hasSpecial {
		return "Password must contain at least one special character"
	}

	return ""
}
