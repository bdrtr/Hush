package hush

import (
	"testing"

	"github.com/fasthttp/websocket"
	"github.com/valyala/fasthttp"
)

func TestContextHelpers(t *testing.T) {
	c, cleanup := NewTestContext(fasthttp.MethodGet, "/test")
	defer cleanup()

	// Test JSON
	c.JSON(fasthttp.StatusOK, map[string]string{"msg": "ok"})
	if string(c.Ctx.Response.Header.ContentType()) != "application/json" {
		t.Errorf("Expected JSON content type")
	}

	// Test Error
	c.Error("bad request", fasthttp.StatusBadRequest)
	if c.Ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Errorf("Expected 400, got %d", c.Ctx.Response.StatusCode())
	}

	// Test Redirect
	c.Redirect("/login", fasthttp.StatusFound)
	if c.Ctx.Response.StatusCode() != fasthttp.StatusFound {
		t.Errorf("Expected 302, got %d", c.Ctx.Response.StatusCode())
	}
	if string(c.Ctx.Response.Header.Peek("Location")) != "/login" {
		t.Errorf("Expected Redirect location /login")
	}
}

func TestContextWebSocket_OriginFail(t *testing.T) {
	c, cleanup := NewTestContext(fasthttp.MethodGet, "/ws")
	defer cleanup()

	// Pretend the client sends a malicious origin
	c.Ctx.Request.Header.Set("Origin", "http://evil.com")
	
	// Upgrade requires specific origin
	err := c.Upgrade([]string{"http://localhost:8080"}, func(conn *websocket.Conn) {})
	
	if err == nil {
		t.Fatalf("Expected WebSocket upgrade to fail due to origin mismatch")
	}
	
	if c.Ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Errorf("Expected 400 Bad Request, got %d", c.Ctx.Response.StatusCode())
	}
}
