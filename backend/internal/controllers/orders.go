package controllers

import (
	"log/slog"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	httpmw "github.com/harem-brasil/backend/internal/middleware"
	"github.com/harem-brasil/backend/internal/services"
	"github.com/harem-brasil/backend/internal/utils"
)

type createCatalogOrderBody struct {
	ItemID string `json:"item_id" binding:"required"`
}

type billingPaidBody struct {
	PaymentRef string `json:"payment_ref" binding:"required,min=1,max=256"`
}

// InternalBillingSecretMiddleware valida X-Internal-Billing-Secret para callbacks da fila billing.
func InternalBillingSecretMiddleware(svc *services.Services, logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		sec := strings.TrimSpace(c.GetHeader("X-Internal-Billing-Secret"))
		if !svc.ValidateInternalBillingSecret(sec) {
			if logger != nil {
				logger.Warn("internal billing auth failed", "path", c.FullPath())
			}
			utils.RespondProblem(c, http.StatusUnauthorized, http.StatusText(http.StatusUnauthorized), "Invalid internal billing credentials")
			c.Abort()
			return
		}
		c.Next()
	}
}

func postMeCatalogOrder(svc *services.Services, logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req createCatalogOrderBody
		if err := c.ShouldBindJSON(&req); err != nil {
			utils.RespondProblem(c, http.StatusBadRequest, http.StatusText(http.StatusBadRequest), "Invalid JSON")
			return
		}
		if _, err := uuid.Parse(strings.TrimSpace(req.ItemID)); err != nil {
			utils.RespondProblem(c, http.StatusBadRequest, http.StatusText(http.StatusBadRequest), "Invalid item id")
			return
		}
		u := httpmw.MustUserClaims(c)
		o, err := svc.CreateCatalogOrder(c.Request.Context(), u, req.ItemID)
		if err != nil {
			utils.HandleServiceError(c, logger, err)
			return
		}
		utils.RespondJSON(c, http.StatusCreated, o)
	}
}

func getMeCatalogOrders(svc *services.Services, logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		cur := c.Query("cursor")
		if utf8.RuneCountInString(cur) > maxCursorQueryLen {
			utils.RespondProblem(c, http.StatusBadRequest, http.StatusText(http.StatusBadRequest), "cursor too long")
			return
		}
		if !validateCreatorPaginationCursor(cur) {
			utils.RespondProblem(c, http.StatusBadRequest, http.StatusText(http.StatusBadRequest), "invalid cursor")
			return
		}
		u := httpmw.MustUserClaims(c)
		page, err := svc.ListBuyerCatalogOrders(c.Request.Context(), u, cur)
		if err != nil {
			utils.HandleServiceError(c, logger, err)
			return
		}
		utils.RespondJSON(c, http.StatusOK, page)
	}
}

func getCatalogOrderByID(svc *services.Services, logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := strings.TrimSpace(c.Param("order_id"))
		if _, err := uuid.Parse(raw); err != nil {
			utils.RespondProblem(c, http.StatusBadRequest, http.StatusText(http.StatusBadRequest), "Invalid order id")
			return
		}
		u := httpmw.MustUserClaims(c)
		o, err := svc.GetCatalogOrder(c.Request.Context(), u, raw)
		if err != nil {
			utils.HandleServiceError(c, logger, err)
			return
		}
		utils.RespondJSON(c, http.StatusOK, o)
	}
}

func postMeCatalogOrderCheckout(svc *services.Services, logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := strings.TrimSpace(c.Param("order_id"))
		if _, err := uuid.Parse(raw); err != nil {
			utils.RespondProblem(c, http.StatusBadRequest, http.StatusText(http.StatusBadRequest), "Invalid order id")
			return
		}
		u := httpmw.MustUserClaims(c)
		o, err := svc.CheckoutCatalogOrder(c.Request.Context(), u, raw)
		if err != nil {
			utils.HandleServiceError(c, logger, err)
			return
		}
		utils.RespondJSON(c, http.StatusOK, o)
	}
}

func postMeCatalogOrderCancel(svc *services.Services, logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := strings.TrimSpace(c.Param("order_id"))
		if _, err := uuid.Parse(raw); err != nil {
			utils.RespondProblem(c, http.StatusBadRequest, http.StatusText(http.StatusBadRequest), "Invalid order id")
			return
		}
		u := httpmw.MustUserClaims(c)
		o, err := svc.CancelBuyerCatalogOrder(c.Request.Context(), u, raw)
		if err != nil {
			utils.HandleServiceError(c, logger, err)
			return
		}
		utils.RespondJSON(c, http.StatusOK, o)
	}
}

func postInternalBillingCatalogOrderPaid(svc *services.Services, logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := strings.TrimSpace(c.Param("order_id"))
		if _, err := uuid.Parse(raw); err != nil {
			utils.RespondProblem(c, http.StatusBadRequest, http.StatusText(http.StatusBadRequest), "Invalid order id")
			return
		}
		var req billingPaidBody
		if err := c.ShouldBindJSON(&req); err != nil {
			utils.RespondProblem(c, http.StatusBadRequest, http.StatusText(http.StatusBadRequest), "Invalid JSON")
			return
		}
		o, err := svc.ApplyBillingPaidCatalogOrder(c.Request.Context(), raw, req.PaymentRef)
		if err != nil {
			utils.HandleServiceError(c, logger, err)
			return
		}
		utils.RespondJSON(c, http.StatusOK, o)
	}
}

func postCreatorFulfillOrder(svc *services.Services, logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := strings.TrimSpace(c.Param("order_id"))
		if _, err := uuid.Parse(raw); err != nil {
			utils.RespondProblem(c, http.StatusBadRequest, http.StatusText(http.StatusBadRequest), "Invalid order id")
			return
		}
		u := httpmw.MustUserClaims(c)
		o, err := svc.FulfillCatalogOrder(c.Request.Context(), u, raw)
		if err != nil {
			utils.HandleServiceError(c, logger, err)
			return
		}
		utils.RespondJSON(c, http.StatusOK, o)
	}
}
