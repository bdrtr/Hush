package hush

import (
	"testing"
)

func TestGroup_MaxHandlersPanic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("The code did not panic when exceeding max handlers limit")
		}
	}()

	e := New()
	var handlers []HandlerFunc
	for i := 0; i < 64; i++ {
		handlers = append(handlers, func(c *Context) {})
	}

	// This should panic because maxHandlers is 63
	e.GET("/test", handlers...)
}

func TestGroup_MiddlewareInheritance(t *testing.T) {
	e := New()
	e.Use(func(c *Context) {}) // 1 middleware

	api := e.Group("/api")
	api.Use(func(c *Context) {}) // +1 middleware = 2

	v1 := api.Group("/v1")
	v1.Use(func(c *Context) {}) // +1 middleware = 3

	// Add another middleware to api AFTER v1 is created.
	// v1 should NOT inherit this.
	api.Use(func(c *Context) {}) // api = 3, v1 still 3

	v1.GET("/test", func(c *Context) {}) // +1 endpoint handler = 4 total for v1 route

	// Find the node in the router to inspect handlers
	c := &Context{}
	n := e.router.get("GET", "/api/v1/test", c)

	if n == nil {
		t.Fatalf("Route not found")
	}

	if len(n.handlers) != 4 {
		t.Errorf("Expected 4 handlers (3 middlewares + 1 endpoint), got %d", len(n.handlers))
	}
}
