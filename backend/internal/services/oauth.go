package services

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/harem-brasil/backend/internal/domain"
	"github.com/harem-brasil/backend/internal/utils"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// OAuthAuthorizeResult is returned by OAuthAuthorize — the caller must
// redirect the user to AuthorizeURL and set State as a cookie (CSRF).
type OAuthAuthorizeResult struct {
	AuthorizeURL string // full URL to redirect the user to
	State        string // opaque state value — must be echoed back in callback
}

// OAuthCallbackResult is returned by OAuthCallback on success.
type OAuthCallbackResult struct {
	AuthResponse *domain.AuthResponse
	IsNewUser    bool // true if this was a first-time login (user created)
}

// OAuthUserInfo represents the normalized identity from the provider.
type OAuthUserInfo struct {
	Subject     string
	Email       string
	DisplayName string
	AvatarURL   string
}

// OAuthAuthorize initiates an Authorization Code + PKCE flow for the given provider.
// It generates a code_verifier, stores it with a state nonce in oauth_states,
// and returns the provider's authorize URL with code_challenge (S256).
func (s *Services) OAuthAuthorize(ctx context.Context, provider string, redirectURI string) (*OAuthAuthorizeResult, error) {
	cfg, ok := s.OAuthProviders[provider]
	if !ok {
		return nil, domain.Err(400, fmt.Sprintf("Unsupported OAuth provider: %s", provider))
	}

	// Validate redirect_uri against allowlist
	if err := validateRedirectURI(cfg, redirectURI); err != nil {
		return nil, domain.Err(400, err.Error())
	}

	// Generate PKCE code_verifier (43-128 chars, RFC 7636)
	codeVerifier, err := generateCodeVerifier()
	if err != nil {
		return nil, domain.Err(500, "Failed to generate PKCE verifier")
	}

	// Generate state nonce (CSRF protection)
	nonce, err := generateState()
	if err != nil {
		return nil, domain.Err(500, "Failed to generate state")
	}

	// Store state + code_verifier + nonce in DB
	stateID := uuid.New().String()
	_, err = s.DB.Exec(ctx,
		`INSERT INTO oauth_states (id, provider, code_verifier, nonce, redirect_uri, created_at, expires_at)
		 VALUES ($1, $2, $3, $4, $5, NOW(), NOW() + INTERVAL '10 minutes')`,
		stateID, provider, codeVerifier, nonce, redirectURI,
	)
	if err != nil {
		return nil, domain.Err(500, "Failed to store OAuth state")
	}

	// Compute code_challenge = BASE64URL(SHA256(code_verifier))
	codeChallenge := computeCodeChallengeS256(codeVerifier)

	// Build authorize URL
	params := url.Values{}
	params.Set("client_id", cfg.ClientID)
	params.Set("redirect_uri", redirectURI)
	params.Set("response_type", "code")
	params.Set("scope", strings.Join(cfg.Scopes, " "))
	params.Set("state", stateID+"."+nonce) // stateID for DB lookup + nonce for CSRF
	params.Set("code_challenge", codeChallenge)
	params.Set("code_challenge_method", "S256")

	authorizeURL := cfg.AuthorizeURL + "?" + params.Encode()

	return &OAuthAuthorizeResult{
		AuthorizeURL: authorizeURL,
		State:        stateID + "." + nonce,
	}, nil
}

// OAuthCallback completes the Authorization Code + PKCE flow.
// It exchanges the code for tokens, fetches user info, and creates/links the user.
func (s *Services) OAuthCallback(ctx context.Context, provider, stateParam, code, redirectURI string, meta *SessionMeta) (*OAuthCallbackResult, error) {
	cfg, ok := s.OAuthProviders[provider]
	if !ok {
		return nil, domain.Err(400, fmt.Sprintf("Unsupported OAuth provider: %s", provider))
	}

	// Parse and validate state (format: stateID.nonce)
	parts := strings.SplitN(stateParam, ".", 2)
	if len(parts) != 2 {
		return nil, domain.Err(400, "Invalid state parameter")
	}
	stateID, nonce := parts[0], parts[1]

	// Look up state in DB
	var stored struct {
		Provider     string
		CodeVerifier string
		Nonce        string
		RedirectURI  string
		ExpiresAt    time.Time
	}
	err := s.DB.QueryRow(ctx,
		`SELECT provider, code_verifier, nonce, redirect_uri, expires_at FROM oauth_states WHERE id = $1`,
		stateID,
	).Scan(&stored.Provider, &stored.CodeVerifier, &stored.Nonce, &stored.RedirectURI, &stored.ExpiresAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.Err(400, "Invalid or expired OAuth state")
		}
		return nil, domain.Err(500, "Database error")
	}

	// Delete state (single-use)
	_, _ = s.DB.Exec(ctx, `DELETE FROM oauth_states WHERE id = $1`, stateID)

	// Validate state
	if stored.Provider != provider {
		return nil, domain.Err(400, "State provider mismatch")
	}
	if stored.Nonce != nonce {
		return nil, domain.Err(400, "State nonce mismatch")
	}
	if stored.RedirectURI != redirectURI {
		return nil, domain.Err(400, "State redirect_uri mismatch")
	}
	if time.Now().UTC().After(stored.ExpiresAt) {
		return nil, domain.Err(400, "OAuth state expired")
	}

	// Exchange code for tokens
	tokenResp, err := exchangeCodeForTokens(ctx, cfg, code, stored.CodeVerifier, redirectURI)
	if err != nil {
		if s.Logger != nil {
			s.Logger.Error("OAuth token exchange failed", "provider", provider, "error", err)
		}
		return nil, domain.Err(502, "Failed to exchange authorization code")
	}

	// Validate ID token claims (iss, aud) if an ID token was returned (OIDC §3)
	if tokenResp.IDToken != "" {
		if err := validateIDToken(tokenResp.IDToken, cfg, stored.Nonce); err != nil {
			if s.Logger != nil {
				s.Logger.Error("OAuth ID token validation failed", "provider", provider, "error", err)
			}
			return nil, domain.Err(400, "ID token validation failed")
		}
	}

	// Fetch user info from provider
	userInfo, err := fetchUserInfo(ctx, cfg, tokenResp.AccessToken)
	if err != nil {
		if s.Logger != nil {
			s.Logger.Error("OAuth userinfo fetch failed", "provider", provider, "error", err)
		}
		return nil, domain.Err(502, "Failed to fetch user info from provider")
	}

	if userInfo.Subject == "" {
		return nil, domain.Err(502, "Provider did not return subject identifier")
	}

	// Find or create user
	authResp, isNewUser, err := s.findOrCreateOAuthUser(ctx, provider, *userInfo, meta)
	if err != nil {
		return nil, err
	}

	if s.Logger != nil {
		s.Logger.Info("oauth callback success",
			"provider", provider,
			"user_id", authResp.User.ID,
			"is_new_user", isNewUser,
		)
	}

	return &OAuthCallbackResult{
		AuthResponse: authResp,
		IsNewUser:    isNewUser,
	}, nil
}

// findOrCreateOAuthUser handles the identity mapping:
//  1. If oauth_accounts row exists for (provider, subject) → link to existing user
//  2. If no oauth_accounts row but user with same email exists → link (merge)
//  3. If neither exists → create new user + oauth_accounts row
func (s *Services) findOrCreateOAuthUser(ctx context.Context, provider string, info OAuthUserInfo, meta *SessionMeta) (*domain.AuthResponse, bool, error) {
	oauthAccountID := provider + "|" + info.Subject

	// 1. Check if oauth_accounts row exists
	var existingUserID string
	err := s.DB.QueryRow(ctx,
		`SELECT user_id FROM oauth_accounts WHERE id = $1`,
		oauthAccountID,
	).Scan(&existingUserID)

	if err == nil {
		// Existing link — load user and issue tokens
		authResp, _, err := s.issueTokensForUser(ctx, existingUserID, meta)
		if err != nil {
			return nil, false, err
		}
		return authResp, false, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, false, domain.Err(500, "Database error")
	}

	// 2. Check if user with same email exists (merge)
	var mergeUserID string
	if info.Email != "" {
		err = s.DB.QueryRow(ctx,
			`SELECT id FROM users WHERE email = $1 AND deleted_at IS NULL`,
			info.Email,
		).Scan(&mergeUserID)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return nil, false, domain.Err(500, "Database error")
		}
	}

	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return nil, false, domain.Err(500, "Database error")
	}
	defer tx.Rollback(ctx)

	var userID string
	isNewUser := false

	if mergeUserID != "" {
		// Merge: link existing user to this provider
		userID = mergeUserID
	} else {
		// 3. Create new user
		userID = uuid.New().String()
		now := time.Now().UTC()
		screenName := info.DisplayName
		if screenName == "" {
			screenName = "user_" + userID[:8]
		}

		_, err = tx.Exec(ctx,
			`INSERT INTO users (id, email, screen_name, password_hash, role, accept_terms_version, created_at, updated_at)
			 VALUES ($1, $2, $3, '', 'user', '1.0', $4, $4)`,
			userID, info.Email, screenName, now,
		)
		if err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "23505" {
				return nil, false, domain.Err(409, "User with this email already exists")
			}
			return nil, false, domain.Err(500, "Failed to create user")
		}
		isNewUser = true
	}

	// Insert oauth_accounts link
	_, err = tx.Exec(ctx,
		`INSERT INTO oauth_accounts (id, user_id, provider, subject, email, display_name, avatar_url, linked_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())`,
		oauthAccountID, userID, provider, info.Subject, info.Email, info.DisplayName, info.AvatarURL,
	)
	if err != nil {
		return nil, false, domain.Err(500, "Failed to link OAuth account")
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, false, domain.Err(500, "Failed to commit transaction")
	}

	authResp, _, err := s.issueTokensForUser(ctx, userID, meta)
	if err != nil {
		return nil, false, err
	}
	return authResp, isNewUser, nil
}

// issueTokensForUser loads a user by ID and issues a fresh token pair.
func (s *Services) issueTokensForUser(ctx context.Context, userID string, meta *SessionMeta) (*domain.AuthResponse, bool, error) {
	var user struct {
		ID         string
		Email      string
		ScreenName string
		Role       string
		CreatedAt  time.Time
	}
	err := s.DB.QueryRow(ctx,
		`SELECT id, email, screen_name, role, created_at FROM users WHERE id = $1 AND deleted_at IS NULL`,
		userID,
	).Scan(&user.ID, &user.Email, &user.ScreenName, &user.Role, &user.CreatedAt)
	if err != nil {
		return nil, false, domain.Err(500, "User not found")
	}

	accessToken, refreshToken, tokenID, expiresAt, err := s.generateTokens(user.ID, user.Email, user.ScreenName, []string{user.Role})
	if err != nil {
		return nil, false, domain.Err(500, "Failed to generate tokens")
	}

	_, secret, ok := splitRefreshToken(refreshToken)
	if !ok {
		return nil, false, domain.Err(500, "Failed to process refresh token")
	}

	tokenHash := sha256Hash(secret)
	if err := s.storeRefreshToken(ctx, s.DB, user.ID, tokenID, tokenHash, meta); err != nil {
		return nil, false, domain.Err(500, "Failed to create session")
	}

	return &domain.AuthResponse{
		AccessToken:      accessToken,
		AccessExpiresIn:  int64(accessTokenExpiry.Seconds()),
		RefreshToken:     refreshToken,
		RefreshExpiresIn: int64(refreshTokenExpiry.Seconds()),
		TokenType:        "Bearer",
		ExpiresAt:        expiresAt,
		User: domain.UserPublic{
			ID:         user.ID,
			ScreenName: user.ScreenName,
			Email:      user.Email,
			Role:       user.Role,
			CreatedAt:  utils.FormatRFC3339UTC(user.CreatedAt),
		},
	}, false, nil
}

func generateCodeVerifier() (string, error) {
	b := make([]byte, 32) // 32 bytes → 43 base64url chars
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func computeCodeChallengeS256(verifier string) string {
	h := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(h[:])
}

func generateState() (string, error) {
	b := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// --- Token exchange ---

type tokenResponse struct {
	AccessToken string `json:"access_token"`
	IDToken     string `json:"id_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
	Scope       string `json:"scope"`
}

func exchangeCodeForTokens(ctx context.Context, cfg OAuthProviderConfig, code, codeVerifier, redirectURI string) (*tokenResponse, error) {
	params := url.Values{}
	params.Set("grant_type", "authorization_code")
	params.Set("code", code)
	params.Set("client_id", cfg.ClientID)
	params.Set("client_secret", cfg.ClientSecret)
	params.Set("redirect_uri", redirectURI)
	params.Set("code_verifier", codeVerifier)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.TokenURL, strings.NewReader(params.Encode()))
	if err != nil {
		return nil, fmt.Errorf("token request creation failed: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("token request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("token endpoint returned %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp tokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("failed to decode token response: %w", err)
	}

	return &tokenResp, nil
}

// --- UserInfo fetch ---

func fetchUserInfo(ctx context.Context, cfg OAuthProviderConfig, accessToken string) (*OAuthUserInfo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, cfg.UserInfoURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create userinfo request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("userinfo request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("userinfo endpoint returned %d", resp.StatusCode)
	}

	var raw map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("failed to decode userinfo response: %w", err)
	}

	info := &OAuthUserInfo{}
	if sub, ok := raw["sub"].(string); ok {
		info.Subject = sub
	}
	if email, ok := raw["email"].(string); ok {
		info.Email = email
	}
	if name, ok := raw["name"].(string); ok {
		info.DisplayName = name
	} else if given, ok := raw["given_name"].(string); ok {
		info.DisplayName = given
	}
	if picture, ok := raw["picture"].(string); ok {
		info.AvatarURL = picture
	}

	return info, nil
}

// --- Redirect URI allowlist ---

// validateRedirectURI checks that the redirect_uri is in the provider's allowlist.
// If the allowlist is empty (development), all URIs are accepted.
func validateRedirectURI(cfg OAuthProviderConfig, redirectURI string) error {
	if redirectURI == "" {
		return fmt.Errorf("redirect_uri is required")
	}
	if len(cfg.AllowedRedirectURIs) == 0 {
		return nil // no allowlist configured — accept all (dev mode)
	}
	for _, allowed := range cfg.AllowedRedirectURIs {
		if redirectURI == allowed {
			return nil
		}
	}
	return fmt.Errorf("redirect_uri not allowed: %s", redirectURI)
}

// --- OIDC ID token validation ---

// idTokenClaims represents the OIDC claims we validate from an ID token.
type idTokenClaims struct {
	Iss   string `json:"iss"`   // issuer — must match cfg.IssuerURL
	Aud   string `json:"aud"`   // audience — must match cfg.ClientID (single-audience check)
	Sub   string `json:"sub"`   // subject identifier
	Nonce string `json:"nonce"` // must match the nonce stored in oauth_states
	Exp   int64  `json:"exp"`   // expiry — must not be in the past
}

// validateIDToken performs lightweight validation of an OIDC ID token.
// It decodes the JWT payload (unverified signature — signature verification
// requires JWKS fetching which is deferred to a future iteration) and checks:
//   - iss matches the configured IssuerURL
//   - aud matches the configured ClientID
//   - nonce matches the nonce from the authorize request
//   - exp is not in the past
func validateIDToken(idToken string, cfg OAuthProviderConfig, expectedNonce string) error {
	// JWT format: header.payload.signature — we only need the payload
	parts := strings.SplitN(idToken, ".", 3)
	if len(parts) != 3 {
		return fmt.Errorf("malformed ID token: expected 3 parts, got %d", len(parts))
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return fmt.Errorf("failed to decode ID token payload: %w", err)
	}

	var claims idTokenClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return fmt.Errorf("failed to parse ID token claims: %w", err)
	}

	// Validate iss (OIDC §3.1.3.7 — step 2)
	if cfg.IssuerURL != "" && claims.Iss != cfg.IssuerURL {
		return fmt.Errorf("ID token iss %q does not match expected %q", claims.Iss, cfg.IssuerURL)
	}

	// Validate aud (OIDC §3.1.3.7 — step 3)
	if claims.Aud != cfg.ClientID {
		return fmt.Errorf("ID token aud %q does not match expected %q", claims.Aud, cfg.ClientID)
	}

	// Validate nonce (OIDC §3.1.3.7 — step 11)
	if expectedNonce != "" && claims.Nonce != expectedNonce {
		return fmt.Errorf("ID token nonce does not match")
	}

	// Validate exp (OIDC §3.1.3.7 — step 9)
	if claims.Exp > 0 && time.Now().Unix() > claims.Exp {
		return fmt.Errorf("ID token expired")
	}

	return nil
}

// --- Expired state cleanup ---

// CleanupExpiredOAuthStates removes expired rows from oauth_states.
// Call this periodically (e.g. via cron or on server idle) to prevent table bloat.
func (s *Services) CleanupExpiredOAuthStates(ctx context.Context) (int64, error) {
	tag, err := s.DB.Exec(ctx, `DELETE FROM oauth_states WHERE expires_at < NOW()`)
	if err != nil {
		if s.Logger != nil {
			s.Logger.Error("failed to cleanup expired oauth_states", "error", err)
		}
		return 0, domain.Err(500, "Failed to cleanup expired OAuth states")
	}
	deleted := tag.RowsAffected()
	if s.Logger != nil {
		s.Logger.Info("cleaned up expired oauth_states", "deleted", deleted)
	}
	return deleted, nil
}
