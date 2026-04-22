package controllers

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/harem-brasil/backend/internal/domain"
	httpmw "github.com/harem-brasil/backend/internal/middleware"
	"github.com/harem-brasil/backend/internal/services"
	"github.com/harem-brasil/backend/internal/utils"
)

// RegisterRoutes define todos os endpoints REST sobre Gin (routing + binding + controllers finos).
func RegisterRoutes(engine *gin.Engine, svc *services.Services, jwtSecret []byte, logger *slog.Logger) {
	engine.GET("/health", func(c *gin.Context) {
		status := svc.Health(c.Request.Context())
		code := http.StatusOK
		if status["status"] == "degraded" {
			code = http.StatusServiceUnavailable
		}
		utils.RespondJSON(c, code, status)
	})

	engine.GET("/healthz", func(c *gin.Context) {
		ok, checks, ver := svc.Healthz(c.Request.Context())
		code := http.StatusOK
		if !ok {
			code = http.StatusServiceUnavailable
		}
		utils.RespondJSON(c, code, gin.H{
			"status":  map[bool]string{true: "healthy", false: "unhealthy"}[ok],
			"checks":  checks,
			"version": ver,
		})
	})

	engine.GET("/readyz", func(c *gin.Context) {
		ready, checks := svc.Readyz(c.Request.Context())
		code := http.StatusOK
		if !ready {
			code = http.StatusServiceUnavailable
		}
		utils.RespondJSON(c, code, gin.H{
			"status": map[bool]string{true: "ready", false: "not_ready"}[ready],
			"checks": checks,
		})
	})

	engine.GET("/version", func(c *gin.Context) {
		utils.RespondJSON(c, http.StatusOK, svc.Version())
	})

	v1 := engine.Group("/api/v1")

	authPublic := v1.Group("")
	authPublic.Use(httpmw.MaxBodySize(1 << 20))
	{
		authPublic.POST("/auth/register", func(c *gin.Context) {
			var req domain.RegisterRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				utils.RespondProblem(c, http.StatusBadRequest, http.StatusText(http.StatusBadRequest), "Invalid JSON")
				return
			}
			resp, err := svc.Register(c.Request.Context(), req)
			if err != nil {
				utils.HandleServiceError(c, logger, err)
				return
			}
			utils.RespondJSON(c, http.StatusCreated, resp)
		})
		authPublic.POST("/auth/login", func(c *gin.Context) {
			var req domain.LoginRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				utils.RespondProblem(c, http.StatusBadRequest, http.StatusText(http.StatusBadRequest), "Invalid JSON")
				return
			}
			resp, err := svc.Login(c.Request.Context(), req)
			if err != nil {
				utils.HandleServiceError(c, logger, err)
				return
			}
			utils.RespondJSON(c, http.StatusOK, resp)
		})
		authPublic.POST("/auth/refresh", func(c *gin.Context) {
			var req services.RefreshBody
			if err := c.ShouldBindJSON(&req); err != nil {
				utils.RespondProblem(c, http.StatusBadRequest, http.StatusText(http.StatusBadRequest), "Invalid JSON")
				return
			}
			resp, err := svc.Refresh(c.Request.Context(), req)
			if err != nil {
				utils.HandleServiceError(c, logger, err)
				return
			}
			utils.RespondJSON(c, http.StatusOK, resp)
		})
		authPublic.POST("/auth/logout", func(c *gin.Context) {
			var req services.LogoutBody
			_ = c.ShouldBindJSON(&req)
			if err := svc.Logout(c.Request.Context(), req); err != nil {
				utils.HandleServiceError(c, logger, err)
				return
			}
			c.Status(http.StatusNoContent)
		})
		authPublic.POST("/auth/logout-all", func(c *gin.Context) {
			if err := svc.LogoutAll(c.Request.Context()); err != nil {
				utils.HandleServiceError(c, logger, err)
				return
			}
			c.Status(http.StatusNoContent)
		})
		authPublic.GET("/auth/oauth/:provider/authorize", func(c *gin.Context) {
			if err := svc.OAuthAuthorize(c.Request.Context(), c.Param("provider")); err != nil {
				utils.HandleServiceError(c, logger, err)
				return
			}
		})
		authPublic.GET("/auth/oauth/:provider/callback", func(c *gin.Context) {
			if err := svc.OAuthCallback(c.Request.Context(), c.Param("provider")); err != nil {
				utils.HandleServiceError(c, logger, err)
				return
			}
		})
		authPublic.POST("/auth/email/verify", func(c *gin.Context) {
			if err := svc.EmailVerify(c.Request.Context()); err != nil {
				utils.HandleServiceError(c, logger, err)
				return
			}
		})
		authPublic.POST("/auth/password/forgot", func(c *gin.Context) {
			if err := svc.PasswordForgot(c.Request.Context()); err != nil {
				utils.HandleServiceError(c, logger, err)
				return
			}
		})
		authPublic.POST("/auth/password/reset", func(c *gin.Context) {
			if err := svc.PasswordReset(c.Request.Context()); err != nil {
				utils.HandleServiceError(c, logger, err)
				return
			}
		})
	}

	me := v1.Group("")
	me.Use(httpmw.MaxBodySize(1 << 20))
	me.Use(httpmw.GinAuth(jwtSecret, []string{"user", "creator", "moderator", "admin"}, logger))
	{
		me.GET("/me", func(c *gin.Context) {
			u := httpmw.MustUserClaims(c)
			resp, err := svc.GetMe(c.Request.Context(), u)
			if err != nil {
				utils.HandleServiceError(c, logger, err)
				return
			}
			utils.RespondJSON(c, http.StatusOK, resp)
		})
		me.PATCH("/me", func(c *gin.Context) {
			var updates map[string]any
			if err := c.ShouldBindJSON(&updates); err != nil {
				utils.RespondProblem(c, http.StatusBadRequest, http.StatusText(http.StatusBadRequest), "Invalid JSON")
				return
			}
			u := httpmw.MustUserClaims(c)
			if err := svc.UpdateMe(c.Request.Context(), u, updates); err != nil {
				utils.HandleServiceError(c, logger, err)
				return
			}
			c.Status(http.StatusNoContent)
		})
		me.DELETE("/me", func(c *gin.Context) {
			u := httpmw.MustUserClaims(c)
			if err := svc.DeleteMe(c.Request.Context(), u); err != nil {
				utils.HandleServiceError(c, logger, err)
				return
			}
			c.Status(http.StatusNoContent)
		})
	}

	u10 := v1.Group("")
	u10.Use(httpmw.MaxBodySize(10 << 20))
	u10.Use(httpmw.GinAuth(jwtSecret, []string{"user", "creator", "moderator", "admin"}, logger))
	{
		u10.GET("/users/:id", func(c *gin.Context) {
			resp, err := svc.GetUserByID(c.Request.Context(), c.Param("id"))
			if err != nil {
				utils.HandleServiceError(c, logger, err)
				return
			}
			utils.RespondJSON(c, http.StatusOK, resp)
		})
		u10.GET("/users/:id/posts", func(c *gin.Context) {
			page, err := svc.GetUserPosts(c.Request.Context(), c.Param("id"), c.Query("cursor"))
			if err != nil {
				utils.HandleServiceError(c, logger, err)
				return
			}
			utils.RespondJSON(c, http.StatusOK, page)
		})
		u10.GET("/users", func(c *gin.Context) {
			page, err := svc.ListUsers(c.Request.Context(), c.Query("cursor"))
			if err != nil {
				utils.HandleServiceError(c, logger, err)
				return
			}
			utils.RespondJSON(c, http.StatusOK, page)
		})
		u10.GET("/users/search", func(c *gin.Context) {
			page, err := svc.SearchUsers(c.Request.Context(), c.Query("q"), c.Query("cursor"))
			if err != nil {
				utils.HandleServiceError(c, logger, err)
				return
			}
			utils.RespondJSON(c, http.StatusOK, page)
		})
	}

	posts := v1.Group("")
	posts.Use(httpmw.MaxBodySize(10 << 20))
	posts.Use(httpmw.GinAuth(jwtSecret, []string{"user", "creator", "moderator", "admin"}, logger))
	{
		posts.GET("/posts", func(c *gin.Context) {
			page, err := svc.ListPosts(c.Request.Context(), c.Query("cursor"))
			if err != nil {
				utils.HandleServiceError(c, logger, err)
				return
			}
			utils.RespondJSON(c, http.StatusOK, page)
		})
		posts.GET("/posts/:id", func(c *gin.Context) {
			p, err := svc.GetPost(c.Request.Context(), c.Param("id"))
			if err != nil {
				utils.HandleServiceError(c, logger, err)
				return
			}
			utils.RespondJSON(c, http.StatusOK, p)
		})
		posts.POST("/posts", func(c *gin.Context) {
			var req domain.CreatePostRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				utils.RespondProblem(c, http.StatusBadRequest, http.StatusText(http.StatusBadRequest), "Invalid JSON")
				return
			}
			u := httpmw.MustUserClaims(c)
			p, err := svc.CreatePost(c.Request.Context(), u, req)
			if err != nil {
				utils.HandleServiceError(c, logger, err)
				return
			}
			utils.RespondJSON(c, http.StatusCreated, p)
		})
		posts.PATCH("/posts/:id", func(c *gin.Context) {
			var updates map[string]any
			if err := c.ShouldBindJSON(&updates); err != nil {
				utils.RespondProblem(c, http.StatusBadRequest, http.StatusText(http.StatusBadRequest), "Invalid JSON")
				return
			}
			u := httpmw.MustUserClaims(c)
			if err := svc.UpdatePost(c.Request.Context(), u, c.Param("id"), updates); err != nil {
				utils.HandleServiceError(c, logger, err)
				return
			}
			c.Status(http.StatusNoContent)
		})
		posts.DELETE("/posts/:id", func(c *gin.Context) {
			u := httpmw.MustUserClaims(c)
			if err := svc.DeletePost(c.Request.Context(), u, c.Param("id")); err != nil {
				utils.HandleServiceError(c, logger, err)
				return
			}
			c.Status(http.StatusNoContent)
		})
		posts.POST("/posts/:id/like", func(c *gin.Context) {
			u := httpmw.MustUserClaims(c)
			if err := svc.LikePost(c.Request.Context(), u, c.Param("id")); err != nil {
				utils.HandleServiceError(c, logger, err)
				return
			}
			c.Status(http.StatusNoContent)
		})
		posts.DELETE("/posts/:id/like", func(c *gin.Context) {
			u := httpmw.MustUserClaims(c)
			if err := svc.UnlikePost(c.Request.Context(), u, c.Param("id")); err != nil {
				utils.HandleServiceError(c, logger, err)
				return
			}
			c.Status(http.StatusNoContent)
		})
		posts.GET("/posts/:id/comments", func(c *gin.Context) {
			page, err := svc.ListComments(c.Request.Context(), c.Param("id"), c.Query("cursor"))
			if err != nil {
				utils.HandleServiceError(c, logger, err)
				return
			}
			utils.RespondJSON(c, http.StatusOK, page)
		})
		posts.POST("/posts/:id/comments", func(c *gin.Context) {
			var req services.CreateCommentBody
			if err := c.ShouldBindJSON(&req); err != nil {
				utils.RespondProblem(c, http.StatusBadRequest, http.StatusText(http.StatusBadRequest), "Invalid JSON")
				return
			}
			u := httpmw.MustUserClaims(c)
			resp, err := svc.CreateComment(c.Request.Context(), u, c.Param("id"), req)
			if err != nil {
				utils.HandleServiceError(c, logger, err)
				return
			}
			utils.RespondJSON(c, http.StatusCreated, resp)
		})
		posts.GET("/feed/home", func(c *gin.Context) {
			u := httpmw.MustUserClaims(c)
			page, err := svc.FeedHome(c.Request.Context(), u, c.Query("cursor"))
			if err != nil {
				utils.HandleServiceError(c, logger, err)
				return
			}
			utils.RespondJSON(c, http.StatusOK, page)
		})
	}

	forum := v1.Group("")
	forum.Use(httpmw.MaxBodySize(1 << 20))
	forum.Use(httpmw.GinAuth(jwtSecret, []string{"user", "creator", "moderator", "admin"}, logger))
	{
		forum.GET("/forum/categories", func(c *gin.Context) {
			list, err := svc.ListForumCategories(c.Request.Context())
			if err != nil {
				utils.HandleServiceError(c, logger, err)
				return
			}
			utils.RespondJSON(c, http.StatusOK, list)
		})
		forum.GET("/forum/topics", func(c *gin.Context) {
			page, err := svc.ListForumTopics(c.Request.Context(), c.Query("category_id"), c.Query("cursor"))
			if err != nil {
				utils.HandleServiceError(c, logger, err)
				return
			}
			utils.RespondJSON(c, http.StatusOK, page)
		})
		forum.POST("/forum/topics", func(c *gin.Context) {
			var req services.CreateForumTopicBody
			if err := c.ShouldBindJSON(&req); err != nil {
				utils.RespondProblem(c, http.StatusBadRequest, http.StatusText(http.StatusBadRequest), "Invalid JSON")
				return
			}
			u := httpmw.MustUserClaims(c)
			topic, err := svc.CreateForumTopic(c.Request.Context(), u, req)
			if err != nil {
				utils.HandleServiceError(c, logger, err)
				return
			}
			utils.RespondJSON(c, http.StatusCreated, topic)
		})
		forum.GET("/forum/topics/:id", func(c *gin.Context) {
			topic, err := svc.GetForumTopic(c.Request.Context(), c.Param("id"))
			if err != nil {
				utils.HandleServiceError(c, logger, err)
				return
			}
			utils.RespondJSON(c, http.StatusOK, topic)
		})
		forum.POST("/forum/topics/:id/posts", func(c *gin.Context) {
			var req services.ForumPostBody
			if err := c.ShouldBindJSON(&req); err != nil {
				utils.RespondProblem(c, http.StatusBadRequest, http.StatusText(http.StatusBadRequest), "Invalid JSON")
				return
			}
			u := httpmw.MustUserClaims(c)
			post, err := svc.CreateForumPost(c.Request.Context(), u, c.Param("id"), req)
			if err != nil {
				utils.HandleServiceError(c, logger, err)
				return
			}
			utils.RespondJSON(c, http.StatusCreated, post)
		})
	}

	chat := v1.Group("")
	chat.Use(httpmw.MaxBodySize(1 << 20))
	chat.Use(httpmw.GinAuth(jwtSecret, []string{"user", "creator", "moderator", "admin"}, logger))
	{
		chat.GET("/chat/rooms", func(c *gin.Context) {
			u := httpmw.MustUserClaims(c)
			list, err := svc.ListChatRooms(c.Request.Context(), u)
			if err != nil {
				utils.HandleServiceError(c, logger, err)
				return
			}
			utils.RespondJSON(c, http.StatusOK, list)
		})
		chat.POST("/chat/rooms", func(c *gin.Context) {
			var req services.CreateChatRoomBody
			if err := c.ShouldBindJSON(&req); err != nil {
				utils.RespondProblem(c, http.StatusBadRequest, http.StatusText(http.StatusBadRequest), "Invalid JSON")
				return
			}
			u := httpmw.MustUserClaims(c)
			room, err := svc.CreateChatRoom(c.Request.Context(), u, req)
			if err != nil {
				utils.HandleServiceError(c, logger, err)
				return
			}
			utils.RespondJSON(c, http.StatusCreated, room)
		})
		chat.GET("/chat/rooms/:id", func(c *gin.Context) {
			u := httpmw.MustUserClaims(c)
			room, err := svc.GetChatRoom(c.Request.Context(), u, c.Param("id"))
			if err != nil {
				utils.HandleServiceError(c, logger, err)
				return
			}
			utils.RespondJSON(c, http.StatusOK, room)
		})
		chat.POST("/chat/rooms/:id/join", func(c *gin.Context) {
			u := httpmw.MustUserClaims(c)
			if err := svc.JoinChatRoom(c.Request.Context(), u, c.Param("id")); err != nil {
				utils.HandleServiceError(c, logger, err)
				return
			}
			c.Status(http.StatusNoContent)
		})
		chat.GET("/chat/rooms/:id/messages", func(c *gin.Context) {
			page, err := svc.ListChatMessages(c.Request.Context(), c.Param("id"), c.Query("cursor"))
			if err != nil {
				utils.HandleServiceError(c, logger, err)
				return
			}
			utils.RespondJSON(c, http.StatusOK, page)
		})
	}

	notif := v1.Group("")
	notif.Use(httpmw.MaxBodySize(1 << 20))
	notif.Use(httpmw.GinAuth(jwtSecret, []string{"user", "creator", "moderator", "admin"}, logger))
	{
		notif.GET("/notifications", func(c *gin.Context) {
			u := httpmw.MustUserClaims(c)
			page, err := svc.ListNotifications(c.Request.Context(), u, c.Query("cursor"), c.Query("unread") == "true")
			if err != nil {
				utils.HandleServiceError(c, logger, err)
				return
			}
			utils.RespondJSON(c, http.StatusOK, page)
		})
		notif.PATCH("/notifications/:id/read", func(c *gin.Context) {
			u := httpmw.MustUserClaims(c)
			if err := svc.MarkNotificationRead(c.Request.Context(), u, c.Param("id")); err != nil {
				utils.HandleServiceError(c, logger, err)
				return
			}
			c.Status(http.StatusNoContent)
		})
		notif.GET("/notifications/unread-count", func(c *gin.Context) {
			u := httpmw.MustUserClaims(c)
			n, err := svc.UnreadNotificationCount(c.Request.Context(), u)
			if err != nil {
				utils.HandleServiceError(c, logger, err)
				return
			}
			utils.RespondJSON(c, http.StatusOK, gin.H{"unread_count": n})
		})
	}

	creator := v1.Group("")
	creator.Use(httpmw.MaxBodySize(1 << 20))
	creator.Use(httpmw.GinAuth(jwtSecret, []string{"creator", "admin"}, logger))
	{
		creator.POST("/creator/apply", func(c *gin.Context) {
			var req struct {
				Bio         string   `json:"bio"`
				SocialLinks []string `json:"social_links"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				utils.RespondProblem(c, http.StatusBadRequest, http.StatusText(http.StatusBadRequest), "Invalid JSON")
				return
			}
			u := httpmw.MustUserClaims(c)
			app, err := svc.CreatorApply(c.Request.Context(), u, req.Bio, req.SocialLinks)
			if err != nil {
				utils.HandleServiceError(c, logger, err)
				return
			}
			utils.RespondJSON(c, http.StatusCreated, app)
		})
		creator.GET("/creator/dashboard", func(c *gin.Context) {
			u := httpmw.MustUserClaims(c)
			d, err := svc.CreatorDashboard(c.Request.Context(), u)
			if err != nil {
				utils.HandleServiceError(c, logger, err)
				return
			}
			utils.RespondJSON(c, http.StatusOK, d)
		})
		creator.GET("/creator/earnings", func(c *gin.Context) {
			u := httpmw.MustUserClaims(c)
			data, err := svc.CreatorEarnings(c.Request.Context(), u)
			if err != nil {
				utils.HandleServiceError(c, logger, err)
				return
			}
			utils.RespondJSON(c, http.StatusOK, data)
		})
		creator.GET("/creator/catalog", func(c *gin.Context) {
			u := httpmw.MustUserClaims(c)
			page, err := svc.CreatorCatalog(c.Request.Context(), u, c.Query("cursor"))
			if err != nil {
				utils.HandleServiceError(c, logger, err)
				return
			}
			utils.RespondJSON(c, http.StatusOK, page)
		})
		creator.GET("/creator/orders", func(c *gin.Context) {
			u := httpmw.MustUserClaims(c)
			page, err := svc.CreatorOrders(c.Request.Context(), u, c.Query("cursor"))
			if err != nil {
				utils.HandleServiceError(c, logger, err)
				return
			}
			utils.RespondJSON(c, http.StatusOK, page)
		})
	}

	bill := v1.Group("")
	bill.Use(httpmw.MaxBodySize(1 << 20))
	bill.Use(httpmw.GinAuth(jwtSecret, []string{"user", "creator", "moderator", "admin"}, logger))
	{
		bill.GET("/billing/plans", func(c *gin.Context) {
			plans, err := svc.ListPlans(c.Request.Context())
			if err != nil {
				utils.HandleServiceError(c, logger, err)
				return
			}
			utils.RespondJSON(c, http.StatusOK, plans)
		})
		bill.POST("/billing/checkout", func(c *gin.Context) {
			var req services.BillingCheckoutBody
			if err := c.ShouldBindJSON(&req); err != nil {
				utils.RespondProblem(c, http.StatusBadRequest, http.StatusText(http.StatusBadRequest), "Invalid JSON")
				return
			}
			u := httpmw.MustUserClaims(c)
			resp, err := svc.BillingCheckout(c.Request.Context(), u, req)
			if err != nil {
				utils.HandleServiceError(c, logger, err)
				return
			}
			utils.RespondJSON(c, http.StatusCreated, resp)
		})
		bill.GET("/billing/subscription", func(c *gin.Context) {
			u := httpmw.MustUserClaims(c)
			sub, err := svc.GetMySubscription(c.Request.Context(), u)
			if err != nil {
				utils.HandleServiceError(c, logger, err)
				return
			}
			utils.RespondJSON(c, http.StatusOK, sub)
		})
		bill.POST("/billing/subscription/cancel", func(c *gin.Context) {
			u := httpmw.MustUserClaims(c)
			if err := svc.CancelSubscription(c.Request.Context(), u); err != nil {
				utils.HandleServiceError(c, logger, err)
				return
			}
			c.Status(http.StatusNoContent)
		})
		bill.POST("/billing/subscription/resume", func(c *gin.Context) {
			u := httpmw.MustUserClaims(c)
			if err := svc.ResumeSubscription(c.Request.Context(), u, c.Param("id")); err != nil {
				utils.HandleServiceError(c, logger, err)
				return
			}
			c.Status(http.StatusNoContent)
		})
		bill.POST("/subscriptions", func(c *gin.Context) {
			var req services.CreateSubscriptionBody
			if err := c.ShouldBindJSON(&req); err != nil {
				utils.RespondProblem(c, http.StatusBadRequest, http.StatusText(http.StatusBadRequest), "Invalid JSON")
				return
			}
			u := httpmw.MustUserClaims(c)
			sub, err := svc.CreateSubscription(c.Request.Context(), u, req)
			if err != nil {
				utils.HandleServiceError(c, logger, err)
				return
			}
			utils.RespondJSON(c, http.StatusCreated, sub)
		})
		bill.GET("/subscriptions/me", func(c *gin.Context) {
			u := httpmw.MustUserClaims(c)
			sub, err := svc.GetMySubscription(c.Request.Context(), u)
			if err != nil {
				utils.HandleServiceError(c, logger, err)
				return
			}
			utils.RespondJSON(c, http.StatusOK, sub)
		})
	}

	media := v1.Group("")
	media.Use(httpmw.MaxBodySize(10 << 20))
	media.Use(httpmw.GinAuth(jwtSecret, []string{"user", "creator", "moderator", "admin"}, logger))
	{
		media.POST("/media/upload-sessions", func(c *gin.Context) {
			var req services.CreateUploadSessionBody
			if err := c.ShouldBindJSON(&req); err != nil {
				utils.RespondProblem(c, http.StatusBadRequest, http.StatusText(http.StatusBadRequest), "Invalid JSON")
				return
			}
			u := httpmw.MustUserClaims(c)
			sess, err := svc.CreateUploadSession(c.Request.Context(), u, req)
			if err != nil {
				utils.HandleServiceError(c, logger, err)
				return
			}
			utils.RespondJSON(c, http.StatusCreated, sess)
		})
		media.POST("/media/upload-sessions/:id/complete", func(c *gin.Context) {
			var req services.CompleteUploadBody
			if err := c.ShouldBindJSON(&req); err != nil {
				utils.RespondProblem(c, http.StatusBadRequest, http.StatusText(http.StatusBadRequest), "Invalid JSON")
				return
			}
			u := httpmw.MustUserClaims(c)
			sess, err := svc.CompleteUpload(c.Request.Context(), u, c.Param("id"), req)
			if err != nil {
				utils.HandleServiceError(c, logger, err)
				return
			}
			utils.RespondJSON(c, http.StatusOK, sess)
		})
	}

	admin := v1.Group("")
	admin.Use(httpmw.MaxBodySize(1 << 20))
	admin.Use(httpmw.GinAuth(jwtSecret, []string{"admin"}, logger))
	{
		admin.GET("/admin/users", func(c *gin.Context) {
			page, err := svc.AdminListUsers(c.Request.Context(), c.Query("cursor"))
			if err != nil {
				utils.HandleServiceError(c, logger, err)
				return
			}
			utils.RespondJSON(c, http.StatusOK, page)
		})
		admin.PATCH("/admin/users/:id/role", func(c *gin.Context) {
			var req services.AdminUpdateRoleBody
			if err := c.ShouldBindJSON(&req); err != nil {
				utils.RespondProblem(c, http.StatusBadRequest, http.StatusText(http.StatusBadRequest), "Invalid JSON")
				return
			}
			if err := svc.AdminUpdateRole(c.Request.Context(), c.Param("id"), req); err != nil {
				utils.HandleServiceError(c, logger, err)
				return
			}
			c.Status(http.StatusNoContent)
		})
		admin.DELETE("/admin/users/:id", func(c *gin.Context) {
			if err := svc.AdminDeleteUser(c.Request.Context(), c.Param("id")); err != nil {
				utils.HandleServiceError(c, logger, err)
				return
			}
			c.Status(http.StatusNoContent)
		})
		admin.GET("/admin/stats", func(c *gin.Context) {
			stats, err := svc.AdminStats(c.Request.Context())
			if err != nil {
				utils.HandleServiceError(c, logger, err)
				return
			}
			utils.RespondJSON(c, http.StatusOK, stats)
		})
		admin.GET("/admin/audit-log", func(c *gin.Context) {
			page, err := svc.AdminAuditLog(c.Request.Context(), c.Query("cursor"))
			if err != nil {
				utils.HandleServiceError(c, logger, err)
				return
			}
			utils.RespondJSON(c, http.StatusOK, page)
		})
	}

	wh := v1.Group("")
	wh.Use(httpmw.MaxBodySize(1 << 20))
	{
		wh.POST("/webhooks/stripe", func(c *gin.Context) {
			body, err := c.GetRawData()
			if err != nil {
				utils.RespondProblem(c, http.StatusBadRequest, http.StatusText(http.StatusBadRequest), "Failed to read body")
				return
			}
			if err := svc.WebhookStripe(c.Request.Context(), body, c.GetHeader("X-Signature"), c.GetHeader("Stripe-Signature")); err != nil {
				utils.HandleServiceError(c, logger, err)
				return
			}
			utils.RespondJSON(c, http.StatusOK, gin.H{"status": "received"})
		})
		wh.POST("/webhooks/pagseguro", func(c *gin.Context) {
			body, err := c.GetRawData()
			if err != nil {
				utils.RespondProblem(c, http.StatusBadRequest, http.StatusText(http.StatusBadRequest), "Failed to read body")
				return
			}
			if err := svc.WebhookPagSeguro(c.Request.Context(), body, c.GetHeader("X-Signature")); err != nil {
				utils.HandleServiceError(c, logger, err)
				return
			}
			utils.RespondJSON(c, http.StatusOK, gin.H{"status": "received"})
		})
		wh.POST("/webhooks/mercadopago", func(c *gin.Context) {
			body, err := c.GetRawData()
			if err != nil {
				utils.RespondProblem(c, http.StatusBadRequest, http.StatusText(http.StatusBadRequest), "Failed to read body")
				return
			}
			if err := svc.WebhookMercadoPago(c.Request.Context(), body, c.GetHeader("X-Signature")); err != nil {
				utils.HandleServiceError(c, logger, err)
				return
			}
			utils.RespondJSON(c, http.StatusOK, gin.H{"status": "received"})
		})
		wh.POST("/webhooks/:provider", func(c *gin.Context) {
			body, err := c.GetRawData()
			if err != nil {
				utils.RespondProblem(c, http.StatusBadRequest, http.StatusText(http.StatusBadRequest), "Failed to read body")
				return
			}
			resp, err := svc.WebhookGeneric(c.Request.Context(), c.Param("provider"), body, c.GetHeader("X-Signature"))
			if err != nil {
				utils.HandleServiceError(c, logger, err)
				return
			}
			utils.RespondJSON(c, http.StatusOK, resp)
		})
	}
}
