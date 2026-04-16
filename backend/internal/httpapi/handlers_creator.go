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

func (s *Server) handleCreatorCatalog(w http.ResponseWriter, r *http.Request) {
	user := httpmw.GetUser(r.Context())
	if user == nil {
		respondError(w, http.StatusUnauthorized, "Not authenticated")
		return
	}

	cursor := r.URL.Query().Get("cursor")
	limit := 20

	rows, err := s.db.Query(r.Context(),
		`SELECT id, title, description, price_cents, currency, visibility, created_at 
		 FROM creator_catalog 
		 WHERE creator_id = $1 AND deleted_at IS NULL
		 AND ($2 = '' OR created_at < $2)
		 ORDER BY created_at DESC LIMIT $3`,
		user.UserID, cursor, limit+1,
	)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Database error")
		return
	}
	defer rows.Close()

	var items []any
	for rows.Next() {
		var item struct {
			ID          string `json:"id"`
			Title       string `json:"title"`
			Description string `json:"description"`
			PriceCents  int    `json:"price_cents"`
			Currency    string `json:"currency"`
			Visibility  string `json:"visibility"`
			CreatedAt   string `json:"created_at"`
		}
		err := rows.Scan(&item.ID, &item.Title, &item.Description, &item.PriceCents, &item.Currency, &item.Visibility, &item.CreatedAt)
		if err != nil {
			continue
		}
		items = append(items, item)
	}

	hasMore := len(items) > limit
	if hasMore {
		items = items[:limit]
	}

	nextCursor := ""
	if hasMore && len(items) > 0 {
		nextCursor = items[len(items)-1].(struct {
			ID          string `json:"id"`
			Title       string `json:"title"`
			Description string `json:"description"`
			PriceCents  int    `json:"price_cents"`
			Currency    string `json:"currency"`
			Visibility  string `json:"visibility"`
			CreatedAt   string `json:"created_at"`
		}).CreatedAt
	}

	respondJSON(w, CursorPage{
		Data:       items,
		NextCursor: nextCursor,
		HasMore:    hasMore,
	})
}

func (s *Server) handleCreatorOrders(w http.ResponseWriter, r *http.Request) {
	user := httpmw.GetUser(r.Context())
	if user == nil {
		respondError(w, http.StatusUnauthorized, "Not authenticated")
		return
	}

	cursor := r.URL.Query().Get("cursor")
	limit := 20

	rows, err := s.db.Query(r.Context(),
		`SELECT id, buyer_id, item_id, status, amount_cents, currency, created_at 
		 FROM creator_orders 
		 WHERE creator_id = $1
		 AND ($2 = '' OR created_at < $2)
		 ORDER BY created_at DESC LIMIT $3`,
		user.UserID, cursor, limit+1,
	)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Database error")
		return
	}
	defer rows.Close()

	var orders []any
	for rows.Next() {
		var order struct {
			ID          string `json:"id"`
			BuyerID     string `json:"buyer_id"`
			ItemID      string `json:"item_id"`
			Status      string `json:"status"`
			AmountCents int    `json:"amount_cents"`
			Currency    string `json:"currency"`
			CreatedAt   string `json:"created_at"`
		}
		err := rows.Scan(&order.ID, &order.BuyerID, &order.ItemID, &order.Status, &order.AmountCents, &order.Currency, &order.CreatedAt)
		if err != nil {
			continue
		}
		orders = append(orders, order)
	}

	hasMore := len(orders) > limit
	if hasMore {
		orders = orders[:limit]
	}

	nextCursor := ""
	if hasMore && len(orders) > 0 {
		nextCursor = orders[len(orders)-1].(struct {
			ID          string `json:"id"`
			BuyerID     string `json:"buyer_id"`
			ItemID      string `json:"item_id"`
			Status      string `json:"status"`
			AmountCents int    `json:"amount_cents"`
			Currency    string `json:"currency"`
			CreatedAt   string `json:"created_at"`
		}).CreatedAt
	}

	respondJSON(w, CursorPage{
		Data:       orders,
		NextCursor: nextCursor,
		HasMore:    hasMore,
	})
}
