package middleware_test

import (
	"testing"
	"time"

	"github.com/bdrtr/hush"
	"github.com/bdrtr/hush/middleware"
	"github.com/valyala/fasthttp"
)

func TestMiddlewares_CORSAndRecovery(t *testing.T) {
	app := hush.New()
	app.Use(middleware.CORS("*"))
	app.Use(middleware.Recovery())

	app.GET("/panic", func(c *hush.Context) {
		panic("test panic")
	})

	app.GET("/ok", func(c *hush.Context) {
		c.Ok("ok")
	})

	// Test normal
	ctxOk := &fasthttp.RequestCtx{}
	ctxOk.Request.SetRequestURI("/ok")
	ctxOk.Request.Header.SetMethod(fasthttp.MethodGet)
	app.Handler(ctxOk)

	if string(ctxOk.Response.Header.Peek("Access-Control-Allow-Origin")) != "*" {
		t.Errorf("Expected CORS header missing on normal request")
	}

	// Test Panic Recovery
	ctxPanic := &fasthttp.RequestCtx{}
	ctxPanic.Request.SetRequestURI("/panic")
	ctxPanic.Request.Header.SetMethod(fasthttp.MethodGet)
	app.Handler(ctxPanic)

	if ctxPanic.Response.StatusCode() != fasthttp.StatusInternalServerError {
		t.Errorf("Expected 500 status code on panic, got %d", ctxPanic.Response.StatusCode())
	}
	if string(ctxPanic.Response.Header.Peek("Access-Control-Allow-Origin")) != "*" {
		t.Errorf("Expected CORS header missing on panic request")
	}
}

func TestCacheMiddleware(t *testing.T) {
	app := hush.New()
	
	callCount := 0
	app.Use(middleware.Cache(100*time.Millisecond, 10, 1024*1024))
	
	app.GET("/data", func(c *hush.Context) {
		callCount++
		c.Ok("cached data")
	})

	// First call - Cache Miss
	ctx1 := &fasthttp.RequestCtx{}
	ctx1.Request.SetRequestURI("/data")
	ctx1.Request.Header.SetMethod(fasthttp.MethodGet)
	app.Handler(ctx1)

	if callCount != 1 {
		t.Errorf("Expected call count 1, got %d", callCount)
	}

	// Second call - Cache Hit
	ctx2 := &fasthttp.RequestCtx{}
	ctx2.Request.SetRequestURI("/data")
	ctx2.Request.Header.SetMethod(fasthttp.MethodGet)
	app.Handler(ctx2)

	if callCount != 1 {
		t.Errorf("Expected call count 1 (cache hit), got %d", callCount)
	}

	// Wait for TTL expiry
	time.Sleep(150 * time.Millisecond)

	// Third call - Cache Miss (expired)
	ctx3 := &fasthttp.RequestCtx{}
	ctx3.Request.SetRequestURI("/data")
	ctx3.Request.Header.SetMethod(fasthttp.MethodGet)
	app.Handler(ctx3)

	if callCount != 2 {
		t.Errorf("Expected call count 2 (cache expired), got %d", callCount)
	}
}

func TestRateLimitMiddleware(t *testing.T) {
	app := hush.New()
	
	// Allow 2 requests per 100ms
	app.Use(middleware.RateLimit(2, 100*time.Millisecond))
	
	app.GET("/ping", func(c *hush.Context) {
		c.Ok("pong")
	})

	runReq := func(ip string) int {
		ctx := &fasthttp.RequestCtx{}
		ctx.Request.SetRequestURI("/ping")
		ctx.Request.Header.SetMethod(fasthttp.MethodGet)
		ctx.Request.Header.Set("X-Forwarded-For", ip)
		app.Handler(ctx)
		return ctx.Response.StatusCode()
	}

	// Client 1
	if status := runReq("192.168.1.1"); status != fasthttp.StatusOK {
		t.Errorf("Req 1 failed, status %d", status)
	}
	if status := runReq("192.168.1.1"); status != fasthttp.StatusOK {
		t.Errorf("Req 2 failed, status %d", status)
	}
	if status := runReq("192.168.1.1"); status != fasthttp.StatusTooManyRequests {
		t.Errorf("Req 3 should be 429 Too Many Requests, got %d", status)
	}

	// Client 2 should be unaffected
	if status := runReq("192.168.1.2"); status != fasthttp.StatusOK {
		t.Errorf("Client 2 Req 1 failed, status %d", status)
	}
}

func TestJWTMiddleware(t *testing.T) {
	secret := "test-secret"
	app := hush.New()
	
	app.GET("/protected", middleware.JWT(secret), func(c *hush.Context) {
		c.Ok("secret data")
	})

	// Missing token
	ctx1 := &fasthttp.RequestCtx{}
	ctx1.Request.SetRequestURI("/protected")
	ctx1.Request.Header.SetMethod(fasthttp.MethodGet)
	app.Handler(ctx1)

	if ctx1.Response.StatusCode() != fasthttp.StatusUnauthorized {
		t.Errorf("Expected 401 for missing token, got %d", ctx1.Response.StatusCode())
	}

	// Tampered token
	ctx2 := &fasthttp.RequestCtx{}
	ctx2.Request.SetRequestURI("/protected")
	ctx2.Request.Header.SetMethod(fasthttp.MethodGet)
	ctx2.Request.Header.Set("Authorization", "Bearer invalid.token.string")
	app.Handler(ctx2)

	if ctx2.Response.StatusCode() != fasthttp.StatusUnauthorized {
		t.Errorf("Expected 401 for tampered token, got %d", ctx2.Response.StatusCode())
	}
}

func TestStatsMiddleware(t *testing.T) {
	app := hush.New()
	app.Use(middleware.Stats())
	
	app.GET("/ok", func(c *hush.Context) {
		c.Ok("ok")
	})
	
	app.GET("/error", func(c *hush.Context) {
		c.Ctx.SetStatusCode(500)
	})

	initialStats := middleware.GetStats()

	ctx1 := &fasthttp.RequestCtx{}
	ctx1.Request.SetRequestURI("/ok")
	ctx1.Request.Header.SetMethod(fasthttp.MethodGet)
	app.Handler(ctx1)

	ctx2 := &fasthttp.RequestCtx{}
	ctx2.Request.SetRequestURI("/error")
	ctx2.Request.Header.SetMethod(fasthttp.MethodGet)
	app.Handler(ctx2)

	stats := middleware.GetStats()
	
	if stats.TotalRequests != initialStats.TotalRequests + 2 {
		t.Errorf("Expected +2 total requests, got %d", stats.TotalRequests - initialStats.TotalRequests)
	}
	if stats.ErrorResponses != initialStats.ErrorResponses + 1 {
		t.Errorf("Expected +1 error response, got %d", stats.ErrorResponses - initialStats.ErrorResponses)
	}
	if stats.ActiveRequests != 0 {
		t.Errorf("Expected 0 active requests, got %d", stats.ActiveRequests)
	}
}

func TestContextualLogging(t *testing.T) {
	// This test primarily ensures the middleware doesn't panic when injected with request_id
	app := hush.New()
	
	// Use RequestID first to inject into context, then Logger to read it
	app.Use(middleware.RequestID())
	app.Use(middleware.Logger())
	
	app.GET("/test", func(c *hush.Context) {
		reqId, ok := c.Get("request_id")
		if !ok || reqId == "" {
			t.Errorf("Expected request_id to be set by middleware")
		}
		c.Ok("ok")
	})

	ctx := &fasthttp.RequestCtx{}
	ctx.Request.SetRequestURI("/test")
	ctx.Request.Header.SetMethod(fasthttp.MethodGet)
	app.Handler(ctx)
}
