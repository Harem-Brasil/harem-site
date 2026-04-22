package application

import "log/slog"

// Config agrupa dependências da camada HTTP (servidor Gin).
type Config struct {
	Port        string
	DBURL       string
	RedisURL    string
	JWTSecret   string
	Logger      *slog.Logger
	MaxFileSize int64
}
