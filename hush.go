package hush

import (
	"context"
	"net/http"
	"reflect"
)

// Engine is the main framework instance.
type Engine struct {
	*RouterGroup
	router    *Router
	container map[reflect.Type]interface{}
	server    *http.Server
}

// RouterGroup is used to group routes with prefixes and middlewares.
type RouterGroup struct {
	prefix     string
	middlewares []HandlerFunc
	engine     *Engine
}

// New creates a new Hush Engine.
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
		prefix:     rg.prefix + prefix,
		middlewares: append([]HandlerFunc{}, rg.middlewares...),
		engine:     rg.engine,
	}
}

// addRoute handles the actual route registration.
func (rg *RouterGroup) addRoute(method, comp string, handlers []HandlerFunc) {
	path := rg.prefix + comp
	// Combine group middlewares with route handlers
	finalHandlers := append([]HandlerFunc{}, rg.middlewares...)
	finalHandlers = append(finalHandlers, handlers...)
	
	rg.engine.router.insert(method, path, finalHandlers)
}

// GET registers a GET route.
func (rg *RouterGroup) GET(path string, handlers ...HandlerFunc) {
	rg.addRoute("GET", path, handlers)
}

// POST registers a POST route.
func (rg *RouterGroup) POST(path string, handlers ...HandlerFunc) {
	rg.addRoute("POST", path, handlers)
}

// Static serves static files from the given root directory.
func (rg *RouterGroup) Static(path, root string) {
	handler := func(c *Context) {
		filepath := c.Param("filepath")
		if filepath == "" {
			filepath = "/"
		}
		http.ServeFile(c.Writer, c.Request, root+filepath)
	}
	// Note: Our simple router doesn't fully support wildcard tail yet,
	// so we use a special wildcard param syntax for phase 1.
	rg.GET(path+"/:filepath", handler)
	rg.GET(path, handler) // For the root index file
}

// ServeHTTP conforms to the http.Handler interface.
func (engine *Engine) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	node, params := engine.router.get(r.Method, r.URL.Path)
	
	if node != nil {
		c := contextPool.Get().(*Context)
		c.reset(w, r, engine)
		
		for k, v := range params {
			c.Params[k] = v
		}
		
		c.handlers = node.handlers
		c.Next()
		
		contextPool.Put(c)
	} else {
		http.NotFound(w, r)
	}
}

// Run starts the HTTP server.
func (engine *Engine) Run(addr string) error {
	engine.server = &http.Server{
		Addr:    addr,
		Handler: engine,
	}
	return engine.server.ListenAndServe()
}

// RunTLS starts an HTTPS server.
func (engine *Engine) RunTLS(addr, certFile, keyFile string) error {
	engine.server = &http.Server{
		Addr:    addr,
		Handler: engine,
	}
	return engine.server.ListenAndServeTLS(certFile, keyFile)
}

// Shutdown gracefully shuts down the server without interrupting any active connections.
func (engine *Engine) Shutdown(ctx context.Context) error {
	if engine.server != nil {
		return engine.server.Shutdown(ctx)
	}
	return nil
}
