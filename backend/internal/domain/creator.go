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

// CreatorCatalogItem resposta pública de item do catálogo (sem campos internos).
type CreatorCatalogItem struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	PriceCents  int      `json:"price_cents"`
	Currency    string   `json:"currency"`
	Visibility  string   `json:"visibility"`
	Media       []string `json:"media"`
	CreatedAt   string   `json:"created_at"`
	UpdatedAt   string   `json:"updated_at"`
}

// CreatorCatalogCreateRequest whitelist POST /creator/catalog (binding explícito).
type CreatorCatalogCreateRequest struct {
	Title       string   `json:"title" binding:"required,min=1,max=200"`
	Description string   `json:"description" binding:"max=10000"`
	PriceCents  int      `json:"price_cents" binding:"required,min=0,max=999999999"`
	Currency    string   `json:"currency" binding:"required,len=3"`
	Visibility  string   `json:"visibility" binding:"required,oneof=public subscribers premium"`
	Media       []string `json:"media" binding:"max=20,dive,max=2048"`
}

// CreatorCatalogPatchRequest whitelist PATCH (apenas campos enviados são atualizados).
type CreatorCatalogPatchRequest struct {
	Title       *string   `json:"title" binding:"omitempty,min=1,max=200"`
	Description *string   `json:"description" binding:"omitempty,max=10000"`
	PriceCents  *int      `json:"price_cents" binding:"omitempty,min=0,max=999999999"`
	Currency    *string   `json:"currency" binding:"omitempty,len=3"`
	Visibility  *string   `json:"visibility" binding:"omitempty,oneof=public subscribers premium"`
	Media       *[]string `json:"media" binding:"omitempty,max=20,dive,max=2048"`
}

type CreatorOrderRow struct {
	ID          string `json:"id"`
	CreatorID   string `json:"creator_id,omitempty"`
	BuyerID     string `json:"buyer_id"`
	ItemID      string `json:"item_id"`
	Status      string `json:"status"`
	AmountCents int    `json:"amount_cents"`
	Currency    string `json:"currency"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}