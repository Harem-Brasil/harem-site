package httpapi

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	httpmw "github.com/harem-brasil/backend/internal/middleware"
)

type UploadSession struct {
	ID            string `json:"id"`
	Status        string `json:"status"`
	UploadURL     string `json:"upload_url"`
	ContentType   string `json:"content_type,omitempty"`
	ContentLength int64  `json:"content_length,omitempty"`
	ExpiresAt     string `json:"expires_at,omitempty"`
}

func (s *Server) handleCreateUploadSession(w http.ResponseWriter, r *http.Request) {
	user := httpmw.GetUser(r.Context())
	if user == nil {
		respondError(w, http.StatusUnauthorized, "Not authenticated")
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Failed to read request body")
		return
	}

	var req struct {
		FileName    string `json:"file_name"`
		ContentType string `json:"content_type"`
		Size        int64  `json:"size"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	if req.FileName == "" {
		respondValidationError(w, map[string]string{"file_name": "Required"})
		return
	}

	if req.Size <= 0 {
		respondValidationError(w, map[string]string{"size": "Must be greater than 0"})
		return
	}

	// Max file size: 100MB
	if req.Size > 100*1024*1024 {
		respondValidationError(w, map[string]string{"size": "File too large (max 100MB)"})
		return
	}

	sessionID := uuid.New().String()

	// TODO: Generate presigned URL for direct upload to S3/MinIO
	// For now, return a stub

	respondCreated(w, UploadSession{
		ID:            sessionID,
		Status:        "pending",
		UploadURL:     "/upload/not-implemented",
		ContentType:   req.ContentType,
		ContentLength: req.Size,
		ExpiresAt:     "",
	})
}

func (s *Server) handleCompleteUpload(w http.ResponseWriter, r *http.Request) {
	user := httpmw.GetUser(r.Context())
	if user == nil {
		respondError(w, http.StatusUnauthorized, "Not authenticated")
		return
	}

	uploadID := chi.URLParam(r, "id")

	body, err := io.ReadAll(r.Body)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Failed to read request body")
		return
	}

	var req struct {
		ETag string `json:"etag"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	// TODO: Verify upload completion with storage backend
	// TODO: Update session status and create media record

	respondJSON(w, UploadSession{
		ID:     uploadID,
		Status: "completed",
	})
}
