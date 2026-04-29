package services

import (
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

// Dependencies agrega infraestrutura injetável nas regras de negócio.
type Dependencies struct {
	DB          *pgxpool.Pool
	Redis       *redis.Client
	JWTSecret   []byte
	Logger      *slog.Logger
	MaxFileSize int64

	StripeWebhookSecret      string
	PagSeguroWebhookSecret   string
	MercadoPagoWebhookSecret string
	// AppEnv replica ENV (ex.: development, test, production); usado quando o segredo do webhook está vazio.
	AppEnv string

	// OAuth provider configs (keyed by provider name, e.g. "google").
	OAuthProviders map[string]OAuthProviderConfig
}

// OAuthProviderConfig holds client credentials and endpoints for one OIDC provider.
// Secrets come from environment variables — never from the repository.
type OAuthProviderConfig struct {
	ClientID     string
	ClientSecret string
	AuthorizeURL string // e.g. https://accounts.google.com/o/oauth2/v2/auth
	TokenURL     string // e.g. https://oauth2.googleapis.com/token
	UserInfoURL  string // e.g. https://openidconnect.googleapis.com/v1/userinfo
	IssuerURL    string // e.g. https://accounts.google.com — validated against ID token iss claim
	Scopes       []string

	// AllowedRedirectURIs is an allowlist of permitted redirect_uri values.
	// If empty, all URIs are accepted (useful for development only).
	AllowedRedirectURIs []string
}
