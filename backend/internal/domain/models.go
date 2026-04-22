package domain

import "time"

// --- Auth / usuário ---

type RegisterRequest struct {
	Email    string `json:"email" binding:"required,max=320"`
	Username string `json:"username" binding:"required,max=64"`
	Password string `json:"password" binding:"required,max=128"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,max=320"`
	Password string `json:"password" binding:"required,max=128"`
}

type AuthResponse struct {
	AccessToken  string     `json:"access_token"`
	RefreshToken string     `json:"refresh_token"`
	TokenType    string     `json:"token_type"`
	ExpiresAt    time.Time  `json:"expires_at"`
	User         UserPublic `json:"user"`
}

type UserPublic struct {
	ID        string `json:"id"`
	Username  string `json:"username"`
	Email     string `json:"email,omitempty"`
	Role      string `json:"role"`
	AvatarURL string `json:"avatar_url,omitempty"`
	Bio       string `json:"bio,omitempty"`
	CreatedAt string `json:"created_at"`
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
