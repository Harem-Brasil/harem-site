package services

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/harem-brasil/backend/internal/domain"
	"github.com/harem-brasil/backend/internal/middleware"
	"github.com/jackc/pgx/v5"
)

func (s *Services) ListPosts(ctx context.Context, cursor string) (*domain.CursorPage, error) {
	limit := 20

	rows, err := s.DB.Query(ctx,
		`SELECT p.id, p.author_id, p.content, p.media_urls, p.visibility, p.like_count, p.created_at, p.updated_at,
		        u.id, u.username, u.role, u.avatar_url
		 FROM posts p
		 JOIN users u ON p.author_id = u.id
		 WHERE p.deleted_at IS NULL AND p.visibility = 'public'
		 AND ($1 = '' OR p.created_at < $1)
		 ORDER BY p.created_at DESC LIMIT $2`,
		cursor, limit+1,
	)
	if err != nil {
		return nil, domain.Err(500, "Database error")
	}
	defer rows.Close()

	var posts []any
	for rows.Next() {
		var p domain.PostResponse
		var author domain.UserPublic
		err := rows.Scan(&p.ID, &p.AuthorID, &p.Content, &p.MediaURLs, &p.Visibility, &p.LikeCount,
			&p.CreatedAt, &p.UpdatedAt, &author.ID, &author.Username, &author.Role, &author.AvatarURL)
		if err != nil {
			continue
		}
		p.Author = author
		posts = append(posts, p)
	}

	hasMore := len(posts) > limit
	if hasMore {
		posts = posts[:limit]
	}

	nextCursor := ""
	if hasMore && len(posts) > 0 {
		nextCursor = posts[len(posts)-1].(domain.PostResponse).CreatedAt
	}

	return &domain.CursorPage{Data: posts, NextCursor: nextCursor, HasMore: hasMore}, nil
}

func (s *Services) GetPost(ctx context.Context, id string) (*domain.PostResponse, error) {
	var p domain.PostResponse
	var author domain.UserPublic

	err := s.DB.QueryRow(ctx,
		`SELECT p.id, p.author_id, p.content, p.media_urls, p.visibility, p.like_count, p.created_at, p.updated_at,
		        u.id, u.username, u.role, u.avatar_url
		 FROM posts p
		 JOIN users u ON p.author_id = u.id
		 WHERE p.id = $1 AND p.deleted_at IS NULL`,
		id,
	).Scan(&p.ID, &p.AuthorID, &p.Content, &p.MediaURLs, &p.Visibility, &p.LikeCount,
		&p.CreatedAt, &p.UpdatedAt, &author.ID, &author.Username, &author.Role, &author.AvatarURL)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.Err(404, "Post not found")
		}
		return nil, domain.Err(500, "Database error")
	}

	p.Author = author
	return &p, nil
}

func (s *Services) CreatePost(ctx context.Context, user *middleware.UserClaims, req domain.CreatePostRequest) (*domain.PostResponse, error) {
	if req.Visibility == "" {
		req.Visibility = "public"
	}

	if req.Content == "" && len(req.MediaURLs) == 0 {
		return nil, domain.ErrValidation("One or more fields failed validation", map[string]string{
			"content": "Content or media required",
		})
	}

	postID := uuid.New().String()

	_, err := s.DB.Exec(ctx,
		`INSERT INTO posts (id, author_id, content, media_urls, visibility) VALUES ($1, $2, $3, $4, $5)`,
		postID, user.UserID, req.Content, req.MediaURLs, req.Visibility,
	)
	if err != nil {
		return nil, domain.Err(500, "Failed to create post")
	}

	return &domain.PostResponse{
		ID:         postID,
		AuthorID:   user.UserID,
		Content:    req.Content,
		MediaURLs:  req.MediaURLs,
		Visibility: req.Visibility,
	}, nil
}

func (s *Services) UpdatePost(ctx context.Context, user *middleware.UserClaims, id string, updates map[string]any) error {
	var authorID string
	err := s.DB.QueryRow(ctx, `SELECT author_id FROM posts WHERE id = $1 AND deleted_at IS NULL`, id).Scan(&authorID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Err(404, "Post not found")
		}
		return domain.Err(500, "Database error")
	}

	if authorID != user.UserID {
		return domain.Err(403, "Not authorized to update this post")
	}

	_, err = s.DB.Exec(ctx,
		`UPDATE posts SET content = COALESCE($1, content), visibility = COALESCE($2, visibility), updated_at = NOW() WHERE id = $3`,
		updates["content"], updates["visibility"], id,
	)
	if err != nil {
		return domain.Err(500, "Failed to update post")
	}
	return nil
}

func (s *Services) DeletePost(ctx context.Context, user *middleware.UserClaims, id string) error {
	var authorID string
	err := s.DB.QueryRow(ctx, `SELECT author_id FROM posts WHERE id = $1 AND deleted_at IS NULL`, id).Scan(&authorID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Err(404, "Post not found")
		}
		return domain.Err(500, "Database error")
	}

	if authorID != user.UserID {
		return domain.Err(403, "Not authorized to delete this post")
	}

	_, err = s.DB.Exec(ctx, `UPDATE posts SET deleted_at = NOW() WHERE id = $1`, id)
	if err != nil {
		return domain.Err(500, "Failed to delete post")
	}
	return nil
}

func (s *Services) LikePost(ctx context.Context, user *middleware.UserClaims, id string) error {
	_, err := s.DB.Exec(ctx,
		`INSERT INTO post_likes (post_id, user_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
		id, user.UserID,
	)
	if err != nil {
		return domain.Err(500, "Failed to like post")
	}
	return nil
}

func (s *Services) UnlikePost(ctx context.Context, user *middleware.UserClaims, id string) error {
	_, err := s.DB.Exec(ctx,
		`DELETE FROM post_likes WHERE post_id = $1 AND user_id = $2`,
		id, user.UserID,
	)
	if err != nil {
		return domain.Err(500, "Failed to unlike post")
	}
	return nil
}

func (s *Services) ListComments(ctx context.Context, postID, cursor string) (*domain.CursorPage, error) {
	limit := 20

	rows, err := s.DB.Query(ctx,
		`SELECT c.id, c.post_id, c.author_id, c.content, c.created_at,
		        u.id, u.username, u.role, u.avatar_url
		 FROM post_comments c
		 JOIN users u ON c.author_id = u.id
		 WHERE c.post_id = $1 AND c.deleted_at IS NULL
		 AND ($2 = '' OR c.created_at < $2)
		 ORDER BY c.created_at DESC LIMIT $3`,
		postID, cursor, limit+1,
	)
	if err != nil {
		return nil, domain.Err(500, "Database error")
	}
	defer rows.Close()

	var comments []any
	for rows.Next() {
		var c domain.CommentResponse
		var author domain.UserPublic
		err := rows.Scan(&c.ID, &c.PostID, &c.AuthorID, &c.Content, &c.CreatedAt,
			&author.ID, &author.Username, &author.Role, &author.AvatarURL)
		if err != nil {
			continue
		}
		c.Author = author
		comments = append(comments, c)
	}

	hasMore := len(comments) > limit
	if hasMore {
		comments = comments[:limit]
	}

	nextCursor := ""
	if hasMore && len(comments) > 0 {
		nextCursor = comments[len(comments)-1].(domain.CommentResponse).CreatedAt
	}

	return &domain.CursorPage{Data: comments, NextCursor: nextCursor, HasMore: hasMore}, nil
}

type CreateCommentBody struct {
	Content string `json:"content"`
}

func (s *Services) CreateComment(ctx context.Context, user *middleware.UserClaims, postID string, req CreateCommentBody) (*domain.CommentResponse, error) {
	if req.Content == "" {
		return nil, domain.ErrValidation("Content is required", map[string]string{"content": "Content is required"})
	}

	commentID := uuid.New().String()

	_, err := s.DB.Exec(ctx,
		`INSERT INTO post_comments (id, post_id, author_id, content) VALUES ($1, $2, $3, $4)`,
		commentID, postID, user.UserID, req.Content,
	)
	if err != nil {
		return nil, domain.Err(500, "Failed to create comment")
	}

	return &domain.CommentResponse{
		ID:       commentID,
		PostID:   postID,
		AuthorID: user.UserID,
		Content:  req.Content,
	}, nil
}

func (s *Services) FeedHome(ctx context.Context, user *middleware.UserClaims, cursor string) (*domain.CursorPage, error) {
	limit := 20

	rows, err := s.DB.Query(ctx,
		`SELECT p.id, p.author_id, p.content, p.media_urls, p.visibility, p.like_count, p.created_at, p.updated_at,
		        u.id, u.username, u.role, u.avatar_url
		 FROM posts p
		 JOIN users u ON p.author_id = u.id
		 WHERE p.deleted_at IS NULL 
		 AND (p.visibility = 'public' OR p.author_id = $1 OR 
		      EXISTS(SELECT 1 FROM subscriptions s WHERE s.user_id = $1 AND s.status = 'active' AND s.creator_id = p.author_id))
		 AND ($2 = '' OR p.created_at < $2)
		 ORDER BY p.created_at DESC LIMIT $3`,
		user.UserID, cursor, limit+1,
	)
	if err != nil {
		return nil, domain.Err(500, "Database error")
	}
	defer rows.Close()

	var posts []any
	for rows.Next() {
		var p domain.PostResponse
		var author domain.UserPublic
		err := rows.Scan(&p.ID, &p.AuthorID, &p.Content, &p.MediaURLs, &p.Visibility, &p.LikeCount,
			&p.CreatedAt, &p.UpdatedAt, &author.ID, &author.Username, &author.Role, &author.AvatarURL)
		if err != nil {
			continue
		}
		p.Author = author
		posts = append(posts, p)
	}

	hasMore := len(posts) > limit
	if hasMore {
		posts = posts[:limit]
	}

	nextCursor := ""
	if hasMore && len(posts) > 0 {
		nextCursor = posts[len(posts)-1].(domain.PostResponse).CreatedAt
	}

	return &domain.CursorPage{Data: posts, NextCursor: nextCursor, HasMore: hasMore}, nil
}
