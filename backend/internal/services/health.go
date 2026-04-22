package services

import (
	"context"
	"time"
)

func (s *Services) Health(ctx context.Context) map[string]any {
	status := map[string]any{
		"status":    "ok",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}

	dbErr := s.DB.Ping(ctx)
	status["database"] = map[string]string{"status": "ok"}
	if dbErr != nil {
		status["database"] = map[string]string{
			"status": "error",
			"error":  dbErr.Error(),
		}
		status["status"] = "degraded"
	}

	redisErr := s.Redis.Ping(ctx).Err()
	status["redis"] = map[string]string{"status": "ok"}
	if redisErr != nil {
		status["redis"] = map[string]string{
			"status": "error",
			"error":  redisErr.Error(),
		}
		status["status"] = "degraded"
	}

	return status
}

func (s *Services) Healthz(ctx context.Context) (healthy bool, checks map[string]string, version string) {
	checks = map[string]string{}
	healthy = true

	if err := s.DB.Ping(ctx); err != nil {
		healthy = false
		checks["database"] = "unhealthy"
	} else {
		checks["database"] = "healthy"
	}

	if err := s.Redis.Ping(ctx).Err(); err != nil {
		healthy = false
		checks["redis"] = "unhealthy"
	} else {
		checks["redis"] = "healthy"
	}

	return healthy, checks, "1.0.0"
}

func (s *Services) Readyz(ctx context.Context) (ready bool, checks map[string]string) {
	checks = map[string]string{}
	ready = true

	if err := s.DB.Ping(ctx); err != nil {
		ready = false
		checks["database"] = "not_ready"
	} else {
		checks["database"] = "ready"
	}

	if err := s.Redis.Ping(ctx).Err(); err != nil {
		ready = false
		checks["redis"] = "not_ready"
	} else {
		checks["redis"] = "ready"
	}

	return ready, checks
}

func (s *Services) Version() map[string]any {
	return map[string]any{
		"version":   "1.0.0",
		"build":     "dev",
		"api":       "v1",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}
}
