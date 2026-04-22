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
}
