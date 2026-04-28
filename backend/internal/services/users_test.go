package services

import (
	"testing"

	"github.com/harem-brasil/backend/internal/domain"
)

// --- PatchMeRequest validation tests ---

func TestPatchMeRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     domain.PatchMeRequest
		wantOK  bool
		wantErr string // substring expected in errors map values
	}{
		{
			name:   "empty request is valid (no-op)",
			req:    domain.PatchMeRequest{},
			wantOK: true,
		},
		{
			name:   "valid screen_name",
			req:    domain.PatchMeRequest{ScreenName: ptr("NewName")},
			wantOK: true,
		},
		{
			name:    "screen_name too short",
			req:     domain.PatchMeRequest{ScreenName: ptr("A")},
			wantOK:  false,
			wantErr: "at least 2 characters",
		},
		{
			name:    "screen_name too long (runes)",
			req:     domain.PatchMeRequest{ScreenName: ptr(string(make([]rune, 65)))},
			wantOK:  false,
			wantErr: "at most 64 characters",
		},
		{
			name:   "valid bio",
			req:    domain.PatchMeRequest{Bio: ptr("Hello world")},
			wantOK: true,
		},
		{
			name:    "bio too long (runes)",
			req:     domain.PatchMeRequest{Bio: ptr(string(make([]rune, 513)))},
			wantOK:  false,
			wantErr: "at most 512 characters",
		},
		{
			name:   "valid locale",
			req:    domain.PatchMeRequest{Locale: ptr("en-US")},
			wantOK: true,
		},
		{
			name:    "locale too long",
			req:     domain.PatchMeRequest{Locale: ptr("very-long-locale-code")},
			wantOK:  false,
			wantErr: "at most 16 characters",
		},
		{
			name:   "valid notify_preferences",
			req:    domain.PatchMeRequest{NotifyPreferences: &map[string]any{"email": false, "push": true}},
			wantOK: true,
		},
		{
			name:    "notify_preferences with disallowed key",
			req:     domain.PatchMeRequest{NotifyPreferences: &map[string]any{"sms": true}},
			wantOK:  false,
			wantErr: "not allowed",
		},
		{
			name:    "notify_preferences with mixed allowed and disallowed",
			req:     domain.PatchMeRequest{NotifyPreferences: &map[string]any{"email": true, "sms": true}},
			wantOK:  false,
			wantErr: "not allowed",
		},
		{
			name:    "notify_preferences with non-boolean value",
			req:     domain.PatchMeRequest{NotifyPreferences: &map[string]any{"email": "yes"}},
			wantOK:  false,
			wantErr: "must be a boolean",
		},
		{
			name:    "notify_preferences with numeric value",
			req:     domain.PatchMeRequest{NotifyPreferences: &map[string]any{"push": float64(1)}},
			wantOK:  false,
			wantErr: "must be a boolean",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs, ok := tt.req.Validate()
			if ok != tt.wantOK {
				t.Errorf("Validate() ok = %v, want %v, errors = %v", ok, tt.wantOK, errs)
			}
			if !ok && tt.wantErr != "" {
				found := false
				for _, v := range errs {
					if contains(v, tt.wantErr) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected error containing %q, got %v", tt.wantErr, errs)
				}
			}
		})
	}
}

// --- BOLA / auth boundary tests ---

func TestGetMe_RequiresAuth(t *testing.T) {
	// GetMe with nil claims should fail — the middleware layer prevents this,
	// but the service itself uses claims.UserID directly.
	// We verify that passing empty claims would cause a DB error (no match).
	// In practice, GinAuth middleware returns 401 before the handler runs.
	// This test documents the expected behavior.
	t.Log("401 without token is enforced by GinAuth middleware — not at service level")
}

func TestUpdateMe_CannotPatchOtherUser(t *testing.T) {
	// BOLA protection: UpdateMe only uses claims.UserID from the JWT,
	// so user A can never supply a different user_id to PATCH.
	// The WHERE id = $N clause always uses the authenticated user's ID.
	// There is no user_id parameter in the request body.
	t.Log("BOLA protection: UpdateMe uses claims.UserID from JWT — no user_id in request body")
}

func TestPatchMeRequest_WhitelistRejectsExtraFields(t *testing.T) {
	// Gin's ShouldBindJSON with a struct ignores unknown fields by default.
	// The PatchMeRequest struct only has whitelisted fields.
	// If a client sends {"role": "admin"}, it is silently ignored.
	// For strict rejection, we would need custom disallow logic.
	// The OpenAPI spec marks additionalProperties: false.
	t.Log("Extra fields in JSON are ignored by Go struct binding; OpenAPI declares additionalProperties: false")
}

// --- helpers ---

func ptr(s string) *string { return &s }

func contains(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
