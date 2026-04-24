package hush

import (
	"github.com/valyala/fasthttp"
)

// NewTestContext creates a mock Context for isolated handler unit testing.
func NewTestContext(method, path string) *Context {
	ctx := &fasthttp.RequestCtx{}
	ctx.Request.Header.SetMethod(method)
	ctx.Request.SetRequestURI(path)
	
	c := contextPool.Get().(*Context)
	c.reset(ctx, nil)
	
	return c
}
