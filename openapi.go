package hush

import (
	"encoding/json"
	"net/http"
)

// SwaggerSpec represents a simplified OpenAPI 3.0 specification.
type SwaggerSpec struct {
	OpenAPI string                 `json:"openapi"`
	Info    map[string]string      `json:"info"`
	Paths   map[string]interface{} `json:"paths"`
}

// GenerateOpenAPI creates a basic swagger.json
func (e *Engine) GenerateOpenAPI() *SwaggerSpec {
	spec := &SwaggerSpec{
		OpenAPI: "3.0.0",
		Info: map[string]string{
			"title":   "Hush API",
			"version": "1.0.0",
		},
		Paths: make(map[string]interface{}),
	}
	
	// A real implementation would traverse e.router.routes and use reflection 
	// to document schemas. For Phase 1, we just return a skeleton.
	// Users can serve this using /docs endpoint.
	
	return spec
}

// ServeSwaggerUI serves the swagger.json and a basic UI.
func (e *Engine) ServeSwaggerUI(path string) {
	e.GET(path+"/swagger.json", func(c *Context) {
		spec := e.GenerateOpenAPI()
		c.Ok(spec)
	})
	
	e.GET(path, func(c *Context) {
		html := `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <meta name="description" content="SwaggerUI" />
  <title>SwaggerUI</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@4.5.0/swagger-ui.css" />
</head>
<body>
<div id="swagger-ui"></div>
<script src="https://unpkg.com/swagger-ui-dist@4.5.0/swagger-ui-bundle.js" crossorigin></script>
<script>
  window.onload = () => {
    window.ui = SwaggerUIBundle({
      url: '` + path + `/swagger.json',
      dom_id: '#swagger-ui',
    });
  };
</script>
</body>
</html>`
		c.Writer.Header().Set("Content-Type", "text/html; charset=utf-8")
		c.Writer.WriteHeader(http.StatusOK)
		c.Writer.Write([]byte(html))
	})
}
