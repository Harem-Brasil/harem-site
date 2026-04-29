package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/harem-brasil/backend/internal/application"
	"github.com/harem-brasil/backend/internal/migrate"
	"github.com/harem-brasil/backend/internal/seed"
	"github.com/harem-brasil/backend/internal/services"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		// .env file not found or error loading - will use env vars or defaults
	}

	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: harem-api <command> [options]")
		fmt.Fprintln(os.Stderr, "Commands:")
		fmt.Fprintln(os.Stderr, "  serve    Start the API server")
		fmt.Fprintln(os.Stderr, "  migrate  Run database migrations")
		fmt.Fprintln(os.Stderr, "  seed     Seed database with test data")
		os.Exit(1)
	}

	command := os.Args[1]
	os.Args = append([]string{os.Args[0]}, os.Args[2:]...)

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://harem:harem@localhost:5432/harem?sslmode=disable"
	}

	switch command {
	case "serve":
		runServe(logger, dbURL)
	case "migrate":
		runMigrate(dbURL)
	case "seed":
		runSeed(dbURL)
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
		fmt.Fprintln(os.Stderr, "Commands: serve, migrate, seed")
		os.Exit(1)
	}
}

func runServe(logger *slog.Logger, dbURL string) {
	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	port := fs.String("port", getEnv("PORT", "40080"), "Server port")
	redisURL := fs.String("redis", getEnv("REDIS_URL", "redis://localhost:6379/0"), "Redis URL")
	jwtSecret := fs.String("jwt-secret", getEnv("JWT_SECRET", ""), "JWT secret (min 32 chars)")
	fs.Parse(os.Args[1:])

	if *jwtSecret == "" {
		*jwtSecret = "development-secret-change-in-production"
		slog.Warn("using default JWT secret - set JWT_SECRET env var in production")
	}

	corsOrigins := parseCommaSeparatedOrigins(os.Getenv("CORS_ALLOWED_ORIGINS"))

	oauthProviders := buildOAuthProviders()

	initCtx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	server, err := application.NewHTTPServer(initCtx, application.Config{
		Port:                     *port,
		DBURL:                    dbURL,
		RedisURL:                 *redisURL,
		JWTSecret:                *jwtSecret,
		Logger:                   logger,
		CORSAllowedOrigins:       corsOrigins,
		StripeWebhookSecret:      getEnv("STRIPE_WEBHOOK_SECRET", ""),
		PagSeguroWebhookSecret:   getEnv("PAGSEGURO_WEBHOOK_SECRET", ""),
		MercadoPagoWebhookSecret: getEnv("MERCADOPAGO_WEBHOOK_SECRET", ""),
		AppEnv:                   getEnv("ENV", ""),
		CommitHash:               getEnv("COMMIT_HASH", ""),
		OAuthProviders:           oauthProviders,
	})
	if err != nil {
		slog.Error("failed to create server", "error", err)
		os.Exit(1)
	}

	srv := &http.Server{
		Addr:         "0.0.0.0:" + *port,
		Handler:      server,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		slog.Info("starting server", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	slog.Info("shutting down server")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("server shutdown error", "error", err)
	}

	if err := server.Close(); err != nil {
		slog.Error("server close error", "error", err)
	}

	slog.Info("server stopped")
}

func runMigrate(dbURL string) {
	fs := flag.NewFlagSet("migrate", flag.ExitOnError)
	migrationsDir := fs.String("dir", "migrations", "Migrations directory")
	fs.Parse(os.Args[1:])

	migrator, err := migrate.New(dbURL, *migrationsDir)
	if err != nil {
		slog.Error("failed to create migrator", "error", err)
		os.Exit(1)
	}
	defer migrator.Close()

	if err := migrator.Up(); err != nil {
		slog.Error("migration failed", "error", err)
		os.Exit(1)
	}

	slog.Info("migrations completed successfully")
}

func runSeed(dbURL string) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		slog.Error("failed to ping database", "error", err)
		os.Exit(1)
	}

	seeder := seed.New(pool)
	if err := seeder.Run(ctx); err != nil {
		slog.Error("seeding failed", "error", err)
		os.Exit(1)
	}
}

func getEnv(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}

// buildOAuthProviders reads OAUTH_GOOGLE_* env vars and returns a provider map.
// Add more providers by reading additional OAUTH_<PROVIDER>_* env vars.
func buildOAuthProviders() map[string]services.OAuthProviderConfig {
	providers := map[string]services.OAuthProviderConfig{}

	if id := os.Getenv("OAUTH_GOOGLE_CLIENT_ID"); id != "" {
		providers["google"] = services.OAuthProviderConfig{
			ClientID:            id,
			ClientSecret:        os.Getenv("OAUTH_GOOGLE_CLIENT_SECRET"),
			AuthorizeURL:        getEnv("OAUTH_GOOGLE_AUTHORIZE_URL", "https://accounts.google.com/o/oauth2/v2/auth"),
			TokenURL:            getEnv("OAUTH_GOOGLE_TOKEN_URL", "https://oauth2.googleapis.com/token"),
			UserInfoURL:         getEnv("OAUTH_GOOGLE_USERINFO_URL", "https://openidconnect.googleapis.com/v1/userinfo"),
			IssuerURL:           getEnv("OAUTH_GOOGLE_ISSUER_URL", "https://accounts.google.com"),
			Scopes:              []string{"openid", "email", "profile"},
			AllowedRedirectURIs: parseCommaSeparatedOrigins(os.Getenv("OAUTH_GOOGLE_ALLOWED_REDIRECT_URIS")),
		}
	}

	return providers
}

// parseCommaSeparatedOrigins divide CORS_ALLOWED_ORIGINS (vírgulas).
// Valores vazios são ignorados. Lista vazia faz o servidor usar DefaultCORSAllowedOrigins.
func parseCommaSeparatedOrigins(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
