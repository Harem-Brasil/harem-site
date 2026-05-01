package services

import (
	"context"
	"crypto/subtle"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/harem-brasil/backend/internal/domain"
	"github.com/harem-brasil/backend/internal/middleware"
	"github.com/harem-brasil/backend/internal/utils"
	"github.com/jackc/pgx/v5"
)

const catalogOrderPaymentRefMaxLen = 256

// CreateCatalogOrder — comprador cria pedido em requested (validação catálogo + anti–auto-compra).
func (s *Services) CreateCatalogOrder(ctx context.Context, user *middleware.UserClaims, itemID string) (*domain.CreatorOrderRow, error) {
	itemID = strings.TrimSpace(itemID)
	if _, err := uuid.Parse(itemID); err != nil {
		return nil, domain.Err(http.StatusBadRequest, "Invalid item id")
	}

	var creatorID string
	var priceCents int
	var currency string
	err := s.DB.QueryRow(ctx,
		`SELECT creator_id::text, price_cents, currency FROM creator_catalog
		 WHERE id = $1::uuid AND deleted_at IS NULL`,
		itemID,
	).Scan(&creatorID, &priceCents, &currency)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.Err(http.StatusNotFound, "Catalog item not found")
	}
	if err != nil {
		return nil, domain.Err(500, "Database error")
	}
	if creatorID == user.UserID {
		return nil, domain.Err(http.StatusConflict, "Cannot order your own catalog item")
	}

	orderID := uuid.New().String()
	now := time.Now().UTC()

	var createdAt, updatedAt time.Time
	err = s.DB.QueryRow(ctx,
		`INSERT INTO creator_orders (id, creator_id, buyer_id, item_id, status, amount_cents, currency, created_at, updated_at)
		 VALUES ($1::uuid, $2::uuid, $3::uuid, $4::uuid, $5, $6, $7, $8, $9)
		 RETURNING created_at, updated_at`,
		orderID, creatorID, user.UserID, itemID, domain.OrderStatusRequested, priceCents, currency, now, now,
	).Scan(&createdAt, &updatedAt)
	if err != nil {
		return nil, domain.Err(500, "Failed to create order")
	}

	return &domain.CreatorOrderRow{
		ID:          orderID,
		CreatorID:   creatorID,
		BuyerID:     user.UserID,
		ItemID:      itemID,
		Status:      domain.OrderStatusRequested,
		AmountCents: priceCents,
		Currency:    currency,
		CreatedAt:   utils.FormatRFC3339UTC(createdAt),
		UpdatedAt:   utils.FormatRFC3339UTC(updatedAt),
	}, nil
}

// ListBuyerCatalogOrders — BOLA: apenas pedidos do utilizador autenticado.
func (s *Services) ListBuyerCatalogOrders(ctx context.Context, user *middleware.UserClaims, cursor string) (*domain.CursorPage, error) {
	limit := 20

	rows, err := s.DB.Query(ctx,
		`SELECT id::text, creator_id::text, buyer_id::text, item_id::text, status, amount_cents, currency, created_at, updated_at
		 FROM creator_orders
		 WHERE buyer_id = $1::uuid
		 AND ($2::text = '' OR created_at < $2::timestamptz)
		 ORDER BY created_at DESC LIMIT $3`,
		user.UserID, cursor, limit+1,
	)
	if err != nil {
		return nil, domain.Err(500, "Database error")
	}
	defer rows.Close()

	var orders []any
	for rows.Next() {
		var o domain.CreatorOrderRow
		var createdAt, updatedAt time.Time
		if err := rows.Scan(&o.ID, &o.CreatorID, &o.BuyerID, &o.ItemID, &o.Status, &o.AmountCents, &o.Currency, &createdAt, &updatedAt); err != nil {
			continue
		}
		o.CreatedAt = utils.FormatRFC3339UTC(createdAt)
		o.UpdatedAt = utils.FormatRFC3339UTC(updatedAt)
		orders = append(orders, o)
	}

	hasMore := len(orders) > limit
	if hasMore {
		orders = orders[:limit]
	}
	nextCursor := ""
	if hasMore && len(orders) > 0 {
		nextCursor = orders[len(orders)-1].(domain.CreatorOrderRow).CreatedAt
	}
	return &domain.CursorPage{Data: orders, NextCursor: nextCursor, HasMore: hasMore}, nil
}

// GetCreatorOrders — pedidos onde o utilizador é o criador do item (já existia; enriquecido com timestamps).
func (s *Services) GetCreatorOrders(ctx context.Context, user *middleware.UserClaims, cursor string) (*domain.CursorPage, error) {
	limit := 20

	rows, err := s.DB.Query(ctx,
		`SELECT id::text, creator_id::text, buyer_id::text, item_id::text, status, amount_cents, currency, created_at, updated_at
		 FROM creator_orders
		 WHERE creator_id = $1::uuid
		 AND ($2::text = '' OR created_at < $2::timestamptz)
		 ORDER BY created_at DESC LIMIT $3`,
		user.UserID, cursor, limit+1,
	)
	if err != nil {
		return nil, domain.Err(500, "Database error")
	}
	defer rows.Close()

	var orders []any
	for rows.Next() {
		var o domain.CreatorOrderRow
		var createdAt, updatedAt time.Time
		if err := rows.Scan(&o.ID, &o.CreatorID, &o.BuyerID, &o.ItemID, &o.Status, &o.AmountCents, &o.Currency, &createdAt, &updatedAt); err != nil {
			continue
		}
		o.CreatedAt = utils.FormatRFC3339UTC(createdAt)
		o.UpdatedAt = utils.FormatRFC3339UTC(updatedAt)
		orders = append(orders, o)
	}

	hasMore := len(orders) > limit
	if hasMore {
		orders = orders[:limit]
	}
	nextCursor := ""
	if hasMore && len(orders) > 0 {
		nextCursor = orders[len(orders)-1].(domain.CreatorOrderRow).CreatedAt
	}
	return &domain.CursorPage{Data: orders, NextCursor: nextCursor, HasMore: hasMore}, nil
}

// GetCatalogOrder — BOLA: comprador, criador do pedido ou admin.
func (s *Services) GetCatalogOrder(ctx context.Context, user *middleware.UserClaims, orderID string) (*domain.CreatorOrderRow, error) {
	orderID = strings.TrimSpace(orderID)
	if _, err := uuid.Parse(orderID); err != nil {
		return nil, domain.Err(http.StatusBadRequest, "Invalid order id")
	}

	var o domain.CreatorOrderRow
	var createdAt, updatedAt time.Time
	err := s.DB.QueryRow(ctx,
		`SELECT id::text, creator_id::text, buyer_id::text, item_id::text, status, amount_cents, currency, created_at, updated_at
		 FROM creator_orders WHERE id = $1::uuid`,
		orderID,
	).Scan(&o.ID, &o.CreatorID, &o.BuyerID, &o.ItemID, &o.Status, &o.AmountCents, &o.Currency, &createdAt, &updatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.Err(http.StatusNotFound, "Order not found")
	}
	if err != nil {
		return nil, domain.Err(500, "Database error")
	}

	if !userHasRole(user, "admin") && user.UserID != o.BuyerID && user.UserID != o.CreatorID {
		if s.Logger != nil {
			s.Logger.Warn("catalog order access denied",
				"order_id", orderID,
				"subject_user_id", user.UserID,
			)
		}
		return nil, domain.Err(http.StatusNotFound, "Order not found")
	}

	o.CreatedAt = utils.FormatRFC3339UTC(createdAt)
	o.UpdatedAt = utils.FormatRFC3339UTC(updatedAt)
	return &o, nil
}

// CheckoutCatalogOrder — comprador: requested → awaiting_payment (início de fluxo checkout/billing).
func (s *Services) CheckoutCatalogOrder(ctx context.Context, user *middleware.UserClaims, orderID string) (*domain.CreatorOrderRow, error) {
	orderID = strings.TrimSpace(orderID)
	if _, err := uuid.Parse(orderID); err != nil {
		return nil, domain.Err(http.StatusBadRequest, "Invalid order id")
	}

	o, err := s.transitionCatalogOrder(ctx, orderID, user.UserID, transitionBuyer, domain.OrderStatusRequested, domain.OrderStatusAwaitingPayment)
	if err != nil {
		return nil, err
	}
	return o, nil
}

// CancelBuyerCatalogOrder — comprador cancela em requested ou awaiting_payment.
func (s *Services) CancelBuyerCatalogOrder(ctx context.Context, user *middleware.UserClaims, orderID string) (*domain.CreatorOrderRow, error) {
	orderID = strings.TrimSpace(orderID)
	if _, err := uuid.Parse(orderID); err != nil {
		return nil, domain.Err(http.StatusBadRequest, "Invalid order id")
	}

	fromStatus, err := s.loadCatalogOrderStatusForBuyer(ctx, orderID, user.UserID)
	if err != nil {
		return nil, err
	}
	if !domain.CanTransitionCatalogOrder(fromStatus, domain.OrderStatusCanceled) {
		return nil, domain.Err(http.StatusConflict, "Order cannot be canceled in current status")
	}
	o, err := s.transitionCatalogOrder(ctx, orderID, user.UserID, transitionBuyer, fromStatus, domain.OrderStatusCanceled)
	if err != nil {
		return nil, err
	}
	return o, nil
}

// FulfillCatalogOrder — criador: paid → fulfilled.
func (s *Services) FulfillCatalogOrder(ctx context.Context, user *middleware.UserClaims, orderID string) (*domain.CreatorOrderRow, error) {
	orderID = strings.TrimSpace(orderID)
	if _, err := uuid.Parse(orderID); err != nil {
		return nil, domain.Err(http.StatusBadRequest, "Invalid order id")
	}

	fromStatus, err := s.loadCatalogOrderStatusForCreator(ctx, orderID, user.UserID)
	if err != nil {
		return nil, err
	}
	if !domain.CanTransitionCatalogOrder(fromStatus, domain.OrderStatusFulfilled) {
		return nil, domain.Err(http.StatusConflict, "Order cannot be fulfilled in current status")
	}
	o, err := s.transitionCatalogOrder(ctx, orderID, user.UserID, transitionCreator, fromStatus, domain.OrderStatusFulfilled)
	if err != nil {
		return nil, err
	}
	return o, nil
}

type transitionActor int

const (
	transitionBuyer transitionActor = iota
	transitionCreator
)

func (s *Services) transitionCatalogOrder(ctx context.Context, orderID string, actorUserID string, actor transitionActor, fromStatus, toStatus string) (*domain.CreatorOrderRow, error) {
	if !domain.CanTransitionCatalogOrder(fromStatus, toStatus) {
		return nil, domain.Err(http.StatusConflict, "Invalid order status transition")
	}

	var q string
	switch actor {
	case transitionBuyer:
		q = `UPDATE creator_orders SET status = $2, updated_at = NOW()
		     WHERE id = $1::uuid AND buyer_id = $3::uuid AND status = $4
		     RETURNING id::text, creator_id::text, buyer_id::text, item_id::text, status, amount_cents, currency, created_at, updated_at`
	case transitionCreator:
		q = `UPDATE creator_orders SET status = $2, updated_at = NOW()
		     WHERE id = $1::uuid AND creator_id = $3::uuid AND status = $4
		     RETURNING id::text, creator_id::text, buyer_id::text, item_id::text, status, amount_cents, currency, created_at, updated_at`
	default:
		return nil, domain.Err(500, "Invalid transition")
	}

	var o domain.CreatorOrderRow
	var createdAt, updatedAt time.Time
	err := s.DB.QueryRow(ctx, q, orderID, toStatus, actorUserID, fromStatus).Scan(
		&o.ID, &o.CreatorID, &o.BuyerID, &o.ItemID, &o.Status, &o.AmountCents, &o.Currency, &createdAt, &updatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.Err(http.StatusConflict, "Order status changed concurrently; retry or refresh")
	}
	if err != nil {
		return nil, domain.Err(500, "Database error")
	}

	o.CreatedAt = utils.FormatRFC3339UTC(createdAt)
	o.UpdatedAt = utils.FormatRFC3339UTC(updatedAt)
	if s.Logger != nil {
		s.Logger.Info("catalog order transitioned",
			"order_id", orderID,
			"from_status", fromStatus,
			"to_status", toStatus,
		)
	}
	return &o, nil
}

func (s *Services) loadCatalogOrderStatusForBuyer(ctx context.Context, orderID, buyerID string) (string, error) {
	var st string
	err := s.DB.QueryRow(ctx,
		`SELECT status FROM creator_orders WHERE id = $1::uuid AND buyer_id = $2::uuid`,
		orderID, buyerID,
	).Scan(&st)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", domain.Err(http.StatusNotFound, "Order not found")
	}
	if err != nil {
		return "", domain.Err(500, "Database error")
	}
	return st, nil
}

func (s *Services) loadCatalogOrderStatusForCreator(ctx context.Context, orderID, creatorID string) (string, error) {
	var st string
	err := s.DB.QueryRow(ctx,
		`SELECT status FROM creator_orders WHERE id = $1::uuid AND creator_id = $2::uuid`,
		orderID, creatorID,
	).Scan(&st)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", domain.Err(http.StatusNotFound, "Order not found")
	}
	if err != nil {
		return "", domain.Err(500, "Database error")
	}
	return st, nil
}

// ApplyBillingPaidCatalogOrder — callback interno (fila billing): awaiting_payment → paid.
// Idempotente: mesmo payment_ref repetido devolve sucesso sem erro; payment_ref diferente em já pago → conflito explícito.
func (s *Services) ApplyBillingPaidCatalogOrder(ctx context.Context, orderID, paymentRef string) (*domain.CreatorOrderRow, error) {
	orderID = strings.TrimSpace(orderID)
	paymentRef = strings.TrimSpace(paymentRef)
	if _, err := uuid.Parse(orderID); err != nil {
		return nil, domain.Err(http.StatusBadRequest, "Invalid order id")
	}
	if paymentRef == "" || len(paymentRef) > catalogOrderPaymentRefMaxLen {
		return nil, domain.Err(http.StatusBadRequest, "Invalid payment_ref")
	}

	var curStatus string
	var existingRef *string
	err := s.DB.QueryRow(ctx,
		`SELECT status, billing_payment_ref FROM creator_orders WHERE id = $1::uuid`,
		orderID,
	).Scan(&curStatus, &existingRef)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.Err(http.StatusNotFound, "Order not found")
	}
	if err != nil {
		return nil, domain.Err(500, "Database error")
	}

	if curStatus == domain.OrderStatusPaid {
		if existingRef != nil && *existingRef == paymentRef {
			return s.getCatalogOrderRowByID(ctx, orderID)
		}
		if s.Logger != nil {
			s.Logger.Warn("billing paid callback conflict: order already paid with different ref",
				"order_id", orderID,
			)
		}
		return nil, domain.Err(http.StatusConflict, "Order already marked paid with a different payment reference")
	}

	if curStatus != domain.OrderStatusAwaitingPayment {
		return nil, domain.Err(http.StatusConflict, "Order is not awaiting payment")
	}

	var o domain.CreatorOrderRow
	var createdAt, updatedAt time.Time
	err = s.DB.QueryRow(ctx,
		`UPDATE creator_orders
		 SET status = $2, billing_payment_ref = $3, updated_at = NOW()
		 WHERE id = $1::uuid AND status = $4
		 RETURNING id::text, creator_id::text, buyer_id::text, item_id::text, status, amount_cents, currency, created_at, updated_at`,
		orderID, domain.OrderStatusPaid, paymentRef, domain.OrderStatusAwaitingPayment,
	).Scan(&o.ID, &o.CreatorID, &o.BuyerID, &o.ItemID, &o.Status, &o.AmountCents, &o.Currency, &createdAt, &updatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.Err(http.StatusConflict, "Order status changed concurrently; payment not applied")
	}
	if err != nil {
		return nil, domain.Err(500, "Database error")
	}

	o.CreatedAt = utils.FormatRFC3339UTC(createdAt)
	o.UpdatedAt = utils.FormatRFC3339UTC(updatedAt)
	if s.Logger != nil {
		s.Logger.Info("catalog order marked paid via billing",
			"order_id", orderID,
		)
	}
	return &o, nil
}

func (s *Services) getCatalogOrderRowByID(ctx context.Context, orderID string) (*domain.CreatorOrderRow, error) {
	var o domain.CreatorOrderRow
	var createdAt, updatedAt time.Time
	err := s.DB.QueryRow(ctx,
		`SELECT id::text, creator_id::text, buyer_id::text, item_id::text, status, amount_cents, currency, created_at, updated_at
		 FROM creator_orders WHERE id = $1::uuid`,
		orderID,
	).Scan(&o.ID, &o.CreatorID, &o.BuyerID, &o.ItemID, &o.Status, &o.AmountCents, &o.Currency, &createdAt, &updatedAt)
	if err != nil {
		return nil, domain.Err(500, "Database error")
	}
	o.CreatedAt = utils.FormatRFC3339UTC(createdAt)
	o.UpdatedAt = utils.FormatRFC3339UTC(updatedAt)
	return &o, nil
}

// ValidateInternalBillingSecret compara o segredo do worker/fila com env (tempo constante).
func (s *Services) ValidateInternalBillingSecret(header string) bool {
	if s.InternalBillingSecret == "" {
		return s.AppEnv == "development" || s.AppEnv == "test"
	}
	if header == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(header), []byte(s.InternalBillingSecret)) == 1
}
