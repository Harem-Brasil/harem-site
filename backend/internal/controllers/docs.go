package controllers

import (
	"log/slog"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"

	"github.com/harem-brasil/backend/internal/docs"
	"github.com/harem-brasil/backend/internal/utils"
)

const swaggerUIPage = `<!DOCTYPE html>
<html lang="pt">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Harém Brasil API — OpenAPI</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5.11.2/swagger-ui.css" crossorigin="anonymous">
</head>
<body>
<div id="swagger-ui"></div>
<script src="https://unpkg.com/swagger-ui-dist@5.11.2/swagger-ui-bundle.js" crossorigin="anonymous"></script>
<script>
  window.onload = () => {
    SwaggerUIBundle({
      url: window.location.origin + "/openapi.yaml",
      dom_id: '#swagger-ui',
    });
  };
</script>
</body>
</html>
`

// RegisterDocsRoutes expõe GET /openapi.yaml e GET /docs (Swagger UI).
// Deve registar-se antes do middleware que força Content-Type JSON em todas as rotas.
func RegisterDocsRoutes(engine *gin.Engine, logger *slog.Logger) {
	engine.GET("/openapi.yaml", serveOpenAPISpec(logger))
	engine.GET("/docs", redirectDocsSlash)
	engine.GET("/docs/", serveSwaggerUI())
}

func serveOpenAPISpec(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		path, err := docs.ResolveOpenAPIYAMLPath()
		if err != nil {
			if logger != nil {
				logger.Warn("openapi spec unavailable", "error", err.Error())
			}
			utils.RespondProblem(c, http.StatusServiceUnavailable, http.StatusText(http.StatusServiceUnavailable), err.Error())
			return
		}
		data, err := os.ReadFile(path)
		if err != nil {
			if logger != nil {
				logger.Warn("openapi read failed", "path", path, "error", err.Error())
			}
			utils.RespondProblem(c, http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError), "Failed to read OpenAPI spec")
			return
		}
		c.Header("Content-Type", "application/yaml; charset=utf-8")
		c.Header("Cache-Control", "public, max-age=120")
		c.Data(http.StatusOK, "application/yaml; charset=utf-8", data)
	}
}

func redirectDocsSlash(c *gin.Context) {
	c.Redirect(http.StatusFound, "/docs/")
}

func serveSwaggerUI() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Content-Type", "text/html; charset=utf-8")
		c.Header("Cache-Control", "public, max-age=300")
		c.String(http.StatusOK, swaggerUIPage)
	}
}
