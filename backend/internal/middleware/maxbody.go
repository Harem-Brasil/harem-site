package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// MaxBodySize limita o tamanho do corpo (API4 — consumo de recursos).
func MaxBodySize(limit int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Body != nil {
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, limit)
		}
		c.Next()
	}
}
