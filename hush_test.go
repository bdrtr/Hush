package hush

import (
	"testing"
	"github.com/valyala/fasthttp"
)

func TestRouterStatic(t *testing.T) {
	r := newRouter()
	r.insert("GET", "/hello", []HandlerFunc{func(c *Context) {}})
	
	c := &Context{}
	n := r.get("GET", "/hello", c)
	if n == nil || n.path != "/hello" {
		t.Errorf("Expected route /hello, got nil or wrong path")
	}
}

func TestRouterParam(t *testing.T) {
	r := newRouter()
	r.insert("GET", "/users/:id", []HandlerFunc{func(c *Context) {}})
	
	c := &Context{}
	n := r.get("GET", "/users/123", c)
	if n == nil {
		t.Fatal("Expected route, got nil")
	}
	
	id := c.Param("id")
	if id != "123" {
		t.Errorf("Expected param id=123, got %s", id)
	}
}

func TestEngineServe(t *testing.T) {
	app := New()
	app.GET("/ping", func(c *Context) {
		c.Ok(map[string]string{"message": "pong"})
	})
	
	ctx := &fasthttp.RequestCtx{}
	ctx.Request.Header.SetMethod("GET")
	ctx.Request.SetRequestURI("/ping")
	
	app.Handler(ctx)
	
	if ctx.Response.StatusCode() != 200 {
		t.Errorf("Expected status 200, got %d", ctx.Response.StatusCode())
	}
}
