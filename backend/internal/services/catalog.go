package services

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/harem-brasil/backend/internal/domain"
	"github.com/harem-brasil/backend/internal/middleware"
	"github.com/harem-brasil/backend/internal/utils"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func userHasRole(u *middleware.UserClaims, role string) bool {
	for _, r := range u.Roles {
		if r == role {
			return true
		}
	}
	return false
}

func catalogCanEdit(u *middleware.UserClaims, ownerCreatorID string) bool {
	if userHasRole(u, "admin") {
		return true
	}
	return u.UserID == ownerCreatorID
}

// NormalizeCurrencyISO4217 valida código ISO 4217 de 3 letras ASCII maiúsculas.
func NormalizeCurrencyISO4217(raw string) (string, bool) {
	s := strings.TrimSpace(raw)
	if len(s) != 3 {
		return "", false
	}
	s = strings.ToUpper(s)
	for _, c := range s {
		if c < 'A' || c > 'Z' {
			return "", false
		}
	}
	return s, true
}

func validateMediaURLList(urls []string) error {
	for _, u := range urls {
		u = strings.TrimSpace(u)
		if u == "" {
			return domain.Err(http.StatusBadRequest, "Invalid media URL")
		}
		if len(u) > 2048 {
			return domain.Err(http.StatusBadRequest, "Media URL too long")
		}
		parsed, err := url.ParseRequestURI(u)
		if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
			return domain.Err(http.StatusBadRequest, "Invalid media URL")
		}
	}
	return nil
}

func patchCatalogHasAnyField(req *domain.CreatorCatalogPatchRequest) bool {
	if req == nil {
		return false
	}
	return req.Title != nil || req.Description != nil || req.PriceCents != nil ||
		req.Currency != nil || req.Visibility != nil || req.Media != nil
}

func rowToPublicItem(row *domain.CatalogRow) *domain.CreatorCatalogItem {
	media := row.MediaURLs
	if media == nil {
		media = []string{}
	}
	return &domain.CreatorCatalogItem{
		ID:          row.ID,
		Title:       row.Title,
		Description: row.Description,
		PriceCents:  row.PriceCents,
		Currency:    row.Currency,
		Visibility:  row.Visibility,
		Media:       media,
		CreatedAt:   utils.FormatRFC3339UTC(row.CreatedAt),
		UpdatedAt:   utils.FormatRFC3339UTC(row.UpdatedAt),
	}
}

func (s *Services) getCatalogRow(ctx context.Context, itemID string) (*domain.CatalogRow, error) {
	if _, err := uuid.Parse(itemID); err != nil {
		return nil, domain.Err(http.StatusNotFound, "Catalog item not found")
	}

	var row domain.CatalogRow
	err := s.DB.QueryRow(ctx,
		`SELECT id::text, creator_id::text, title, description, price_cents, currency, visibility,
		        COALESCE(media_urls, '{}'), created_at, updated_at
		 FROM creator_catalog
		 WHERE id = $1::uuid AND deleted_at IS NULL`,
		itemID,
	).Scan(&row.ID, &row.CreatorID, &row.Title, &row.Description, &row.PriceCents, &row.Currency,
		&row.Visibility, &row.MediaURLs, &row.CreatedAt, &row.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.Err(http.StatusNotFound, "Catalog item not found")
	}
	if err != nil {
		return nil, domain.Err(500, "Database error")
	}
	return &row, nil
}

// CreateCreatorCatalogItem POST — item ligado ao criador autenticado.
func (s *Services) CreateCreatorCatalogItem(ctx context.Context, user *middleware.UserClaims, req domain.CreatorCatalogCreateRequest) (*domain.CreatorCatalogItem, error) {
	cur, ok := NormalizeCurrencyISO4217(req.Currency)
	if !ok {
		return nil, domain.Err(http.StatusBadRequest, "Invalid currency code")
	}
	media := req.Media
	if media == nil {
		media = []string{}
	}
	if err := validateMediaURLList(media); err != nil {
		return nil, err
	}

	id := uuid.New().String()
	now := time.Now().UTC()

	var createdAt, updatedAt time.Time
	err := s.DB.QueryRow(ctx,
		`INSERT INTO creator_catalog (id, creator_id, title, description, price_cents, currency, visibility, media_urls, created_at, updated_at)
		 VALUES ($1::uuid, $2::uuid, $3, $4, $5, $6, $7, $8::text[], $9, $10)
		 RETURNING created_at, updated_at`,
		id, user.UserID, strings.TrimSpace(req.Title), req.Description, req.PriceCents, cur, req.Visibility, media, now, now,
	).Scan(&createdAt, &updatedAt)
	if err != nil {
		return nil, domain.Err(500, "Failed to create catalog item")
	}

	return &domain.CreatorCatalogItem{
		ID:          id,
		Title:       strings.TrimSpace(req.Title),
		Description: req.Description,
		PriceCents:  req.PriceCents,
		Currency:    cur,
		Visibility:  req.Visibility,
		Media:       media,
		CreatedAt:   utils.FormatRFC3339UTC(createdAt),
		UpdatedAt:   utils.FormatRFC3339UTC(updatedAt),
	}, nil
}

// PatchCreatorCatalogItem PATCH — BOLA: só dono ou admin.
func (s *Services) PatchCreatorCatalogItem(ctx context.Context, user *middleware.UserClaims, itemID string, req domain.CreatorCatalogPatchRequest) (*domain.CreatorCatalogItem, error) {
	if !patchCatalogHasAnyField(&req) {
		return nil, domain.Err(http.StatusBadRequest, "No fields to update")
	}

	row, err := s.getCatalogRow(ctx, itemID)
	if err != nil {
		return nil, err
	}
	if !catalogCanEdit(user, row.CreatorID) {
		if s.Logger != nil {
			s.Logger.Warn("catalog item patch denied",
				"reason", "bola",
				"item_id", itemID,
				"user_id", user.UserID,
			)
		}
		return nil, domain.Err(http.StatusNotFound, "Catalog item not found")
	}

	merged := *row
	if req.Title != nil {
		t := strings.TrimSpace(*req.Title)
		if t == "" {
			return nil, domain.Err(http.StatusBadRequest, "Invalid title")
		}
		merged.Title = t
	}
	if req.Description != nil {
		merged.Description = *req.Description
	}
	if req.PriceCents != nil {
		merged.PriceCents = *req.PriceCents
	}
	if req.Currency != nil {
		cur, ok := NormalizeCurrencyISO4217(*req.Currency)
		if !ok {
			return nil, domain.Err(http.StatusBadRequest, "Invalid currency code")
		}
		merged.Currency = cur
	}
	if req.Visibility != nil {
		merged.Visibility = *req.Visibility
	}
	if req.Media != nil {
		m := *req.Media
		if m == nil {
			m = []string{}
		}
		if vErr := validateMediaURLList(m); vErr != nil {
			return nil, vErr
		}
		merged.MediaURLs = m
	}

	if merged.MediaURLs == nil {
		merged.MediaURLs = []string{}
	}
	err = s.DB.QueryRow(ctx,
		`UPDATE creator_catalog SET
			title = $2,
			description = $3,
			price_cents = $4,
			currency = $5,
			visibility = $6,
			media_urls = $7::text[],
			updated_at = NOW()
		 WHERE id = $1::uuid AND deleted_at IS NULL
		 RETURNING created_at, updated_at`,
		itemID,
		merged.Title,
		merged.Description,
		merged.PriceCents,
		merged.Currency,
		merged.Visibility,
		merged.MediaURLs,
	).Scan(&merged.CreatedAt, &merged.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.Err(http.StatusNotFound, "Catalog item not found")
	}
	if err != nil {
		return nil, domain.Err(500, "Failed to update catalog item")
	}

	return rowToPublicItem(&merged), nil
}

// DeleteCreatorCatalogItem remove o registo na base (DELETE físico). BOLA: só dono ou admin.
// Se existirem linhas em creator_orders a referenciar o item, a FK bloqueia — resposta 409.
func (s *Services) DeleteCreatorCatalogItem(ctx context.Context, user *middleware.UserClaims, itemID string) error {
	row, err := s.getCatalogRow(ctx, itemID)
	if err != nil {
		return err
	}
	if !catalogCanEdit(user, row.CreatorID) {
		if s.Logger != nil {
			s.Logger.Warn("catalog item delete denied",
				"reason", "bola",
				"item_id", itemID,
				"user_id", user.UserID,
			)
		}
		return domain.Err(http.StatusNotFound, "Catalog item not found")
	}

	var tag pgconn.CommandTag
	if userHasRole(user, "admin") {
		tag, err = s.DB.Exec(ctx,
			`DELETE FROM creator_catalog WHERE id = $1::uuid AND deleted_at IS NULL`,
			itemID,
		)
	} else {
		tag, err = s.DB.Exec(ctx,
			`DELETE FROM creator_catalog WHERE id = $1::uuid AND creator_id = $2::uuid AND deleted_at IS NULL`,
			itemID, user.UserID,
		)
	}
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23503" {
			if s.Logger != nil {
				s.Logger.Info("catalog item delete blocked: referenced by orders", "item_id", itemID)
			}
			return domain.Err(http.StatusConflict, "Cannot delete catalog item with existing orders")
		}
		return domain.Err(500, "Failed to delete catalog item")
	}
	if tag.RowsAffected() == 0 {
		return domain.Err(http.StatusNotFound, "Catalog item not found")
	}
	return nil
}
