package middleware_test

import (
	"testing"

	"github.com/bdrtr/hush"
	"github.com/bdrtr/hush/middleware"
	"github.com/valyala/fasthttp"
)

func TestMiddlewares(t *testing.T) {
	app := hush.New()
	
	// Add all middlewares to test
	app.Use(middleware.CORS("*"))
	app.Use(middleware.Logger())
	app.Use(middleware.Recovery())
	
	// Route that panics
	app.GET("/panic", func(c *hush.Context) {
		panic("test panic")
	})
	
	// Normal route
	app.GET("/ok", func(c *hush.Context) {
		c.Ok("ok")
	})
	
	// Simulate request to /ok
	ctxOk := &fasthttp.RequestCtx{}
	ctxOk.Request.SetRequestURI("/ok")
	ctxOk.Request.Header.SetMethod(fasthttp.MethodGet)
	
	app.Handler(ctxOk)
	
	// Verify CORS header was added
	if string(ctxOk.Response.Header.Peek("Access-Control-Allow-Origin")) != "*" {
		t.Errorf("Expected CORS header missing on normal request")
	}
	
	// Simulate request to /panic
	ctxPanic := &fasthttp.RequestCtx{}
	ctxPanic.Request.SetRequestURI("/panic")
	ctxPanic.Request.Header.SetMethod(fasthttp.MethodGet)
	
	// Run handler, expecting recovery to catch panic
	app.Handler(ctxPanic)
	
	// Verify it returned a 500 error instead of crashing
	if ctxPanic.Response.StatusCode() != fasthttp.StatusInternalServerError {
		t.Errorf("Expected 500 status code on panic, got %d", ctxPanic.Response.StatusCode())
	}
	
	// Verify CORS header was still added despite panic
	if string(ctxPanic.Response.Header.Peek("Access-Control-Allow-Origin")) != "*" {
		t.Errorf("Expected CORS header missing on panic request")
	}
}
