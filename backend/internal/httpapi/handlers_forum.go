package httpapi

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"unicode"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"

	httpmw "github.com/harem-brasil/backend/internal/middleware"
)

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

func (s *Server) handleListForumCategories(w http.ResponseWriter, r *http.Request) {
	rows, err := s.db.Query(r.Context(),
		`SELECT id, name, slug, description, topic_count FROM forum_categories ORDER BY sort_order`)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Database error")
		return
	}
	defer rows.Close()

	var categories []any
	for rows.Next() {
		var c ForumCategory
		if err := rows.Scan(&c.ID, &c.Name, &c.Slug, &c.Description, &c.PostCount); err == nil {
			categories = append(categories, c)
		}
	}

	respondJSON(w, categories)
}

func (s *Server) handleListForumTopics(w http.ResponseWriter, r *http.Request) {
	categoryID := r.URL.Query().Get("category_id")
	cursor := r.URL.Query().Get("cursor")
	limit := 20

	var rows pgx.Rows
	var err error

	if categoryID != "" {
		rows, err = s.db.Query(r.Context(),
			`SELECT t.id, t.category_id, t.author_id, t.title, t.slug, t.reply_count, t.view_count,
			        t.is_pinned, t.is_locked, t.last_reply_at, t.created_at,
			        u.id, u.username, u.role, u.avatar_url
			 FROM forum_topics t
			 JOIN users u ON t.author_id = u.id
			 WHERE t.category_id = $1 AND t.deleted_at IS NULL
			 AND ($2 = '' OR t.id > $2)
			 ORDER BY t.is_pinned DESC, t.last_reply_at DESC LIMIT $3`,
			categoryID, cursor, limit+1)
	} else {
		rows, err = s.db.Query(r.Context(),
			`SELECT t.id, t.category_id, t.author_id, t.title, t.slug, t.reply_count, t.view_count,
			        t.is_pinned, t.is_locked, t.last_reply_at, t.created_at,
			        u.id, u.username, u.role, u.avatar_url
			 FROM forum_topics t
			 JOIN users u ON t.author_id = u.id
			 WHERE t.deleted_at IS NULL
			 AND ($1 = '' OR t.id > $1)
			 ORDER BY t.is_pinned DESC, t.last_reply_at DESC LIMIT $2`,
			cursor, limit+1)
	}

	if err != nil {
		respondError(w, http.StatusInternalServerError, "Database error")
		return
	}
	defer rows.Close()

	var topics []any
	for rows.Next() {
		var t ForumTopic
		var author UserPublic
		var lastReply pgtype.Timestamptz
		err := rows.Scan(&t.ID, &t.CategoryID, &t.AuthorID, &t.Title, &t.Slug, &t.ReplyCount, &t.ViewCount,
			&t.IsPinned, &t.IsLocked, &lastReply, &t.CreatedAt,
			&author.ID, &author.Username, &author.Role, &author.AvatarURL)
		if err != nil {
			continue
		}
		if lastReply.Valid {
			t.LastReplyAt = formatTimestamp(lastReply.Time)
		}
		t.Author = author
		topics = append(topics, t)
	}

	hasMore := len(topics) > limit
	if hasMore {
		topics = topics[:limit]
	}

	nextCursor := ""
	if hasMore && len(topics) > 0 {
		nextCursor = topics[len(topics)-1].(ForumTopic).ID
	}

	respondJSON(w, CursorPage{
		Data:       topics,
		NextCursor: nextCursor,
		HasMore:    hasMore,
	})
}

func (s *Server) handleCreateForumTopic(w http.ResponseWriter, r *http.Request) {
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
		CategoryID string `json:"category_id"`
		Title      string `json:"title"`
		Content    string `json:"content"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	if req.CategoryID == "" || req.Title == "" || req.Content == "" {
		respondValidationError(w, map[string]string{
			"fields": "category_id, title, and content are required",
		})
		return
	}

	topicID := uuid.New().String()
	slug := slugify(req.Title)

	tx, err := s.db.Begin(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to start transaction")
		return
	}
	defer tx.Rollback(r.Context())

	_, err = tx.Exec(r.Context(),
		`INSERT INTO forum_topics (id, category_id, author_id, title, slug) VALUES ($1, $2, $3, $4, $5)`,
		topicID, req.CategoryID, user.UserID, req.Title, slug,
	)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to create topic")
		return
	}

	postID := uuid.New().String()
	_, err = tx.Exec(r.Context(),
		`INSERT INTO forum_posts (id, topic_id, author_id, content, is_first_post) VALUES ($1, $2, $3, $4, true)`,
		postID, topicID, user.UserID, req.Content,
	)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to create first post")
		return
	}

	if err := tx.Commit(r.Context()); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to commit transaction")
		return
	}

	respondCreated(w, ForumTopic{
		ID:         topicID,
		CategoryID: req.CategoryID,
		AuthorID:   user.UserID,
		Title:      req.Title,
		Slug:       slug,
	})
}

func (s *Server) handleGetForumTopic(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var t ForumTopic
	var author UserPublic
	var lastReply pgtype.Timestamptz

	err := s.db.QueryRow(r.Context(),
		`SELECT t.id, t.category_id, t.author_id, t.title, t.slug, t.reply_count, t.view_count,
		        t.is_pinned, t.is_locked, t.last_reply_at, t.created_at,
		        u.id, u.username, u.role, u.avatar_url
		 FROM forum_topics t
		 JOIN users u ON t.author_id = u.id
		 WHERE t.id = $1 AND t.deleted_at IS NULL`,
		id,
	).Scan(&t.ID, &t.CategoryID, &t.AuthorID, &t.Title, &t.Slug, &t.ReplyCount, &t.ViewCount,
		&t.IsPinned, &t.IsLocked, &lastReply, &t.CreatedAt,
		&author.ID, &author.Username, &author.Role, &author.AvatarURL)

	if err != nil {
		if err == pgx.ErrNoRows {
			respondError(w, http.StatusNotFound, "Topic not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "Database error")
		return
	}

	if lastReply.Valid {
		t.LastReplyAt = formatTimestamp(lastReply.Time)
	}
	t.Author = author

	respondJSON(w, t)
}

func (s *Server) handleCreateForumPost(w http.ResponseWriter, r *http.Request) {
	user := httpmw.GetUser(r.Context())
	if user == nil {
		respondError(w, http.StatusUnauthorized, "Not authenticated")
		return
	}

	topicID := chi.URLParam(r, "id")

	body, err := io.ReadAll(r.Body)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Failed to read request body")
		return
	}

	var req struct {
		Content string `json:"content"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	if req.Content == "" {
		respondValidationError(w, map[string]string{"content": "Content is required"})
		return
	}

	postID := uuid.New().String()

	_, err = s.db.Exec(r.Context(),
		`INSERT INTO forum_posts (id, topic_id, author_id, content) VALUES ($1, $2, $3, $4)`,
		postID, topicID, user.UserID, req.Content,
	)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to create post")
		return
	}

	respondCreated(w, ForumPost{
		ID:       postID,
		TopicID:  topicID,
		AuthorID: user.UserID,
		Content:  req.Content,
	})
}

func slugify(title string) string {
	// Unicode normalization: decompose accented characters and remove combining marks
	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	normalized, _, _ := transform.String(t, title)

	result := ""
	for _, r := range normalized {
		switch {
		case r >= 'a' && r <= 'z' || r >= '0' && r <= '9':
			result += string(r)
		case r >= 'A' && r <= 'Z':
			result += string(r + ('a' - 'A'))
		case r == ' ', r == '-', r == '_':
			result += "-"
		}
	}

	// Clean up multiple consecutive dashes
	for strings.Contains(result, "--") {
		result = strings.ReplaceAll(result, "--", "-")
	}

	// Trim leading/trailing dashes
	result = strings.Trim(result, "-")

	return result
}
