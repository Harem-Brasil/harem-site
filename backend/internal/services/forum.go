package services

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/harem-brasil/backend/internal/domain"
	"github.com/harem-brasil/backend/internal/middleware"
	"github.com/harem-brasil/backend/internal/utils"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

func (s *Services) ListForumCategories(ctx context.Context) ([]any, error) {
	rows, err := s.DB.Query(ctx,
		`SELECT id, name, slug, description, topic_count FROM forum_categories ORDER BY sort_order`)
	if err != nil {
		return nil, domain.Err(500, "Database error")
	}
	defer rows.Close()

	var categories []any
	for rows.Next() {
		var c domain.ForumCategory
		if err := rows.Scan(&c.ID, &c.Name, &c.Slug, &c.Description, &c.PostCount); err == nil {
			categories = append(categories, c)
		}
	}

	return categories, nil
}

func (s *Services) ListForumTopics(ctx context.Context, categoryID, cursor string) (*domain.CursorPage, error) {
	limit := 20

	var rows pgx.Rows
	var err error

	if categoryID != "" {
		rows, err = s.DB.Query(ctx,
			`SELECT t.id, t.category_id, t.author_id, t.title, t.slug, t.reply_count, t.view_count,
			        t.is_pinned, t.is_locked, t.last_reply_at, t.created_at,
			        u.id, u.username, u.role, u.avatar_url
			 FROM forum_topics t
			 JOIN users u ON t.author_id = u.id
			 WHERE t.category_id = $1 AND t.deleted_at IS NULL
			 AND ($2 = '' OR t.last_reply_at < $2)
			 ORDER BY t.is_pinned DESC, t.last_reply_at DESC LIMIT $3`,
			categoryID, cursor, limit+1)
	} else {
		rows, err = s.DB.Query(ctx,
			`SELECT t.id, t.category_id, t.author_id, t.title, t.slug, t.reply_count, t.view_count,
			        t.is_pinned, t.is_locked, t.last_reply_at, t.created_at,
			        u.id, u.username, u.role, u.avatar_url
			 FROM forum_topics t
			 JOIN users u ON t.author_id = u.id
			 WHERE t.deleted_at IS NULL
			 AND ($1 = '' OR t.last_reply_at < $1)
			 ORDER BY t.is_pinned DESC, t.last_reply_at DESC LIMIT $2`,
			cursor, limit+1)
	}

	if err != nil {
		return nil, domain.Err(500, "Database error")
	}
	defer rows.Close()

	var topics []any
	for rows.Next() {
		var tp domain.ForumTopic
		var author domain.UserPublic
		var lastReply pgtype.Timestamptz
		err := rows.Scan(&tp.ID, &tp.CategoryID, &tp.AuthorID, &tp.Title, &tp.Slug, &tp.ReplyCount, &tp.ViewCount,
			&tp.IsPinned, &tp.IsLocked, &lastReply, &tp.CreatedAt,
			&author.ID, &author.Username, &author.Role, &author.AvatarURL)
		if err != nil {
			continue
		}
		if lastReply.Valid {
			tp.LastReplyAt = utils.FormatRFC3339UTC(lastReply.Time)
		}
		tp.Author = author
		topics = append(topics, tp)
	}

	hasMore := len(topics) > limit
	if hasMore {
		topics = topics[:limit]
	}

	nextCursor := ""
	if hasMore && len(topics) > 0 {
		nextCursor = topics[len(topics)-1].(domain.ForumTopic).LastReplyAt
	}

	return &domain.CursorPage{Data: topics, NextCursor: nextCursor, HasMore: hasMore}, nil
}

type CreateForumTopicBody struct {
	CategoryID string `json:"category_id"`
	Title      string `json:"title"`
	Content    string `json:"content"`
}

func (s *Services) CreateForumTopic(ctx context.Context, user *middleware.UserClaims, req CreateForumTopicBody) (*domain.ForumTopic, error) {
	if req.CategoryID == "" || req.Title == "" || req.Content == "" {
		return nil, domain.ErrValidation("category_id, title, and content are required", map[string]string{
			"fields": "category_id, title, and content are required",
		})
	}

	topicID := uuid.New().String()
	slug := utils.Slugify(req.Title)

	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return nil, domain.Err(500, "Failed to start transaction")
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx,
		`INSERT INTO forum_topics (id, category_id, author_id, title, slug) VALUES ($1, $2, $3, $4, $5)`,
		topicID, req.CategoryID, user.UserID, req.Title, slug,
	)
	if err != nil {
		return nil, domain.Err(500, "Failed to create topic")
	}

	postID := uuid.New().String()
	_, err = tx.Exec(ctx,
		`INSERT INTO forum_posts (id, topic_id, author_id, content, is_first_post) VALUES ($1, $2, $3, $4, true)`,
		postID, topicID, user.UserID, req.Content,
	)
	if err != nil {
		return nil, domain.Err(500, "Failed to create first post")
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, domain.Err(500, "Failed to commit transaction")
	}

	return &domain.ForumTopic{
		ID:         topicID,
		CategoryID: req.CategoryID,
		AuthorID:   user.UserID,
		Title:      req.Title,
		Slug:       slug,
	}, nil
}

func (s *Services) GetForumTopic(ctx context.Context, id string) (*domain.ForumTopic, error) {
	var t domain.ForumTopic
	var author domain.UserPublic
	var lastReply pgtype.Timestamptz

	err := s.DB.QueryRow(ctx,
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
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.Err(404, "Topic not found")
		}
		return nil, domain.Err(500, "Database error")
	}

	if lastReply.Valid {
		t.LastReplyAt = utils.FormatRFC3339UTC(lastReply.Time)
	}
	t.Author = author

	return &t, nil
}

type ForumPostBody struct {
	Content string `json:"content"`
}

func (s *Services) CreateForumPost(ctx context.Context, user *middleware.UserClaims, topicID string, req ForumPostBody) (*domain.ForumPost, error) {
	if req.Content == "" {
		return nil, domain.ErrValidation("Content is required", map[string]string{"content": "Content is required"})
	}

	postID := uuid.New().String()

	_, err := s.DB.Exec(ctx,
		`INSERT INTO forum_posts (id, topic_id, author_id, content) VALUES ($1, $2, $3, $4)`,
		postID, topicID, user.UserID, req.Content,
	)
	if err != nil {
		return nil, domain.Err(500, "Failed to create post")
	}

	return &domain.ForumPost{
		ID:       postID,
		TopicID:  topicID,
		AuthorID: user.UserID,
		Content:  req.Content,
	}, nil
}
