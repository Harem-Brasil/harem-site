package httpapi

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	httpmw "github.com/harem-brasil/backend/internal/middleware"
)

type Plan struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Slug        string   `json:"slug"`
	Description string   `json:"description,omitempty"`
	Price       float64  `json:"price"`
	Currency    string   `json:"currency"`
	Interval    string   `json:"interval"`
	Features    []string `json:"features,omitempty"`
	IsActive    bool     `json:"is_active"`
}

type Subscription struct {
	ID                 string `json:"id"`
	UserID             string `json:"user_id"`
	PlanID             string `json:"plan_id"`
	Plan               *Plan  `json:"plan,omitempty"`
	Status             string `json:"status"`
	CurrentPeriodStart string `json:"current_period_start"`
	CurrentPeriodEnd   string `json:"current_period_end"`
	CreatedAt          string `json:"created_at"`
}

func (s *Server) handleListPlans(w http.ResponseWriter, r *http.Request) {
	rows, err := s.db.Query(r.Context(),
		`SELECT id, name, slug, description, price, currency, interval, features, is_active 
		 FROM plans WHERE is_active = true ORDER BY price`)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Database error")
		return
	}
	defer rows.Close()

	var plans []any
	for rows.Next() {
		var p Plan
		if err := rows.Scan(&p.ID, &p.Name, &p.Slug, &p.Description, &p.Price, &p.Currency, &p.Interval, &p.Features, &p.IsActive); err == nil {
			plans = append(plans, p)
		}
	}

	respondJSON(w, plans)
}

func (s *Server) handleCreateSubscription(w http.ResponseWriter, r *http.Request) {
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
		PlanID string `json:"plan_id"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	var plan Plan
	err = s.db.QueryRow(r.Context(),
		`SELECT id, name, slug, description, price, currency, interval, features, is_active 
		 FROM plans WHERE id = $1 AND is_active = true`,
		req.PlanID,
	).Scan(&plan.ID, &plan.Name, &plan.Slug, &plan.Description, &plan.Price, &plan.Currency, &plan.Interval, &plan.Features, &plan.IsActive)

	if err != nil {
		if err == pgx.ErrNoRows {
			respondError(w, http.StatusNotFound, "Plan not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "Database error")
		return
	}

	subID := uuid.New().String()

	_, err = s.db.Exec(r.Context(),
		`INSERT INTO subscriptions (id, user_id, plan_id, status) VALUES ($1, $2, $3, 'pending_payment')`,
		subID, user.UserID, plan.ID,
	)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Database error")
		return
	}

	respondCreated(w, Subscription{
		ID:     subID,
		UserID: user.UserID,
		PlanID: plan.ID,
		Plan:   &plan,
		Status: "pending_payment",
	})
}

func (s *Server) handleGetMySubscription(w http.ResponseWriter, r *http.Request) {
	user := httpmw.GetUser(r.Context())
	if user == nil {
		respondError(w, http.StatusUnauthorized, "Not authenticated")
		return
	}

	var sub Subscription
	sub.Plan = &Plan{}
	err := s.db.QueryRow(r.Context(),
		`SELECT s.id, s.user_id, s.plan_id, s.status, s.created_at,
		        p.id, p.name, p.slug, p.description, p.price, p.currency, p.interval, p.features, p.is_active
		 FROM subscriptions s
		 LEFT JOIN plans p ON s.plan_id = p.id
		 WHERE s.user_id = $1 AND s.status IN ('active', 'trialing')
		 ORDER BY s.created_at DESC LIMIT 1`,
		user.UserID,
	).Scan(&sub.ID, &sub.UserID, &sub.PlanID, &sub.Status, &sub.CreatedAt,
		&sub.Plan.ID, &sub.Plan.Name, &sub.Plan.Slug, &sub.Plan.Description, &sub.Plan.Price,
		&sub.Plan.Currency, &sub.Plan.Interval, &sub.Plan.Features, &sub.Plan.IsActive)

	if err != nil {
		if err == pgx.ErrNoRows {
			respondJSON(w, nil)
			return
		}
		respondError(w, http.StatusInternalServerError, "Database error")
		return
	}

	respondJSON(w, sub)
}
