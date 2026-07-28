package httpapi

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/neirinzaralwin/patient_management_system_api/docs/openapi"
)

const (
	openapiSpecPath = "/openapi/openapi.yaml"
	swaggerUIPath   = "/docs/swagger"
	redocPath       = "/docs/redoc"
)

// registerDocs mounts the OpenAPI spec, Swagger UI, and ReDoc when enabled.
// These routes are unauthenticated developer tooling and must stay outside
// any JWT-protected group.
func registerDocs(engine *gin.Engine) {
	engine.GET(openapiSpecPath, serveOpenAPISpec)
	engine.GET(swaggerUIPath, serveSwaggerUI)
	engine.GET(swaggerUIPath+"/", serveSwaggerUI)
	engine.GET(redocPath, serveReDoc)
	engine.GET(redocPath+"/", serveReDoc)
}

func serveOpenAPISpec(c *gin.Context) {
	c.Data(http.StatusOK, "application/yaml; charset=utf-8", openapi.Spec)
}

func serveSwaggerUI(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(swaggerUIHTML))
}

func serveReDoc(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(redocHTML))
}

// Swagger UI and ReDoc load from a CDN so the binary stays dependency-light.
// Both UIs point at the same in-process OpenAPI route.
const swaggerUIHTML = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>API Docs — Swagger UI</title>
  <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5/swagger-ui.css">
  <style>
    body { margin: 0; background: #fafafa; }
  </style>
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
  <script src="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5/swagger-ui-standalone-preset.js"></script>
  <script>
    window.onload = function () {
      window.ui = SwaggerUIBundle({
        url: "` + openapiSpecPath + `",
        dom_id: "#swagger-ui",
        presets: [SwaggerUIBundle.presets.apis, SwaggerUIStandalonePreset],
        layout: "StandaloneLayout",
        deepLinking: true,
        persistAuthorization: true
      });
    };
  </script>
</body>
</html>
`

const redocHTML = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>API Docs — ReDoc</title>
  <style>
    body { margin: 0; padding: 0; }
  </style>
</head>
<body>
  <redoc spec-url="` + openapiSpecPath + `"></redoc>
  <script src="https://cdn.jsdelivr.net/npm/redoc@2/bundles/redoc.standalone.js"></script>
</body>
</html>
`
