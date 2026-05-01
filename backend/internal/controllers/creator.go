package controllers

import (
	"log/slog"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/harem-brasil/backend/internal/domain"
	httpmw "github.com/harem-brasil/backend/internal/middleware"
	"github.com/harem-brasil/backend/internal/services"
	"github.com/harem-brasil/backend/internal/utils"
)

const maxCursorQueryLen = 80

// validateCreatorPaginationCursor aceita instantes no formato da própria API (RFC3339 com Z ou offset)
// e também data/hora sem fuso (ex.: 2006-01-02T15:04:05), comum ao colar manualmente.
func validateCreatorPaginationCursor(cur string) bool {
	if cur == "" {
		return true
	}
	layouts := []string{
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02T15:04:05",
	}
	for _, layout := range layouts {
		if _, err := time.Parse(layout, cur); err == nil {
			return true
		}
	}
	return false
}

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
		creator.POST("/creator/catalog", postCreatorCatalogItem(svc, logger))
		creator.GET("/creator/catalog", getCreatorCatalog(svc, logger))
		creator.PATCH("/creator/catalog/:item_id", patchCreatorCatalogItem(svc, logger))
		creator.DELETE("/creator/catalog/:item_id", deleteCreatorCatalogItem(svc, logger))
		creator.GET("/creator/orders", getCreatorOrders(svc, logger))
		creator.POST("/creator/orders/:order_id/fulfill", postCreatorFulfillOrder(svc, logger))
		creator.PATCH("/creator/profile", patchCreatorProfile(svc, logger))
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
		if !validateCreatorPaginationCursor(cur) {
			utils.RespondProblem(c, http.StatusBadRequest, http.StatusText(http.StatusBadRequest), "invalid cursor (use empty for first page or next_cursor / RFC3339 datetime)")
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
		if !validateCreatorPaginationCursor(cur) {
			utils.RespondProblem(c, http.StatusBadRequest, http.StatusText(http.StatusBadRequest), "invalid cursor (use empty for first page or next_cursor / RFC3339 datetime)")
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

func postCreatorCatalogItem(svc *services.Services, logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req domain.CreatorCatalogCreateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			if logger != nil {
				logger.Warn("creator catalog create validation failed", "path", c.FullPath())
			}
			utils.RespondProblem(c, http.StatusBadRequest, http.StatusText(http.StatusBadRequest), "Invalid JSON")
			return
		}
		u := httpmw.MustUserClaims(c)
		item, err := svc.CreateCreatorCatalogItem(c.Request.Context(), u, req)
		if err != nil {
			utils.HandleServiceError(c, logger, err)
			return
		}
		utils.RespondJSON(c, http.StatusCreated, item)
	}
}

func patchCreatorCatalogItem(svc *services.Services, logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		rawID := strings.TrimSpace(c.Param("item_id"))
		if _, err := uuid.Parse(rawID); err != nil {
			utils.RespondProblem(c, http.StatusBadRequest, http.StatusText(http.StatusBadRequest), "Invalid item id")
			return
		}
		var req domain.CreatorCatalogPatchRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			if logger != nil {
				logger.Warn("creator catalog patch validation failed", "path", c.FullPath())
			}
			utils.RespondProblem(c, http.StatusBadRequest, http.StatusText(http.StatusBadRequest), "Invalid JSON")
			return
		}
		u := httpmw.MustUserClaims(c)
		item, err := svc.PatchCreatorCatalogItem(c.Request.Context(), u, rawID, req)
		if err != nil {
			utils.HandleServiceError(c, logger, err)
			return
		}
		utils.RespondJSON(c, http.StatusOK, item)
	}
}

func deleteCreatorCatalogItem(svc *services.Services, logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		rawID := strings.TrimSpace(c.Param("item_id"))
		if _, err := uuid.Parse(rawID); err != nil {
			utils.RespondProblem(c, http.StatusBadRequest, http.StatusText(http.StatusBadRequest), "Invalid item id")
			return
		}
		u := httpmw.MustUserClaims(c)
		if err := svc.DeleteCreatorCatalogItem(c.Request.Context(), u, rawID); err != nil {
			utils.HandleServiceError(c, logger, err)
			return
		}
		c.Status(http.StatusNoContent)
	}
}

func patchCreatorProfile(svc *services.Services, logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req domain.CreatorProfilePatchRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			if logger != nil {
				logger.Warn("creator profile patch validation failed", "path", c.FullPath())
			}
			utils.RespondProblem(c, http.StatusBadRequest, http.StatusText(http.StatusBadRequest), "Invalid JSON")
			return
		}
		bio := strings.TrimSpace(req.Bio)
		if bio == "" {
			utils.RespondProblem(c, http.StatusBadRequest, http.StatusText(http.StatusBadRequest), "bio cannot be empty")
			return
		}
		u := httpmw.MustUserClaims(c)
		if err := svc.PatchCreatorProfile(c.Request.Context(), u, bio); err != nil {
			utils.HandleServiceError(c, logger, err)
			return
		}
		utils.RespondJSON(c, http.StatusOK, gin.H{"bio": bio})
	}
}
