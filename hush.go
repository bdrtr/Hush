package hush

import (
	"context"
	"net"
	"reflect"

	"github.com/valyala/fasthttp"
)

// Engine is the main framework instance.
type Engine struct {
	*RouterGroup
	router    *Router
	container map[reflect.Type]interface{}
	server    *fasthttp.Server
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
func (rg *RouterGroup) addRoute(method, comp string, handlers []HandlerFunc) {
	path := rg.prefix + comp
	finalHandlers := append([]HandlerFunc{}, rg.middlewares...)
	finalHandlers = append(finalHandlers, handlers...)
	
	rg.engine.router.insert(method, path, finalHandlers)
}

// GET registers a GET route.
func (rg *RouterGroup) GET(path string, handlers ...HandlerFunc) {
	rg.addRoute(fasthttp.MethodGet, path, handlers)
}

// POST registers a POST route.
func (rg *RouterGroup) POST(path string, handlers ...HandlerFunc) {
	rg.addRoute(fasthttp.MethodPost, path, handlers)
}

// Static serves static files from the given root directory.
func (rg *RouterGroup) Static(path, root string) {
	handler := func(c *Context) {
		filepath := c.Param("filepath")
		if filepath == "" {
			filepath = "/"
		}
		fasthttp.ServeFile(c.Ctx, root+filepath)
	}
	rg.GET(path+"/:filepath", handler)
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
	} else {
		ctx.Error("Not Found", fasthttp.StatusNotFound)
	}

	contextPool.Put(c)
}

// Run starts the fasthttp server.
func (engine *Engine) Run(addr string) error {
	engine.server = &fasthttp.Server{
		Handler: engine.Handler,
	}
	return engine.server.ListenAndServe(addr)
}

// RunTLS starts an HTTPS server.
func (engine *Engine) RunTLS(addr, certFile, keyFile string) error {
	engine.server = &fasthttp.Server{
		Handler: engine.Handler,
	}
	return engine.server.ListenAndServeTLS(addr, certFile, keyFile)
}

// Shutdown gracefully shuts down the server.
func (engine *Engine) Shutdown(ctx context.Context) error {
	if engine.server != nil {
		return engine.server.Shutdown()
	}
	return nil
}

// Test Run function to mock listener for testing
func (engine *Engine) Serve(ln net.Listener) error {
    engine.server = &fasthttp.Server{
		Handler: engine.Handler,
	}
	return engine.server.Serve(ln)
}
