package application

import "log/slog"

// DefaultCORSAllowedOrigins são origens usadas quando CORS_ALLOWED_ORIGINS não está definido
// (desenvolvimento local com frontends típicos). Em produção, defina CORS_ALLOWED_ORIGINS.
var DefaultCORSAllowedOrigins = []string{
	"http://localhost:3000",
	"http://localhost:5173",
	"http://127.0.0.1:3000",
	"http://127.0.0.1:5173",
}

// Config agrupa dependências da camada HTTP (servidor Gin).
type Config struct {
	Port               string
	DBURL              string
	RedisURL           string
	JWTSecret          string
	Logger             *slog.Logger
	MaxFileSize        int64
	CORSAllowedOrigins []string

	StripeWebhookSecret      string
	PagSeguroWebhookSecret   string
	MercadoPagoWebhookSecret string
	InternalBillingSecret    string
	AppEnv                   string
}
