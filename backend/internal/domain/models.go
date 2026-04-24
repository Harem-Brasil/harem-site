package domain

import (
	"net/mail"
	"time"
	"unicode"
	"unicode/utf8"
)

// --- Auth / usuário ---

type RegisterRequest struct {
	Email              string `json:"email" binding:"required,max=320"`
	ScreenName         string `json:"screen_name" binding:"required,max=64"`
	Password           string `json:"password" binding:"required,max=128"`
	AcceptTermsVersion string `json:"accept_terms_version" binding:"required,max=32"`
}

// Validate checks RegisterRequest fields and returns field-specific errors.
// The bool indicates whether validation passed (true = no errors).
func (req *RegisterRequest) Validate() (map[string]string, bool) {
	errors := make(map[string]string)
	if req.Email == "" {
		errors["email"] = "Email is required"
	} else if addr, err := mail.ParseAddress(req.Email); err != nil {
		errors["email"] = "Invalid email format"
	} else {
		req.Email = addr.Address
	}
	if req.ScreenName == "" {
		errors["screen_name"] = "Screen name is required"
	} else if msg := validateScreenName(req.ScreenName); msg != "" {
		errors["screen_name"] = msg
	}
	if req.Password == "" {
		errors["password"] = "Password is required"
	} else if msg := validatePassword(req.Password); msg != "" {
		errors["password"] = msg
	}
	if req.AcceptTermsVersion == "" {
		errors["accept_terms_version"] = "Terms acceptance version is required"
	}
	return errors, len(errors) == 0
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,max=320"`
	Password string `json:"password" binding:"required,max=128"`
}

// Validate checks LoginRequest fields and returns field-specific errors.
// The bool indicates whether validation passed (true = no errors).
func (req *LoginRequest) Validate() (map[string]string, bool) {
	errors := make(map[string]string)
	if req.Email == "" {
		errors["email"] = "Email is required"
	} else if addr, err := mail.ParseAddress(req.Email); err != nil {
		errors["email"] = "Invalid email format"
	} else {
		req.Email = addr.Address
	}
	if req.Password == "" {
		errors["password"] = "Password is required"
	}
	return errors, len(errors) == 0
}

func validateScreenName(name string) string {
	if utf8.RuneCountInString(name) < 2 {
		return "Screen name must be at least 2 characters long"
	}
	if utf8.RuneCountInString(name) > 64 {
		return "Screen name must be at most 64 characters long"
	}
	for _, r := range name {
		if !unicode.IsPrint(r) || unicode.IsSpace(r) {
			return "Screen name contains invalid characters"
		}
	}
	return ""
}

func validatePassword(password string) string {
	if utf8.RuneCountInString(password) < 8 {
		return "Password must be at least 8 characters long"
	}
	var hasLower, hasUpper, hasDigit, hasSpecial bool
	for _, r := range password {
		switch {
		case unicode.IsLower(r):
			hasLower = true
		case unicode.IsUpper(r):
			hasUpper = true
		case unicode.IsDigit(r):
			hasDigit = true
		case unicode.IsPunct(r) || unicode.IsSymbol(r):
			hasSpecial = true
		}
	}
	if !hasLower {
		return "Password must contain at least one lowercase letter"
	}
	if !hasUpper {
		return "Password must contain at least one uppercase letter"
	}
	if !hasDigit {
		return "Password must contain at least one number"
	}
	if !hasSpecial {
		return "Password must contain at least one special character"
	}
	return ""
}

type AuthResponse struct {
	AccessToken      string     `json:"access_token"`
	AccessExpiresIn  int64      `json:"access_expires_in"`
	RefreshToken     string     `json:"refresh_token"`
	RefreshExpiresIn int64      `json:"refresh_expires_in"`
	TokenType        string     `json:"token_type"`
	ExpiresAt        time.Time  `json:"expires_at"`
	User             UserPublic `json:"user"`
}

type UserPublic struct {
	ID         string `json:"id"`
	ScreenName string `json:"screen_name"`
	Email      string `json:"email,omitempty"`
	Role       string `json:"role"`
	AvatarURL  string `json:"avatar_url,omitempty"`
	Bio        string `json:"bio,omitempty"`
	CreatedAt  string `json:"created_at"`
}

// --- Posts ---

type PostResponse struct {
	ID         string     `json:"id"`
	AuthorID   string     `json:"author_id"`
	Content    string     `json:"content"`
	MediaURLs  []string   `json:"media_urls,omitempty"`
	Visibility string     `json:"visibility"`
	LikeCount  int        `json:"like_count"`
	CreatedAt  string     `json:"created_at"`
	UpdatedAt  string     `json:"updated_at"`
	Author     UserPublic `json:"author,omitempty"`
}

type CreatePostRequest struct {
	Content    string   `json:"content"`
	MediaURLs  []string `json:"media_urls,omitempty"`
	Visibility string   `json:"visibility"`
}

type CommentResponse struct {
	ID        string     `json:"id"`
	PostID    string     `json:"post_id"`
	AuthorID  string     `json:"author_id"`
	Content   string     `json:"content"`
	CreatedAt string     `json:"created_at"`
	Author    UserPublic `json:"author,omitempty"`
}

// --- Fórum ---

type ForumCategory struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description,omitempty"`
	PostCount   int    `json:"post_count"`
}

type ForumTopic struct {
	ID          string     `json:"id"`
	CategoryID  string     `json:"category_id"`
	AuthorID    string     `json:"author_id"`
	Title       string     `json:"title"`
	Slug        string     `json:"slug"`
	ReplyCount  int        `json:"reply_count"`
	ViewCount   int        `json:"view_count"`
	IsPinned    bool       `json:"is_pinned"`
	IsLocked    bool       `json:"is_locked"`
	LastReplyAt string     `json:"last_reply_at,omitempty"`
	CreatedAt   string     `json:"created_at"`
	Author      UserPublic `json:"author,omitempty"`
}

type ForumPost struct {
	ID        string     `json:"id"`
	TopicID   string     `json:"topic_id"`
	AuthorID  string     `json:"author_id"`
	Content   string     `json:"content"`
	CreatedAt string     `json:"created_at"`
	UpdatedAt string     `json:"updated_at,omitempty"`
	Author    UserPublic `json:"author,omitempty"`
}

// --- Chat ---

type ChatRoom struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description,omitempty"`
	CreatedBy   string `json:"created_by"`
	MemberCount int    `json:"member_count"`
	CreatedAt   string `json:"created_at"`
	IsMember    bool   `json:"is_member"`
}

type ChatMessage struct {
	ID        string     `json:"id"`
	RoomID    string     `json:"room_id"`
	SenderID  string     `json:"sender_id"`
	Content   string     `json:"content"`
	CreatedAt string     `json:"created_at"`
	Sender    UserPublic `json:"sender,omitempty"`
}

// --- Notificações ---

type Notification struct {
	ID        string         `json:"id"`
	UserID    string         `json:"user_id"`
	Type      string         `json:"type"`
	Title     string         `json:"title"`
	Body      string         `json:"body"`
	Data      map[string]any `json:"data,omitempty"`
	ReadAt    *string        `json:"read_at,omitempty"`
	CreatedAt string         `json:"created_at"`
}

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

type CreatorDashboard struct {
	TotalPosts      int     `json:"total_posts"`
	TotalLikes      int     `json:"total_likes"`
	TotalFollowers  int     `json:"total_followers"`
	SubscriberCount int     `json:"subscriber_count"`
	MonthlyEarnings float64 `json:"monthly_earnings"`
}

// --- Billing ---

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

// --- Media ---

type UploadSession struct {
	ID            string `json:"id"`
	Status        string `json:"status"`
	UploadURL     string `json:"upload_url"`
	ContentType   string `json:"content_type,omitempty"`
	ContentLength int64  `json:"content_length,omitempty"`
	ExpiresAt     string `json:"expires_at,omitempty"`
}

// --- Admin ---

type AuditLogEntry struct {
	ID        string         `json:"id"`
	UserID    string         `json:"user_id"`
	Action    string         `json:"action"`
	Resource  string         `json:"resource"`
	Details   map[string]any `json:"details,omitempty"`
	IP        string         `json:"ip,omitempty"`
	CreatedAt string         `json:"created_at"`
}
