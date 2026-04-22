package services

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/harem-brasil/backend/internal/domain"
	"github.com/harem-brasil/backend/internal/middleware"
	"github.com/jackc/pgx/v5"
)

func (s *Services) ListPlans(ctx context.Context) ([]any, error) {
	rows, err := s.DB.Query(ctx,
		`SELECT id, name, slug, description, price, currency, interval, features, is_active 
		 FROM plans WHERE is_active = true ORDER BY price`)
	if err != nil {
		return nil, domain.Err(500, "Database error")
	}
	defer rows.Close()

	var plans []any
	for rows.Next() {
		var p domain.Plan
		if err := rows.Scan(&p.ID, &p.Name, &p.Slug, &p.Description, &p.Price, &p.Currency, &p.Interval, &p.Features, &p.IsActive); err == nil {
			plans = append(plans, p)
		}
	}

	return plans, nil
}

type CreateSubscriptionBody struct {
	PlanID string `json:"plan_id"`
}

func (s *Services) CreateSubscription(ctx context.Context, user *middleware.UserClaims, req CreateSubscriptionBody) (*domain.Subscription, error) {
	var plan domain.Plan
	err := s.DB.QueryRow(ctx,
		`SELECT id, name, slug, description, price, currency, interval, features, is_active 
		 FROM plans WHERE id = $1 AND is_active = true`,
		req.PlanID,
	).Scan(&plan.ID, &plan.Name, &plan.Slug, &plan.Description, &plan.Price, &plan.Currency, &plan.Interval, &plan.Features, &plan.IsActive)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.Err(404, "Plan not found")
		}
		return nil, domain.Err(500, "Database error")
	}

	subID := uuid.New().String()

	_, err = s.DB.Exec(ctx,
		`INSERT INTO subscriptions (id, user_id, plan_id, status) VALUES ($1, $2, $3, 'pending_payment')`,
		subID, user.UserID, plan.ID,
	)
	if err != nil {
		return nil, domain.Err(500, "Database error")
	}

	return &domain.Subscription{
		ID:     subID,
		UserID: user.UserID,
		PlanID: plan.ID,
		Plan:   &plan,
		Status: "pending_payment",
	}, nil
}

func (s *Services) GetMySubscription(ctx context.Context, user *middleware.UserClaims) (*domain.Subscription, error) {
	var sub domain.Subscription
	sub.Plan = &domain.Plan{}
	err := s.DB.QueryRow(ctx,
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
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, domain.Err(500, "Database error")
	}

	return &sub, nil
}

type BillingCheckoutBody struct {
	PlanID string `json:"plan_id"`
}

func (s *Services) BillingCheckout(ctx context.Context, user *middleware.UserClaims, req BillingCheckoutBody) (map[string]any, error) {
	if req.PlanID == "" {
		return nil, domain.ErrValidation("plan_id required", map[string]string{"plan_id": "Required"})
	}

	var plan domain.Plan
	err := s.DB.QueryRow(ctx,
		`SELECT id, name, slug, description, price, currency, interval, features, is_active 
		 FROM plans WHERE id = $1 AND is_active = true`,
		req.PlanID,
	).Scan(&plan.ID, &plan.Name, &plan.Slug, &plan.Description, &plan.Price, &plan.Currency, &plan.Interval, &plan.Features, &plan.IsActive)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.Err(404, "Plan not found")
		}
		return nil, domain.Err(500, "Database error")
	}

	sessionID := uuid.New().String()

	return map[string]any{
		"checkout_session_id": sessionID,
		"plan":                plan,
		"payment_url":         "/payment/not-implemented",
		"status":              "pending",
	}, nil
}

func (s *Services) CancelSubscription(ctx context.Context, user *middleware.UserClaims) error {
	_, err := s.DB.Exec(ctx,
		`UPDATE subscriptions SET status = 'canceled', updated_at = NOW() 
		 WHERE user_id = $1 AND status = 'active'`,
		user.UserID,
	)
	if err != nil {
		return domain.Err(500, "Failed to cancel subscription")
	}
	return nil
}

func (s *Services) ResumeSubscription(ctx context.Context, user *middleware.UserClaims, subID string) error {
	result, err := s.DB.Exec(ctx,
		`UPDATE subscriptions SET status = 'active', updated_at = NOW() 
		 WHERE id = $1 AND user_id = $2 AND status = 'canceled'`,
		subID, user.UserID,
	)
	if err != nil {
		return domain.Err(500, "Failed to resume subscription")
	}

	if result.RowsAffected() == 0 {
		return domain.Err(404, "Canceled subscription not found")
	}

	return nil
}
