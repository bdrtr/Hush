package hush

import (
	"time"

	"github.com/valyala/fasthttp"
)

// RouterGroup is used to group routes with prefixes and middlewares.
type RouterGroup struct {
	prefix      string
	middlewares []HandlerFunc
	engine      *Engine
}

// Use adds middleware to the group.
func (rg *RouterGroup) Use(middlewares ...HandlerFunc) {
	rg.middlewares = append(rg.middlewares, middlewares...)
}

// Group creates a new sub-group.
func (rg *RouterGroup) Group(prefix string) *RouterGroup {
	return &RouterGroup{
		prefix:      rg.prefix + prefix,
		middlewares: append([]HandlerFunc{}, rg.middlewares...),
		engine:      rg.engine,
	}
}

// addRoute handles the actual route registration.
func (rg *RouterGroup) addRoute(method, comp string, handlers []HandlerFunc) *Route {
	path := rg.prefix + comp
	finalHandlers := append([]HandlerFunc{}, rg.middlewares...)
	finalHandlers = append(finalHandlers, handlers...)

	rg.engine.mu.Lock()
	defer rg.engine.mu.Unlock()

	rg.engine.router.insert(method, path, finalHandlers)

	route := &Route{
		Method: method,
		Path:   path,
	}
	rg.engine.routes = append(rg.engine.routes, route)
	return route
}

// GET registers a GET route and returns a Route object for building options.
func (rg *RouterGroup) GET(path string, handlers ...HandlerFunc) *Route {
	return rg.addRoute(fasthttp.MethodGet, path, handlers)
}

// POST registers a POST route and returns a Route object for building options.
func (rg *RouterGroup) POST(path string, handlers ...HandlerFunc) *Route {
	return rg.addRoute(fasthttp.MethodPost, path, handlers)
}

// PUT registers a PUT route and returns a Route object for building options.
func (rg *RouterGroup) PUT(path string, handlers ...HandlerFunc) *Route {
	return rg.addRoute(fasthttp.MethodPut, path, handlers)
}

// PATCH registers a PATCH route and returns a Route object for building options.
func (rg *RouterGroup) PATCH(path string, handlers ...HandlerFunc) *Route {
	return rg.addRoute(fasthttp.MethodPatch, path, handlers)
}

// DELETE registers a DELETE route and returns a Route object for building options.
func (rg *RouterGroup) DELETE(path string, handlers ...HandlerFunc) *Route {
	return rg.addRoute(fasthttp.MethodDelete, path, handlers)
}

// HEAD registers a HEAD route and returns a Route object for building options.
func (rg *RouterGroup) HEAD(path string, handlers ...HandlerFunc) *Route {
	return rg.addRoute(fasthttp.MethodHead, path, handlers)
}

// OPTIONS registers an OPTIONS route and returns a Route object for building options.
func (rg *RouterGroup) OPTIONS(path string, handlers ...HandlerFunc) *Route {
	return rg.addRoute(fasthttp.MethodOptions, path, handlers)
}

// Static serves static files from the given root directory securely.
func (rg *RouterGroup) Static(path, root string) {
	fs := &fasthttp.FS{
		Root:               root,
		IndexNames:         []string{"index.html"},
		GenerateIndexPages: false,
		Compress:           false, // Disabled to prevent zip-bomb / CPU exhaustion attacks
		AcceptByteRange:    true,
		CacheDuration:      10 * time.Second, // Prevent disk thrashing
		PathRewrite:        fasthttp.NewPathPrefixStripper(len(path)),
	}
	fsHandler := fs.NewRequestHandler()

	handler := func(c *Context) {
		fsHandler(c.Ctx)
	}
	rg.GET(path+"/*filepath", handler)
	rg.GET(path, handler)
}
