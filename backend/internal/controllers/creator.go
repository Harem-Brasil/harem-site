package controllers

import (
	"log/slog"
	"net/http"
	"unicode/utf8"

	"github.com/gin-gonic/gin"

	"github.com/harem-brasil/backend/internal/domain"
	httpmw "github.com/harem-brasil/backend/internal/middleware"
	"github.com/harem-brasil/backend/internal/services"
	"github.com/harem-brasil/backend/internal/utils"
)

const maxCursorQueryLen = 80

// CreatorRoutes monta /creator/* no grupo v1 (/api/v1/...). Papéis: creator, admin.
// Um rate limit global por IP já é aplicado em server.go; não duplicar aqui.
func CreatorRoutes(engine *gin.Engine, svc *services.Services, jwtSecret []byte, logger *slog.Logger) {
	v1 := engine.Group("/api/v1")

	creator := v1.Group("")
	creator.Use(httpmw.MaxBodySize(1 << 20))
	creator.Use(httpmw.GinAuth(jwtSecret, []string{"creator", "admin"}, logger))
	{
		creator.POST("/creator/apply", postCreatorApply(svc, logger))
		creator.GET("/creator/dashboard", getCreatorDashboard(svc, logger))
		creator.GET("/creator/earnings", getCreatorEarnings(svc, logger))
		creator.GET("/creator/catalog", getCreatorCatalog(svc, logger))
		creator.GET("/creator/orders", getCreatorOrders(svc, logger))
	}
}

func postCreatorApply(svc *services.Services, logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req domain.CreatorApplyRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			utils.RespondProblem(c, http.StatusBadRequest, http.StatusText(http.StatusBadRequest), "Invalid JSON")
			return
		}
		u := httpmw.MustUserClaims(c)
		resp, err := svc.PostCreatorApply(c.Request.Context(), u, req.Bio, req.SocialLinks)
		if err != nil {
			utils.HandleServiceError(c, logger, err)
			return
		}
		utils.RespondJSON(c, http.StatusCreated, resp)
	}
}

func getCreatorDashboard(svc *services.Services, logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		u := httpmw.MustUserClaims(c)
		d, err := svc.GetCreatorDashboard(c.Request.Context(), u)
		if err != nil {
			utils.HandleServiceError(c, logger, err)
			return
		}
		utils.RespondJSON(c, http.StatusOK, d)
	}
}

func getCreatorEarnings(svc *services.Services, logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		u := httpmw.MustUserClaims(c)
		m, err := svc.GetCreatorEarnings(c.Request.Context(), u)
		if err != nil {
			utils.HandleServiceError(c, logger, err)
			return
		}
		utils.RespondJSON(c, http.StatusOK, m)
	}
}

func getCreatorCatalog(svc *services.Services, logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		cur := c.Query("cursor")
		if utf8.RuneCountInString(cur) > maxCursorQueryLen {
			utils.RespondProblem(c, http.StatusBadRequest, http.StatusText(http.StatusBadRequest), "cursor too long")
			return
		}
		u := httpmw.MustUserClaims(c)
		page, err := svc.GetCreatorCatalog(c.Request.Context(), u, cur)
		if err != nil {
			utils.HandleServiceError(c, logger, err)
			return
		}
		utils.RespondJSON(c, http.StatusOK, page)
	}
}

func getCreatorOrders(svc *services.Services, logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		cur := c.Query("cursor")
		if utf8.RuneCountInString(cur) > maxCursorQueryLen {
			utils.RespondProblem(c, http.StatusBadRequest, http.StatusText(http.StatusBadRequest), "cursor too long")
			return
		}
		u := httpmw.MustUserClaims(c)
		page, err := svc.GetCreatorOrders(c.Request.Context(), u, cur)
		if err != nil {
			utils.HandleServiceError(c, logger, err)
			return
		}
		utils.RespondJSON(c, http.StatusOK, page)
	}
}
