package services

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/harem-brasil/backend/internal/domain"
	"github.com/harem-brasil/backend/internal/middleware"
)

type CreateUploadSessionBody struct {
	FileName    string `json:"file_name"`
	ContentType string `json:"content_type"`
	Size        int64  `json:"size"`
}

func (s *Services) CreateUploadSession(ctx context.Context, user *middleware.UserClaims, req CreateUploadSessionBody) (*domain.UploadSession, error) {
	if req.FileName == "" {
		return nil, domain.ErrValidation("file_name required", map[string]string{"file_name": "Required"})
	}

	if req.Size <= 0 {
		return nil, domain.ErrValidation("size invalid", map[string]string{"size": "Must be greater than 0"})
	}

	maxFileSize := s.MaxFileSize
	if maxFileSize == 0 {
		maxFileSize = 100 * 1024 * 1024
	}
	if req.Size > maxFileSize {
		return nil, domain.ErrValidation("file too large", map[string]string{
			"size": fmt.Sprintf("File too large (max %d bytes)", maxFileSize),
		})
	}

	sessionID := uuid.New().String()

	return &domain.UploadSession{
		ID:            sessionID,
		Status:        "pending",
		UploadURL:     "/upload/not-implemented",
		ContentType:   req.ContentType,
		ContentLength: req.Size,
		ExpiresAt:     "",
	}, nil
}

type CompleteUploadBody struct {
	ETag string `json:"etag"`
}

func (s *Services) CompleteUpload(ctx context.Context, user *middleware.UserClaims, uploadID string, req CompleteUploadBody) (*domain.UploadSession, error) {
	return &domain.UploadSession{
		ID:     uploadID,
		Status: "completed",
	}, nil
}
