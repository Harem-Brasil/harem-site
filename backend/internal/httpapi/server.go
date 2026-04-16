package httpapi

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	httpmw "github.com/harem-brasil/backend/internal/middleware"
)

type Config struct {
	Port        string
	DBURL       string
	RedisURL    string
	JWTSecret   string
	Logger      *slog.Logger
	MaxFileSize int64
}

type Server struct {
	config    Config
	router    chi.Router
	db        *pgxpool.Pool
	redis     *redis.Client
	jwtSecret []byte
}

func New(cfg Config) (*Server, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db, err := pgxpool.New(ctx, cfg.DBURL)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(ctx); err != nil {
		return nil, err
	}

	opt, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		return nil, err
	}
	rdb := redis.NewClient(opt)

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	s := &Server{
		config:    cfg,
		db:        db,
		redis:     rdb,
		jwtSecret: []byte(cfg.JWTSecret),
	}

	s.setupRouter()
	return s, nil
}

func (s *Server) setupRouter() {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(httpmw.RequestLogger(s.config.Logger))
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Request-Id", "Idempotency-Key", "If-Match"},
		ExposedHeaders:   []string{"RateLimit-Limit", "RateLimit-Remaining", "RateLimit-Reset", "Retry-After"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	r.Use(httpmw.RateLimit(s.redis, s.config.Logger))

	r.Use(middleware.SetHeader("Content-Type", "application/json; charset=utf-8"))

	r.Get("/health", s.handleHealth)
	r.Get("/healthz", s.handleHealthz)
	r.Get("/readyz", s.handleReadyz)
	r.Get("/version", s.handleVersion)

	r.Route("/api/v1", func(r chi.Router) {
		r.Use(middleware.StripSlashes)

		r.Group(func(r chi.Router) {
			r.Use(httpmw.MaxBodySize(1 << 20))
			r.Post("/auth/register", s.handleRegister)
			r.Post("/auth/login", s.handleLogin)
			r.Post("/auth/refresh", s.handleRefresh)
			r.Post("/auth/logout", s.handleLogout)
			r.Post("/auth/logout-all", s.handleLogoutAll)
			r.Get("/auth/oauth/{provider}/authorize", s.handleOAuthAuthorize)
			r.Get("/auth/oauth/{provider}/callback", s.handleOAuthCallback)
			r.Post("/auth/email/verify", s.handleEmailVerify)
			r.Post("/auth/password/forgot", s.handlePasswordForgot)
			r.Post("/auth/password/reset", s.handlePasswordReset)
		})

		r.Group(func(r chi.Router) {
			r.Use(httpmw.MaxBodySize(1 << 20))
			r.Use(httpmw.Auth(s.jwtSecret, []string{"user", "creator", "moderator", "admin"}))
			r.Get("/me", s.handleGetMe)
			r.Patch("/me", s.handleUpdateMe)
			r.Delete("/me", s.handleDeleteMe)
		})

		r.Group(func(r chi.Router) {
			r.Use(httpmw.MaxBodySize(10 << 20))
			r.Use(httpmw.Auth(s.jwtSecret, []string{"user", "creator", "moderator", "admin"}))
			r.Get("/users/{id}", s.handleGetUser)
			r.Get("/users/{id}/posts", s.handleGetUserPosts)
			r.Get("/users", s.handleListUsers)
			r.Get("/users/search", s.handleSearchUsers)
		})

		r.Group(func(r chi.Router) {
			r.Use(httpmw.MaxBodySize(10 << 20))
			r.Use(httpmw.Auth(s.jwtSecret, []string{"user", "creator", "moderator", "admin"}))
			r.Get("/posts", s.handleListPosts)
			r.Get("/posts/{id}", s.handleGetPost)
			r.Post("/posts", s.handleCreatePost)
			r.Patch("/posts/{id}", s.handleUpdatePost)
			r.Delete("/posts/{id}", s.handleDeletePost)
			r.Post("/posts/{id}/like", s.handleLikePost)
			r.Delete("/posts/{id}/like", s.handleUnlikePost)
			r.Get("/posts/{id}/comments", s.handleListComments)
			r.Post("/posts/{id}/comments", s.handleCreateComment)
			r.Get("/feed/home", s.handleFeedHome)
		})

		r.Group(func(r chi.Router) {
			r.Use(httpmw.MaxBodySize(1 << 20))
			r.Use(httpmw.Auth(s.jwtSecret, []string{"user", "creator", "moderator", "admin"}))
			r.Get("/forum/categories", s.handleListForumCategories)
			r.Get("/forum/topics", s.handleListForumTopics)
			r.Post("/forum/topics", s.handleCreateForumTopic)
			r.Get("/forum/topics/{id}", s.handleGetForumTopic)
			r.Post("/forum/topics/{id}/posts", s.handleCreateForumPost)
		})

		r.Group(func(r chi.Router) {
			r.Use(httpmw.MaxBodySize(1 << 20))
			r.Use(httpmw.Auth(s.jwtSecret, []string{"user", "creator", "moderator", "admin"}))
			r.Get("/chat/rooms", s.handleListChatRooms)
			r.Post("/chat/rooms", s.handleCreateChatRoom)
			r.Get("/chat/rooms/{id}", s.handleGetChatRoom)
			r.Post("/chat/rooms/{id}/join", s.handleJoinChatRoom)
			r.Get("/chat/rooms/{id}/messages", s.handleListMessages)
		})

		r.Group(func(r chi.Router) {
			r.Use(httpmw.MaxBodySize(1 << 20))
			r.Use(httpmw.Auth(s.jwtSecret, []string{"user", "creator", "moderator", "admin"}))
			r.Get("/notifications", s.handleListNotifications)
			r.Patch("/notifications/{id}/read", s.handleMarkNotificationRead)
			r.Get("/notifications/unread-count", s.handleUnreadCount)
		})

		r.Group(func(r chi.Router) {
			r.Use(httpmw.MaxBodySize(1 << 20))
			r.Use(httpmw.Auth(s.jwtSecret, []string{"creator", "admin"}))
			r.Post("/creator/apply", s.handleCreatorApply)
			r.Get("/creator/dashboard", s.handleCreatorDashboard)
			r.Get("/creator/earnings", s.handleCreatorEarnings)
			r.Get("/creator/catalog", s.handleCreatorCatalog)
			r.Get("/creator/orders", s.handleCreatorOrders)
		})

		r.Group(func(r chi.Router) {
			r.Use(httpmw.MaxBodySize(1 << 20))
			r.Use(httpmw.Auth(s.jwtSecret, []string{"user", "creator", "moderator", "admin"}))
			r.Get("/billing/plans", s.handleListPlans)
			r.Post("/billing/checkout", s.handleBillingCheckout)
			r.Get("/billing/subscription", s.handleGetMySubscription)
			r.Post("/billing/subscription/cancel", s.handleCancelSubscription)
			r.Post("/billing/subscription/resume", s.handleResumeSubscription)
			r.Post("/subscriptions", s.handleCreateSubscription)
			r.Get("/subscriptions/me", s.handleGetMySubscription)
		})

		r.Group(func(r chi.Router) {
			r.Use(httpmw.MaxBodySize(10 << 20))
			r.Use(httpmw.Auth(s.jwtSecret, []string{"user", "creator", "moderator", "admin"}))
			r.Post("/media/upload-sessions", s.handleCreateUploadSession)
			r.Post("/media/upload-sessions/{id}/complete", s.handleCompleteUpload)
		})

		r.Group(func(r chi.Router) {
			r.Use(httpmw.MaxBodySize(1 << 20))
			r.Use(httpmw.Auth(s.jwtSecret, []string{"admin"}))
			r.Get("/admin/users", s.handleAdminListUsers)
			r.Patch("/admin/users/{id}/role", s.handleAdminUpdateRole)
			r.Delete("/admin/users/{id}", s.handleAdminDeleteUser)
			r.Get("/admin/stats", s.handleAdminStats)
			r.Get("/admin/audit-log", s.handleAdminAuditLog)
		})

		r.Group(func(r chi.Router) {
			r.Use(httpmw.MaxBodySize(1 << 20))
			r.Post("/webhooks/stripe", s.handleWebhookStripe)
			r.Post("/webhooks/pagseguro", s.handleWebhookPagSeguro)
			r.Post("/webhooks/mercadopago", s.handleWebhookMercadoPago)
			r.Post("/webhooks/{provider}", s.handleWebhookGeneric)
		})
	})

	s.router = r
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}

func (s *Server) Close() error {
	if s.db != nil {
		s.db.Close()
	}
	if s.redis != nil {
		return s.redis.Close()
	}
	return nil
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	status := map[string]any{
		"status":    "ok",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}

	dbErr := s.db.Ping(ctx)
	status["database"] = map[string]string{
		"status": "ok",
	}
	if dbErr != nil {
		status["database"] = map[string]string{
			"status": "error",
			"error":  dbErr.Error(),
		}
		status["status"] = "degraded"
	}

	redisErr := s.redis.Ping(ctx).Err()
	status["redis"] = map[string]string{
		"status": "ok",
	}
	if redisErr != nil {
		status["redis"] = map[string]string{
			"status": "error",
			"error":  redisErr.Error(),
		}
		status["status"] = "degraded"
	}

	if status["status"] == "degraded" {
		w.WriteHeader(http.StatusServiceUnavailable)
	}
	respondJSON(w, status)
}

func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	healthy := true
	checks := map[string]string{}

	if err := s.db.Ping(ctx); err != nil {
		healthy = false
		checks["database"] = "unhealthy"
	} else {
		checks["database"] = "healthy"
	}

	if err := s.redis.Ping(ctx).Err(); err != nil {
		healthy = false
		checks["redis"] = "unhealthy"
	} else {
		checks["redis"] = "healthy"
	}

	if !healthy {
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	respondJSON(w, map[string]any{
		"status":  map[bool]string{true: "healthy", false: "unhealthy"}[healthy],
		"checks":  checks,
		"version": "1.0.0",
	})
}

func (s *Server) handleReadyz(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	ready := true
	checks := map[string]string{}

	if err := s.db.Ping(ctx); err != nil {
		ready = false
		checks["database"] = "not_ready"
	} else {
		checks["database"] = "ready"
	}

	if err := s.redis.Ping(ctx).Err(); err != nil {
		ready = false
		checks["redis"] = "not_ready"
	} else {
		checks["redis"] = "ready"
	}

	if !ready {
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	respondJSON(w, map[string]any{
		"status": map[bool]string{true: "ready", false: "not_ready"}[ready],
		"checks": checks,
	})
}

func (s *Server) handleVersion(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, map[string]any{
		"version":   "1.0.0",
		"build":     "dev",
		"api":       "v1",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}
