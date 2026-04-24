package application

import (
	"context"
	"net/http"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"github.com/harem-brasil/backend/internal/controllers"
	"github.com/harem-brasil/backend/internal/datasources"
	httpmw "github.com/harem-brasil/backend/internal/middleware"
	"github.com/harem-brasil/backend/internal/services"
)

// HTTPServer encapsula Engine Gin e ciclo de vida dos serviços.
type HTTPServer struct {
	Engine   *gin.Engine
	Services *services.Services
}

// NewHTTPServer monta pool Postgres, Redis, serviços e router Gin (composition root).
// ctx governa timeouts e cancelamento durante a inicialização (Postgres e Redis).
func NewHTTPServer(ctx context.Context, cfg Config) (*HTTPServer, error) {
	db, err := datasources.NewPostgresPool(ctx, cfg.DBURL)
	if err != nil {
		return nil, err
	}

	opt, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		db.Close()
		return nil, err
	}
	rdb := redis.NewClient(opt)

	if err := rdb.Ping(ctx).Err(); err != nil {
		_ = rdb.Close()
		db.Close()
		return nil, err
	}

	svc := services.New(services.Dependencies{
		DB:                       db,
		Redis:                    rdb,
		JWTSecret:                []byte(cfg.JWTSecret),
		Logger:                   cfg.Logger,
		MaxFileSize:              cfg.MaxFileSize,
		StripeWebhookSecret:      cfg.StripeWebhookSecret,
		PagSeguroWebhookSecret:   cfg.PagSeguroWebhookSecret,
		MercadoPagoWebhookSecret: cfg.MercadoPagoWebhookSecret,
		AppEnv:                   cfg.AppEnv,
	})

	corsOrigins := cfg.CORSAllowedOrigins
	if len(corsOrigins) == 0 {
		corsOrigins = DefaultCORSAllowedOrigins
	}

	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()
	engine.Use(gin.Recovery())
	engine.Use(httpmw.RequestID())
	engine.Use(httpmw.RequestLogger(cfg.Logger))
	engine.Use(cors.New(cors.Config{
		AllowOrigins:     corsOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Accept", "Authorization", "Content-Type", "X-Request-Id", "Idempotency-Key", "If-Match"},
		ExposeHeaders:    []string{"RateLimit-Limit", "RateLimit-Remaining", "RateLimit-Reset", "Retry-After"},
		AllowCredentials: true,
		MaxAge:           300,
	}))
	engine.Use(httpmw.GinRateLimit(rdb, cfg.Logger))

	engine.Use(func(c *gin.Context) {
		c.Header("Content-Type", "application/json; charset=utf-8")
		c.Next()
	})

	jwtSecret := []byte(cfg.JWTSecret)
	controllers.RegisterRoutes(engine, svc, jwtSecret, cfg.Logger, rdb)

	return &HTTPServer{Engine: engine, Services: svc}, nil
}

// ServeHTTP implementa http.Handler.
func (s *HTTPServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.Engine.ServeHTTP(w, r)
}

// Close libera pool e Redis.
func (s *HTTPServer) Close() error {
	return s.Services.Close()
}
