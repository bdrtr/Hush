package hush

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"reflect"
	"syscall"
	"time"

	"github.com/valyala/fasthttp"
)

// Engine is the main framework instance.
type Engine struct {
	*RouterGroup
	router    *Router
	container map[reflect.Type]interface{}
	server    *fasthttp.Server
	routes    []*Route
}

// Route represents a registered route and its metadata for OpenAPI
type Route struct {
	Method       string
	Path         string
	Summary      string
	Tags         []string
	RequestBody  reflect.Type
	ResponseBody reflect.Type
}

// WithSummary adds a summary to the route for OpenAPI
func (r *Route) WithSummary(summary string) *Route {
	r.Summary = summary
	return r
}

// WithTags adds tags to the route for OpenAPI
func (r *Route) WithTags(tags ...string) *Route {
	r.Tags = append(r.Tags, tags...)
	return r
}

// WithBody documents the request body type for OpenAPI
func WithBody[T any](r *Route) *Route {
	r.RequestBody = reflect.TypeOf((*T)(nil)).Elem()
	return r
}

// WithResponse documents the response body type for OpenAPI
func WithResponse[T any](r *Route) *Route {
	r.ResponseBody = reflect.TypeOf((*T)(nil)).Elem()
	return r
}

// RouterGroup is used to group routes with prefixes and middlewares.
type RouterGroup struct {
	prefix      string
	middlewares []HandlerFunc
	engine      *Engine
}

// New creates a new Hush Engine based on fasthttp.
func New() *Engine {
	engine := &Engine{
		router:    newRouter(),
		container: make(map[reflect.Type]interface{}),
	}
	engine.RouterGroup = &RouterGroup{
		engine: engine,
	}
	return engine
}

// Provide registers a singleton dependency.
func Provide[T any](e *Engine, instance T) {
	typ := reflect.TypeOf((*T)(nil)).Elem()
	e.container[typ] = instance
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
		Compress:           true,
		PathRewrite:        fasthttp.NewPathPrefixStripper(len(path)),
	}
	fsHandler := fs.NewRequestHandler()

	handler := func(c *Context) {
		fsHandler(c.Ctx)
	}
	rg.GET(path+"/*filepath", handler)
	rg.GET(path, handler)
}

// Handler conforms to fasthttp.RequestHandler
func (engine *Engine) Handler(ctx *fasthttp.RequestCtx) {
	method := string(ctx.Method())
	path := string(ctx.Path())

	c := contextPool.Get().(*Context)
	c.reset(ctx, engine)

	node := engine.router.get(method, path, c)

	if node != nil {
		c.handlers = node.handlers
		c.Next()
	} else if method == fasthttp.MethodOptions {
		// Automatically handle CORS preflight if no specific OPTIONS route exists
		c.handlers = append([]HandlerFunc{}, engine.middlewares...)
		c.handlers = append(c.handlers, func(ctx *Context) {
			ctx.Ctx.SetStatusCode(fasthttp.StatusNoContent)
		})
		c.Next()
	} else {
		// Run global middlewares even for 404s (so Logger/CORS still fire)
		c.handlers = append([]HandlerFunc{}, engine.middlewares...)
		c.handlers = append(c.handlers, func(ctx *Context) {
			ctx.Ctx.Error("Not Found", fasthttp.StatusNotFound)
		})
		c.Next()
	}

	contextPool.Put(c)
}

// Run starts the fasthttp server and listens for OS signals for graceful shutdown.
func (engine *Engine) Run(addr string) error {
	engine.server = &fasthttp.Server{
		Handler: engine.Handler,
	}
	
	errCh := make(chan error, 1)
	go func() {
		errCh <- engine.server.ListenAndServe(addr)
	}()
	
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	
	select {
	case err := <-errCh:
		return err
	case sig := <-quit:
		log.Printf("Received signal: %v. Shutting down server...", sig)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return engine.Shutdown(ctx)
	}
}

// RunTLS starts an HTTPS server and listens for OS signals for graceful shutdown.
func (engine *Engine) RunTLS(addr, certFile, keyFile string) error {
	engine.server = &fasthttp.Server{
		Handler: engine.Handler,
	}
	
	errCh := make(chan error, 1)
	go func() {
		errCh <- engine.server.ListenAndServeTLS(addr, certFile, keyFile)
	}()
	
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	
	select {
	case err := <-errCh:
		return err
	case sig := <-quit:
		log.Printf("Received signal: %v. Shutting down server...", sig)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return engine.Shutdown(ctx)
	}
}

// Shutdown gracefully shuts down the server.
func (engine *Engine) Shutdown(ctx context.Context) error {
	if engine.server == nil {
		return nil
	}
	
	errCh := make(chan error, 1)
	go func() {
		errCh <- engine.server.Shutdown()
	}()
	
	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Test Run function to mock listener for testing
func (engine *Engine) Serve(ln net.Listener) error {
    engine.server = &fasthttp.Server{
		Handler: engine.Handler,
	}
	return engine.server.Serve(ln)
}
