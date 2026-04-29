package services

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// --- PKCE helpers tests ---

func TestGenerateCodeVerifier(t *testing.T) {
	v, err := generateCodeVerifier()
	if err != nil {
		t.Fatalf("generateCodeVerifier() error: %v", err)
	}
	// RFC 7636: verifier length 43-128 chars, unreserved chars only
	if len(v) < 43 || len(v) > 128 {
		t.Errorf("verifier length = %d, want 43-128", len(v))
	}
	// Must be base64url (no padding)
	for _, c := range v {
		if !isUnreserved(c) {
			t.Errorf("verifier contains invalid char: %c", c)
		}
	}
}

func TestGenerateCodeVerifier_Uniqueness(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < 100; i++ {
		v, err := generateCodeVerifier()
		if err != nil {
			t.Fatalf("error on iteration %d: %v", i, err)
		}
		if seen[v] {
			t.Fatalf("duplicate verifier generated on iteration %d", i)
		}
		seen[v] = true
	}
}

func TestComputeCodeChallengeS256(t *testing.T) {
	// RFC 7636 Appendix B test vector
	verifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
	expected := "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM"

	challenge := computeCodeChallengeS256(verifier)
	if challenge != expected {
		t.Errorf("challenge = %q, want %q", challenge, expected)
	}
}

func TestComputeCodeChallengeS256_RoundTrip(t *testing.T) {
	verifier, err := generateCodeVerifier()
	if err != nil {
		t.Fatalf("generateCodeVerifier() error: %v", err)
	}
	challenge := computeCodeChallengeS256(verifier)

	// Manually verify: BASE64URL(SHA256(verifier))
	h := sha256.Sum256([]byte(verifier))
	manual := base64.RawURLEncoding.EncodeToString(h[:])
	if challenge != manual {
		t.Errorf("challenge = %q, manual = %q", challenge, manual)
	}
}

func TestGenerateState(t *testing.T) {
	s, err := generateState()
	if err != nil {
		t.Fatalf("generateState() error: %v", err)
	}
	if len(s) < 32 {
		t.Errorf("state too short: %d chars", len(s))
	}
}

func TestGenerateState_Uniqueness(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < 100; i++ {
		s, err := generateState()
		if err != nil {
			t.Fatalf("error on iteration %d: %v", i, err)
		}
		if seen[s] {
			t.Fatalf("duplicate state generated on iteration %d", i)
		}
		seen[s] = true
	}
}

// --- State format validation tests ---

func TestOAuthStateFormat(t *testing.T) {
	// State is "stateID.nonce" — both parts must be non-empty
	tests := []struct {
		name  string
		state string
		valid bool
	}{
		{"valid", "abc123.def456", true},
		{"missing nonce", "abc123.", false},
		{"missing stateID", ".def456", false},
		{"no dot separator", "abc123def456", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parts := strings.SplitN(tt.state, ".", 2)
			valid := len(parts) == 2 && parts[0] != "" && parts[1] != ""
			if valid != tt.valid {
				t.Errorf("state %q: valid = %v, want %v", tt.state, valid, tt.valid)
			}
		})
	}
}

// --- OAuth provider config tests ---

func TestOAuthProviderConfig_Validation(t *testing.T) {
	// Verify that OAuthAuthorize rejects unsupported providers
	svc := &Services{
		Dependencies: Dependencies{
			OAuthProviders: map[string]OAuthProviderConfig{
				"google": {
					ClientID:     "test-client-id",
					ClientSecret: "test-client-secret",
					AuthorizeURL: "https://accounts.google.com/o/oauth2/v2/auth",
					TokenURL:     "https://oauth2.googleapis.com/token",
					UserInfoURL:  "https://openidconnect.googleapis.com/v1/userinfo",
					Scopes:       []string{"openid", "email", "profile"},
				},
			},
		},
	}

	_, err := svc.OAuthAuthorize(context.TODO(), "github", "http://localhost:3000/callback")
	if err == nil {
		t.Error("expected error for unsupported provider, got nil")
	}
}

// --- Redirect URI allowlist tests ---

func TestValidateRedirectURI(t *testing.T) {
	tests := []struct {
		name        string
		allowlist   []string
		redirectURI string
		wantErr     bool
	}{
		{
			name:        "empty allowlist accepts all",
			allowlist:   nil,
			redirectURI: "http://evil.com/callback",
			wantErr:     false,
		},
		{
			name:        "uri in allowlist",
			allowlist:   []string{"http://localhost:3000/callback", "https://app.example.com/callback"},
			redirectURI: "http://localhost:3000/callback",
			wantErr:     false,
		},
		{
			name:        "uri not in allowlist",
			allowlist:   []string{"http://localhost:3000/callback"},
			redirectURI: "http://evil.com/callback",
			wantErr:     true,
		},
		{
			name:        "empty redirect_uri rejected",
			allowlist:   nil,
			redirectURI: "",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := OAuthProviderConfig{AllowedRedirectURIs: tt.allowlist}
			err := validateRedirectURI(cfg, tt.redirectURI)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateRedirectURI() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// --- ID token validation tests ---

func TestValidateIDToken(t *testing.T) {
	cfg := OAuthProviderConfig{
		ClientID:  "my-client-id",
		IssuerURL: "https://accounts.google.com",
	}

	t.Run("valid ID token", func(t *testing.T) {
		token := buildTestIDToken(t, "https://accounts.google.com", "my-client-id", "test-nonce", time.Now().Add(time.Hour).Unix())
		if err := validateIDToken(token, cfg, "test-nonce"); err != nil {
			t.Errorf("validateIDToken() unexpected error: %v", err)
		}
	})

	t.Run("wrong issuer", func(t *testing.T) {
		token := buildTestIDToken(t, "https://evil.com", "my-client-id", "test-nonce", time.Now().Add(time.Hour).Unix())
		if err := validateIDToken(token, cfg, "test-nonce"); err == nil {
			t.Error("expected error for wrong issuer, got nil")
		}
	})

	t.Run("wrong audience", func(t *testing.T) {
		token := buildTestIDToken(t, "https://accounts.google.com", "wrong-aud", "test-nonce", time.Now().Add(time.Hour).Unix())
		if err := validateIDToken(token, cfg, "test-nonce"); err == nil {
			t.Error("expected error for wrong audience, got nil")
		}
	})

	t.Run("wrong nonce", func(t *testing.T) {
		token := buildTestIDToken(t, "https://accounts.google.com", "my-client-id", "wrong-nonce", time.Now().Add(time.Hour).Unix())
		if err := validateIDToken(token, cfg, "test-nonce"); err == nil {
			t.Error("expected error for wrong nonce, got nil")
		}
	})

	t.Run("expired token", func(t *testing.T) {
		token := buildTestIDToken(t, "https://accounts.google.com", "my-client-id", "test-nonce", time.Now().Add(-time.Hour).Unix())
		if err := validateIDToken(token, cfg, "test-nonce"); err == nil {
			t.Error("expected error for expired token, got nil")
		}
	})

	t.Run("malformed token", func(t *testing.T) {
		if err := validateIDToken("not-a-jwt", cfg, ""); err == nil {
			t.Error("expected error for malformed token, got nil")
		}
	})

	t.Run("empty IssuerURL skips iss check", func(t *testing.T) {
		cfgNoIss := OAuthProviderConfig{ClientID: "my-client-id", IssuerURL: ""}
		token := buildTestIDToken(t, "https://any-issuer.com", "my-client-id", "nonce1", time.Now().Add(time.Hour).Unix())
		if err := validateIDToken(token, cfgNoIss, "nonce1"); err != nil {
			t.Errorf("validateIDToken() with empty IssuerURL unexpected error: %v", err)
		}
	})
}

// buildTestIDToken creates a minimal JWT with the given claims for testing.
// The signature is a dummy (not verified by validateIDToken).
func buildTestIDToken(t *testing.T, iss, aud, nonce string, exp int64) string {
	t.Helper()

	header := map[string]any{"alg": "RS256", "typ": "JWT"}
	payload := idTokenClaims{Iss: iss, Aud: aud, Nonce: nonce, Exp: exp, Sub: "12345"}

	headerJSON, _ := json.Marshal(header)
	payloadJSON, _ := json.Marshal(payload)

	headerB64 := base64.RawURLEncoding.EncodeToString(headerJSON)
	payloadB64 := base64.RawURLEncoding.EncodeToString(payloadJSON)
	sigB64 := base64.RawURLEncoding.EncodeToString([]byte("dummy-signature"))

	return headerB64 + "." + payloadB64 + "." + sigB64
}

// --- helper ---

func isUnreserved(r rune) bool {
	return (r >= 'A' && r <= 'Z') ||
		(r >= 'a' && r <= 'z') ||
		(r >= '0' && r <= '9') ||
		r == '-' || r == '.' || r == '_' || r == '~'
}
