package domain

import "time"

type CatalogRow struct {
	ID          string
	CreatorID   string
	Title       string
	Description string
	PriceCents  int
	Currency    string
	Visibility  string
	MediaURLs   []string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
