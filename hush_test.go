package hush

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/valyala/fasthttp"
)

func TestEngine_ServeHTTP(t *testing.T) {
	app := New()
	app.GET("/ping", func(c *Context) {
		c.Ok(map[string]string{"message": "pong"})
	})

	ctx := &fasthttp.RequestCtx{}
	ctx.Request.Header.SetMethod(fasthttp.MethodGet)
	ctx.Request.SetRequestURI("/ping")

	app.Handler(ctx)

	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Errorf("Expected status 200, got %d", ctx.Response.StatusCode())
	}
}

func TestEngine_404Handler(t *testing.T) {
	app := New()
	
	ctx := &fasthttp.RequestCtx{}
	ctx.Request.Header.SetMethod(fasthttp.MethodGet)
	ctx.Request.SetRequestURI("/not-found")

	app.Handler(ctx)

	if ctx.Response.StatusCode() != fasthttp.StatusNotFound {
		t.Errorf("Expected status 404, got %d", ctx.Response.StatusCode())
	}
}

func TestEngine_OPTIONSPreflight(t *testing.T) {
	app := New()
	app.GET("/api/data", func(c *Context) {})
	// CORS middleware usually handles OPTIONS, but let's test built-in OPTIONS handler if registered
	app.OPTIONS("/api/data", func(c *Context) {
		c.Ctx.SetStatusCode(fasthttp.StatusNoContent)
	})

	ctx := &fasthttp.RequestCtx{}
	ctx.Request.Header.SetMethod(fasthttp.MethodOptions)
	ctx.Request.SetRequestURI("/api/data")

	app.Handler(ctx)

	if ctx.Response.StatusCode() != fasthttp.StatusNoContent {
		t.Errorf("Expected status 204 for OPTIONS, got %d", ctx.Response.StatusCode())
	}
}

func TestEngine_StaticPathTraversal(t *testing.T) {
	app := New()
	// Create a temporary static dir
	os.MkdirAll("test_static", 0755)
	os.WriteFile("test_static/index.html", []byte("hello"), 0644)
	defer os.RemoveAll("test_static")

	app.Static("/static", "test_static")

	ctx := &fasthttp.RequestCtx{}
	ctx.Request.Header.SetMethod(fasthttp.MethodGet)
	// Attempt path traversal
	ctx.Request.SetRequestURI("/static/../../../../../../etc/passwd")

	app.Handler(ctx)

	// fasthttp FS should block this and return 400, 403, or 404
	if ctx.Response.StatusCode() == fasthttp.StatusOK {
		t.Errorf("Path traversal attempt succeeded when it should have been blocked")
	}
}

func TestEngine_GracefulShutdown(t *testing.T) {
	app := New()
	app.GET("/sleep", func(c *Context) {
		time.Sleep(2 * time.Second) // Simulate long request
	})

	app.server = &fasthttp.Server{
		Handler: app.Handler,
	}

	// Context with very short timeout to trigger forced close
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	err := app.Shutdown(ctx)
	if err != nil {
		t.Errorf("Expected nil from immediate shutdown, got %v", err)
	}
}
