package hush

import (
	"testing"
)

func TestRouter_TableDriven(t *testing.T) {
	tests := []struct {
		name       string
		routes     []string
		method     string
		reqPath    string
		expectPath string
		expectParams map[string]string
		expectNil  bool
	}{
		{
			name:       "Static Route Exact Match",
			routes:     []string{"/hello"},
			method:     "GET",
			reqPath:    "/hello",
			expectPath: "/hello",
		},
		{
			name:       "Param Route Match",
			routes:     []string{"/users/:id"},
			method:     "GET",
			reqPath:    "/users/123",
			expectPath: "/users/:id",
			expectParams: map[string]string{"id": "123"},
		},
		{
			name:       "Static Over Dynamic Priority",
			routes:     []string{"/users/:id", "/users/profile"},
			method:     "GET",
			reqPath:    "/users/profile",
			expectPath: "/users/profile", // Should match static first
		},
		{
			name:       "Wildcard Match",
			routes:     []string{"/static/*filepath"},
			method:     "GET",
			reqPath:    "/static/css/main.css",
			expectPath: "/static/*filepath",
			expectParams: map[string]string{"filepath": "css/main.css"},
		},
		{
			name:       "Not Found Path",
			routes:     []string{"/hello"},
			method:     "GET",
			reqPath:    "/world",
			expectNil:  true,
		},
		{
			name:       "Trailing Slash Exactness",
			routes:     []string{"/hello/"},
			method:     "GET",
			reqPath:    "/hello",
			expectNil:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := newRouter()
			for _, route := range tt.routes {
				r.insert("GET", route, []HandlerFunc{func(c *Context) {}})
			}

			c := &Context{}
			n := r.get(tt.method, tt.reqPath, c)

			if tt.expectNil {
				if n != nil {
					t.Fatalf("Expected nil route, got %v", n.path)
				}
				return
			}

			if n == nil {
				t.Fatalf("Expected route %s, got nil", tt.expectPath)
			}

			if n.path != tt.expectPath {
				t.Errorf("Expected path %s, got %s", tt.expectPath, n.path)
			}

			if tt.expectParams != nil {
				for k, v := range tt.expectParams {
					if c.Param(k) != v {
						t.Errorf("Expected param %s=%s, got %s", k, v, c.Param(k))
					}
				}
			}
		})
	}
}

func TestRouter_ConflictPanic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("The code did not panic when inserting conflicting dynamic routes")
		}
	}()

	r := newRouter()
	r.insert("GET", "/users/:id", []HandlerFunc{func(c *Context) {}})
	r.insert("GET", "/users/:username", []HandlerFunc{func(c *Context) {}})
}
