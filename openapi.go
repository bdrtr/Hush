package hush

import (
	"reflect"
	"regexp"
	"strings"
	"time"

	"github.com/valyala/fasthttp"
)

var paramRegex = regexp.MustCompile(`([:\*])([a-zA-Z0-9_]+)`)

// SwaggerSpec represents a simplified OpenAPI 3.0 specification.
type SwaggerSpec struct {
	OpenAPI    string                 `json:"openapi"`
	Info       map[string]string      `json:"info"`
	Paths      map[string]interface{} `json:"paths"`
	Components map[string]interface{} `json:"components,omitempty"`
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
	
	e.mu.RLock()
	routes := make([]*Route, len(e.routes))
	copy(routes, e.routes)
	e.mu.RUnlock()
	
	for _, route := range routes {
		openAPIPath := paramRegex.ReplaceAllString(route.Path, "{$2}")

		if _, ok := spec.Paths[openAPIPath]; !ok {
			spec.Paths[openAPIPath] = make(map[string]interface{})
		}
		
		pathItem := spec.Paths[openAPIPath].(map[string]interface{})
		methodLower := strings.ToLower(route.Method)
		
		op := map[string]interface{}{
			"responses": map[string]interface{}{
				"200": map[string]interface{}{
					"description": "OK",
				},
			},
		}

		if route.Summary != "" {
			op["summary"] = route.Summary
		}
		if len(route.Tags) > 0 {
			op["tags"] = route.Tags
		}

		var parameters []map[string]interface{}
		
		// Auto-extract path parameters (e.g., :id or *filepath)
		matches := paramRegex.FindAllStringSubmatch(route.Path, -1)
		for _, match := range matches {
			if len(match) > 2 {
				paramName := match[2]
				parameters = append(parameters, map[string]interface{}{
					"name":     paramName,
					"in":       "path",
					"required": true,
					"schema": map[string]interface{}{
						"type": "string",
					},
				})
			}
		}
		
		if route.QueryParams != nil {
			parameters = append(parameters, buildParameters(route.QueryParams, "query")...)
		}
		if route.HeaderParams != nil {
			parameters = append(parameters, buildParameters(route.HeaderParams, "header")...)
		}
		
		if len(parameters) > 0 {
			op["parameters"] = parameters
		}

		if route.RequestBody != nil {
			op["requestBody"] = map[string]interface{}{
				"content": map[string]interface{}{
					"application/json": map[string]interface{}{
						"schema": buildSchema(route.RequestBody),
					},
				},
			}
		}

		if route.ResponseBody != nil {
			op["responses"].(map[string]interface{})["200"] = map[string]interface{}{
				"description": "Successful Response",
				"content": map[string]interface{}{
					"application/json": map[string]interface{}{
						"schema": buildSchema(route.ResponseBody),
					},
				},
			}
		}
		
		pathItem[methodLower] = op
	}
	
	return spec
}

// buildParameters extracts OpenAPI parameters from struct tags.
// Struct tags like `query:"name"` or `header:"name"` should be used.
// Example: Name string `query:"name" validate:"required"`
func buildParameters(t reflect.Type, in string) []map[string]interface{} {
	var params []map[string]interface{}
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return params
	}
	
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		
		tag := field.Tag.Get(in)
		if tag == "" || tag == "-" {
			continue
		}
		
		param := map[string]interface{}{
			"name": tag,
			"in":   in,
			"required": strings.Contains(field.Tag.Get("validate"), "required"),
			"schema": buildSchemaWithSeen(field.Type, make(map[reflect.Type]bool)),
		}
		params = append(params, param)
	}
	return params
}

// buildSchema recursively builds an OpenAPI schema from a reflect.Type
func buildSchema(t reflect.Type) map[string]interface{} {
	return buildSchemaWithSeen(t, make(map[reflect.Type]bool))
}

func buildSchemaWithSeen(t reflect.Type, seen map[reflect.Type]bool) map[string]interface{} {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	if seen[t] {
		return map[string]interface{}{"type": "object"} // Break recursive cycle
	}
	if t.Kind() == reflect.Struct {
		if t == reflect.TypeOf(time.Time{}) {
			schema["type"] = "string"
			schema["format"] = "date-time"
			return schema
		}
		seen[t] = true
	}

	schema := make(map[string]interface{})

	switch t.Kind() {
	case reflect.Struct:
		schema["type"] = "object"
		properties := make(map[string]interface{})
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			jsonTag := field.Tag.Get("json")
			if jsonTag == "" || jsonTag == "-" {
				jsonTag = field.Name
			} else {
				jsonTag = strings.Split(jsonTag, ",")[0]
			}
			properties[jsonTag] = buildSchemaWithSeen(field.Type, seen)
		}
		schema["properties"] = properties
	case reflect.Slice, reflect.Array:
		schema["type"] = "array"
		schema["items"] = buildSchemaWithSeen(t.Elem(), seen)
	case reflect.Map:
		schema["type"] = "object"
		schema["additionalProperties"] = buildSchemaWithSeen(t.Elem(), seen)
	case reflect.String:
		schema["type"] = "string"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		schema["type"] = "integer"
	case reflect.Float32, reflect.Float64:
		schema["type"] = "number"
	case reflect.Bool:
		schema["type"] = "boolean"
	default:
		schema["type"] = "string" // fallback
	}

	return schema
}

// ServeSwaggerUI serves the swagger.json and a basic UI.
func (e *Engine) ServeSwaggerUI(path string) {
	if !strings.HasPrefix(path, "/") || strings.ContainsAny(path, "'\"<>") {
		panic("hush: invalid swagger path, potential XSS or malformed route")
	}

	e.GET(path+"/swagger.json", func(c *Context) {
		spec := e.GenerateOpenAPI()
		c.Ok(spec)
	}).WithSummary("Serve OpenAPI Specification")
	
	e.GET(path, func(c *Context) {
		html := `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <meta name="description" content="SwaggerUI" />
  <title>SwaggerUI</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5.11.8/swagger-ui.css" integrity="sha384-H/0BRJAt4dZN0emsA7KWNBXSR7MAz3EbpckPsfkxP0pn7zYIZbH087mKXFoBkXNw" crossorigin="anonymous" />
</head>
<body>
<div id="swagger-ui"></div>
<script src="https://unpkg.com/swagger-ui-dist@5.11.8/swagger-ui-bundle.js" integrity="sha384-WRSjyuE/ddFcTrGFUQ9YBNKPru2cRfeWfTSu6ATw7SzHl1f+TUV7JafqRk/6NaSq" crossorigin="anonymous"></script>
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
	}).WithSummary("Serve Swagger UI")
}
