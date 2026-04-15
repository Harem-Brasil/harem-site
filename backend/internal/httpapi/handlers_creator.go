package httpapi

import (
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"

	httpmw "github.com/harem-brasil/backend/internal/middleware"
)

type CreatorApplication struct {
	ID          string   `json:"id"`
	UserID      string   `json:"user_id"`
	Status      string   `json:"status"`
	Bio         string   `json:"bio"`
	SocialLinks []string `json:"social_links,omitempty"`
	SubmittedAt string   `json:"submitted_at"`
	ReviewedAt  *string  `json:"reviewed_at,omitempty"`
}

type CreatorDashboard struct {
	TotalPosts      int     `json:"total_posts"`
	TotalLikes      int     `json:"total_likes"`
	TotalFollowers  int     `json:"total_followers"`
	SubscriberCount int     `json:"subscriber_count"`
	MonthlyEarnings float64 `json:"monthly_earnings"`
}

func (s *Server) handleCreatorApply(w http.ResponseWriter, r *http.Request) {
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
		Bio         string   `json:"bio"`
		SocialLinks []string `json:"social_links"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	appID := uuid.New().String()
	now := time.Now().UTC()

	_, err = s.db.Exec(r.Context(),
		`INSERT INTO creator_applications (id, user_id, bio, social_links, status, submitted_at) 
		 VALUES ($1, $2, $3, $4, 'pending', $5)`,
		appID, user.UserID, req.Bio, req.SocialLinks, now,
	)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to submit application")
		return
	}

	respondCreated(w, CreatorApplication{
		ID:          appID,
		UserID:      user.UserID,
		Status:      "pending",
		Bio:         req.Bio,
		SocialLinks: req.SocialLinks,
		SubmittedAt: formatTimestamp(now),
	})
}

func (s *Server) handleCreatorDashboard(w http.ResponseWriter, r *http.Request) {
	user := httpmw.GetUser(r.Context())
	if user == nil {
		respondError(w, http.StatusUnauthorized, "Not authenticated")
		return
	}

	var dashboard CreatorDashboard

	_ = s.db.QueryRow(r.Context(),
		`SELECT COUNT(*) FROM posts WHERE author_id = $1 AND deleted_at IS NULL`,
		user.UserID,
	).Scan(&dashboard.TotalPosts)

	_ = s.db.QueryRow(r.Context(),
		`SELECT COALESCE(SUM(like_count), 0) FROM posts WHERE author_id = $1 AND deleted_at IS NULL`,
		user.UserID,
	).Scan(&dashboard.TotalLikes)

	respondJSON(w, dashboard)
}

func (s *Server) handleCreatorEarnings(w http.ResponseWriter, r *http.Request) {
	user := httpmw.GetUser(r.Context())
	if user == nil {
		respondError(w, http.StatusUnauthorized, "Not authenticated")
		return
	}

	respondJSON(w, map[string]any{
		"earnings": []any{},
		"total":    0.0,
	})
}
