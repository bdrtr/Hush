package hush

import (
	"github.com/valyala/fasthttp"
)

// NewTestContext creates a mock Context for isolated handler unit testing.
// It returns the Context and a cleanup function that MUST be called to prevent memory leaks.
func NewTestContext(method, path string) (*Context, func()) {
	ctx := &fasthttp.RequestCtx{}
	ctx.Request.Header.SetMethod(method)
	ctx.Request.SetRequestURI(path)
	
	c := contextPool.Get().(*Context)
	c.reset(ctx, nil)
	
	return c, func() {
		contextPool.Put(c)
	}
}
