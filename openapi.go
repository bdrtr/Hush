package hush

import (
	"github.com/valyala/fasthttp"
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
	
	for _, route := range e.routes {
		if _, ok := spec.Paths[route.Path]; !ok {
			spec.Paths[route.Path] = make(map[string]interface{})
		}
		
		pathItem := spec.Paths[route.Path].(map[string]interface{})
		methodLower := strings.ToLower(route.Method)
		
		pathItem[methodLower] = map[string]interface{}{
			"summary": route.Summary,
			"tags":    route.Tags,
			"responses": map[string]interface{}{
				"200": map[string]interface{}{
					"description": "OK",
				},
			},
		}
	}
	
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
		c.Ctx.Response.Header.Set("Content-Type", "text/html; charset=utf-8")
		c.Ctx.SetStatusCode(fasthttp.StatusOK)
		c.Ctx.Write([]byte(html))
	})
}
