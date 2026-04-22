package utils

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/harem-brasil/backend/internal/domain"
)

// RespondProblem envia application/problem+json (RFC 7807).
func RespondProblem(c *gin.Context, status int, title, detail string) {
	c.Header("Content-Type", "application/problem+json; charset=utf-8")
	c.JSON(status, domain.ProblemDetail{
		Type:   "about:blank",
		Title:  title,
		Status: status,
		Detail: detail,
	})
}

// RespondJSON define Content-Type JSON explícito (rotas API).
func RespondJSON(c *gin.Context, status int, data any) {
	c.Header("Content-Type", "application/json; charset=utf-8")
	c.JSON(status, data)
}

// RespondValidation envia 422 com campos inválidos.
func RespondValidation(c *gin.Context, detail string, fields map[string]string) {
	c.Header("Content-Type", "application/problem+json; charset=utf-8")
	c.JSON(http.StatusUnprocessableEntity, domain.ProblemDetail{
		Type:       "validation-error",
		Title:      "Validation Error",
		Status:     http.StatusUnprocessableEntity,
		Detail:     detail,
		Extensions: map[string]any{"fields": fields},
	})
}

// HandleServiceError mapeia domain.AppError e erros genéricos para resposta HTTP.
func HandleServiceError(c *gin.Context, logger *slog.Logger, err error) {
	var appErr *domain.AppError
	if errors.As(err, &appErr) {
		if len(appErr.FieldErrors) > 0 {
			RespondValidation(c, appErr.Detail, appErr.FieldErrors)
			return
		}
		title := appErr.Title
		if title == "" {
			title = http.StatusText(appErr.HTTPStatus)
		}
		RespondProblem(c, appErr.HTTPStatus, title, appErr.Detail)
		return
	}
	if logger != nil {
		logger.Error("internal server error", "path", c.FullPath(), "error", err)
	}
	RespondProblem(c, http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError),
		"An unexpected error occurred")
}
