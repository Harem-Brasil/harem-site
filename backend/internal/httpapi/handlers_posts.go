package httpapi

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	httpmw "github.com/harem-brasil/backend/internal/middleware"
)

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

func (s *Server) handleListPosts(w http.ResponseWriter, r *http.Request) {
	cursor := r.URL.Query().Get("cursor")
	limit := 20

	rows, err := s.db.Query(r.Context(),
		`SELECT p.id, p.author_id, p.content, p.media_urls, p.visibility, p.like_count, p.created_at, p.updated_at,
		        u.id, u.username, u.role, u.avatar_url
		 FROM posts p
		 JOIN users u ON p.author_id = u.id
		 WHERE p.deleted_at IS NULL AND p.visibility = 'public'
		 AND ($1 = '' OR p.id > $1)
		 ORDER BY p.created_at DESC LIMIT $2`,
		cursor, limit+1,
	)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Database error")
		return
	}
	defer rows.Close()

	var posts []any
	for rows.Next() {
		var p PostResponse
		var author UserPublic
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
		nextCursor = posts[len(posts)-1].(PostResponse).ID
	}

	respondJSON(w, CursorPage{
		Data:       posts,
		NextCursor: nextCursor,
		HasMore:    hasMore,
	})
}

func (s *Server) handleGetPost(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var p PostResponse
	var author UserPublic

	err := s.db.QueryRow(r.Context(),
		`SELECT p.id, p.author_id, p.content, p.media_urls, p.visibility, p.like_count, p.created_at, p.updated_at,
		        u.id, u.username, u.role, u.avatar_url
		 FROM posts p
		 JOIN users u ON p.author_id = u.id
		 WHERE p.id = $1 AND p.deleted_at IS NULL`,
		id,
	).Scan(&p.ID, &p.AuthorID, &p.Content, &p.MediaURLs, &p.Visibility, &p.LikeCount,
		&p.CreatedAt, &p.UpdatedAt, &author.ID, &author.Username, &author.Role, &author.AvatarURL)

	if err != nil {
		if err == pgx.ErrNoRows {
			respondError(w, http.StatusNotFound, "Post not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "Database error")
		return
	}

	p.Author = author
	respondJSON(w, p)
}

func (s *Server) handleCreatePost(w http.ResponseWriter, r *http.Request) {
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

	var req CreatePostRequest
	if err := json.Unmarshal(body, &req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	if req.Visibility == "" {
		req.Visibility = "public"
	}

	if req.Content == "" && len(req.MediaURLs) == 0 {
		respondValidationError(w, map[string]string{
			"content": "Content or media required",
		})
		return
	}

	postID := uuid.New().String()

	_, err = s.db.Exec(r.Context(),
		`INSERT INTO posts (id, author_id, content, media_urls, visibility) VALUES ($1, $2, $3, $4, $5)`,
		postID, user.UserID, req.Content, req.MediaURLs, req.Visibility,
	)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to create post")
		return
	}

	respondCreated(w, PostResponse{
		ID:         postID,
		AuthorID:   user.UserID,
		Content:    req.Content,
		MediaURLs:  req.MediaURLs,
		Visibility: req.Visibility,
	})
}

func (s *Server) handleUpdatePost(w http.ResponseWriter, r *http.Request) {
	user := httpmw.GetUser(r.Context())
	if user == nil {
		respondError(w, http.StatusUnauthorized, "Not authenticated")
		return
	}

	id := chi.URLParam(r, "id")

	var authorID string
	err := s.db.QueryRow(r.Context(), `SELECT author_id FROM posts WHERE id = $1 AND deleted_at IS NULL`, id).Scan(&authorID)
	if err != nil {
		if err == pgx.ErrNoRows {
			respondError(w, http.StatusNotFound, "Post not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "Database error")
		return
	}

	if authorID != user.UserID {
		respondError(w, http.StatusForbidden, "Not authorized to update this post")
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Failed to read request body")
		return
	}

	var updates map[string]any
	if err := json.Unmarshal(body, &updates); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	_, err = s.db.Exec(r.Context(),
		`UPDATE posts SET content = COALESCE($1, content), visibility = COALESCE($2, visibility), updated_at = NOW() WHERE id = $3`,
		updates["content"], updates["visibility"], id,
	)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to update post")
		return
	}

	respondNoContent(w)
}

func (s *Server) handleDeletePost(w http.ResponseWriter, r *http.Request) {
	user := httpmw.GetUser(r.Context())
	if user == nil {
		respondError(w, http.StatusUnauthorized, "Not authenticated")
		return
	}

	id := chi.URLParam(r, "id")

	var authorID string
	err := s.db.QueryRow(r.Context(), `SELECT author_id FROM posts WHERE id = $1 AND deleted_at IS NULL`, id).Scan(&authorID)
	if err != nil {
		if err == pgx.ErrNoRows {
			respondError(w, http.StatusNotFound, "Post not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "Database error")
		return
	}

	if authorID != user.UserID {
		respondError(w, http.StatusForbidden, "Not authorized to delete this post")
		return
	}

	_, err = s.db.Exec(r.Context(), `UPDATE posts SET deleted_at = NOW() WHERE id = $1`, id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to delete post")
		return
	}

	respondNoContent(w)
}

func (s *Server) handleLikePost(w http.ResponseWriter, r *http.Request) {
	user := httpmw.GetUser(r.Context())
	if user == nil {
		respondError(w, http.StatusUnauthorized, "Not authenticated")
		return
	}

	id := chi.URLParam(r, "id")

	_, err := s.db.Exec(r.Context(),
		`INSERT INTO post_likes (post_id, user_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
		id, user.UserID,
	)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to like post")
		return
	}

	respondNoContent(w)
}

func (s *Server) handleUnlikePost(w http.ResponseWriter, r *http.Request) {
	user := httpmw.GetUser(r.Context())
	if user == nil {
		respondError(w, http.StatusUnauthorized, "Not authenticated")
		return
	}

	id := chi.URLParam(r, "id")

	_, err := s.db.Exec(r.Context(),
		`DELETE FROM post_likes WHERE post_id = $1 AND user_id = $2`,
		id, user.UserID,
	)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to unlike post")
		return
	}

	respondNoContent(w)
}

type CommentResponse struct {
	ID        string     `json:"id"`
	PostID    string     `json:"post_id"`
	AuthorID  string     `json:"author_id"`
	Content   string     `json:"content"`
	CreatedAt string     `json:"created_at"`
	Author    UserPublic `json:"author,omitempty"`
}

func (s *Server) handleListComments(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	cursor := r.URL.Query().Get("cursor")
	limit := 20

	rows, err := s.db.Query(r.Context(),
		`SELECT c.id, c.post_id, c.author_id, c.content, c.created_at,
		        u.id, u.username, u.role, u.avatar_url
		 FROM post_comments c
		 JOIN users u ON c.author_id = u.id
		 WHERE c.post_id = $1 AND c.deleted_at IS NULL
		 AND ($2 = '' OR c.id > $2)
		 ORDER BY c.created_at DESC LIMIT $3`,
		id, cursor, limit+1,
	)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Database error")
		return
	}
	defer rows.Close()

	var comments []any
	for rows.Next() {
		var c CommentResponse
		var author UserPublic
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
		nextCursor = comments[len(comments)-1].(CommentResponse).ID
	}

	respondJSON(w, CursorPage{
		Data:       comments,
		NextCursor: nextCursor,
		HasMore:    hasMore,
	})
}

func (s *Server) handleCreateComment(w http.ResponseWriter, r *http.Request) {
	user := httpmw.GetUser(r.Context())
	if user == nil {
		respondError(w, http.StatusUnauthorized, "Not authenticated")
		return
	}

	postID := chi.URLParam(r, "id")

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

	commentID := uuid.New().String()

	_, err = s.db.Exec(r.Context(),
		`INSERT INTO post_comments (id, post_id, author_id, content) VALUES ($1, $2, $3, $4)`,
		commentID, postID, user.UserID, req.Content,
	)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to create comment")
		return
	}

	respondCreated(w, CommentResponse{
		ID:       commentID,
		PostID:   postID,
		AuthorID: user.UserID,
		Content:  req.Content,
	})
}

func (s *Server) handleFeedHome(w http.ResponseWriter, r *http.Request) {
	user := httpmw.GetUser(r.Context())
	if user == nil {
		respondError(w, http.StatusUnauthorized, "Not authenticated")
		return
	}

	cursor := r.URL.Query().Get("cursor")
	limit := 20

	rows, err := s.db.Query(r.Context(),
		`SELECT p.id, p.author_id, p.content, p.media_urls, p.visibility, p.like_count, p.created_at, p.updated_at,
		        u.id, u.username, u.role, u.avatar_url
		 FROM posts p
		 JOIN users u ON p.author_id = u.id
		 WHERE p.deleted_at IS NULL 
		 AND (p.visibility = 'public' OR p.author_id = $1 OR 
		      EXISTS(SELECT 1 FROM subscriptions s WHERE s.user_id = $1 AND s.status = 'active' AND s.creator_id = p.author_id))
		 AND ($2 = '' OR p.id > $2)
		 ORDER BY p.created_at DESC LIMIT $3`,
		user.UserID, cursor, limit+1,
	)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Database error")
		return
	}
	defer rows.Close()

	var posts []any
	for rows.Next() {
		var p PostResponse
		var author UserPublic
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
		nextCursor = posts[len(posts)-1].(PostResponse).ID
	}

	respondJSON(w, CursorPage{
		Data:       posts,
		NextCursor: nextCursor,
		HasMore:    hasMore,
	})
}
