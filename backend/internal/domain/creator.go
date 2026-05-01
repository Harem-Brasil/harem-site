package domain

// --- Creator ---

type CreatorApplication struct {
	ID          string   `json:"id"`
	UserID      string   `json:"user_id"`
	Status      string   `json:"status"`
	Bio         string   `json:"bio"`
	SocialLinks []string `json:"social_links,omitempty"`
	SubmittedAt string   `json:"submitted_at"`
	ReviewedAt  *string  `json:"reviewed_at,omitempty"`
}

type CreatorApplyRequest struct {
	Bio         string   `json:"bio" binding:"omitempty,max=5000"`
	SocialLinks []string `json:"social_links" binding:"omitempty,max=20,dive,max=500"`
}

// CreatorProfilePatchRequest atualização parcial do perfil do criador (whitelist explícita).
type CreatorProfilePatchRequest struct {
	Bio string `json:"bio" binding:"required,max=5000"`
}

type CreatorDashboard struct {
	TotalPosts      int     `json:"total_posts"`
	TotalLikes      int     `json:"total_likes"`
	TotalFollowers  int     `json:"total_followers"`
	SubscriberCount int     `json:"subscriber_count"`
	MonthlyEarnings float64 `json:"monthly_earnings"`
}

type CreatorCatalogItem struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	PriceCents  int    `json:"price_cents"`
	Currency    string `json:"currency"`
	Visibility  string `json:"visibility"`
	CreatedAt   string `json:"created_at"`
}

type CreatorOrderRow struct {
	ID          string `json:"id"`
	BuyerID     string `json:"buyer_id"`
	ItemID      string `json:"item_id"`
	Status      string `json:"status"`
	AmountCents int    `json:"amount_cents"`
	Currency    string `json:"currency"`
	CreatedAt   string `json:"created_at"`
}