package hush

import (
	"net/http"
	"net/http/httptest"
)

// NewTestServer creates an in-memory httptest.Server using the Engine.
func NewTestServer(e *Engine) *httptest.Server {
	return httptest.NewServer(e)
}

// NewTestContext creates a mock Context for isolated handler unit testing.
func NewTestContext(method, path string) (*Context, *httptest.ResponseRecorder) {
	req := httptest.NewRequest(method, path, nil)
	w := httptest.NewRecorder()
	
	c := contextPool.Get().(*Context)
	c.reset(w, req, nil)
	
	return c, w
}
