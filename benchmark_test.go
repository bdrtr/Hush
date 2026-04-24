package hush

import (
	"testing"
	"github.com/valyala/fasthttp"
)

// BenchmarkRouter_Static measures the performance of routing to a static path
func BenchmarkRouter_Static(b *testing.B) {
	app := New()
	app.GET("/hello", func(c *Context) {})

	ctx := &fasthttp.RequestCtx{}
	ctx.Request.Header.SetMethod("GET")
	ctx.Request.SetRequestURI("/hello")

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		app.Handler(ctx)
	}
}

// BenchmarkRouter_Param measures the performance of routing to a parameterized path
func BenchmarkRouter_Param(b *testing.B) {
	app := New()
	app.GET("/users/:id", func(c *Context) {})

	ctx := &fasthttp.RequestCtx{}
	ctx.Request.Header.SetMethod("GET")
	ctx.Request.SetRequestURI("/users/12345")

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		app.Handler(ctx)
	}
}

// BenchmarkRouter_Wildcard measures the performance of routing to a wildcard path
func BenchmarkRouter_Wildcard(b *testing.B) {
	app := New()
	app.GET("/assets/*filepath", func(c *Context) {})

	ctx := &fasthttp.RequestCtx{}
	ctx.Request.Header.SetMethod("GET")
	ctx.Request.SetRequestURI("/assets/js/main.js")

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		app.Handler(ctx)
	}
}

// BenchmarkJSONResponse measures the performance of JSON serialization using Sonic
func BenchmarkJSONResponse(b *testing.B) {
	app := New()
	app.GET("/json", func(c *Context) {
		c.JSON(200, map[string]string{"status": "ok", "message": "hello world"})
	})

	ctx := &fasthttp.RequestCtx{}
	ctx.Request.Header.SetMethod("GET")
	ctx.Request.SetRequestURI("/json")

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		app.Handler(ctx)
	}
}
