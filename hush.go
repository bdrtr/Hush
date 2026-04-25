package hush

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"reflect"
	"runtime/debug"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/reuseport"
)


// Engine is the main framework instance.
type Engine struct {
	*RouterGroup
	mu        sync.RWMutex
	router    *Router
	container map[reflect.Type]interface{}
	server    *fasthttp.Server
	routes    []*Route
	config    *Config
}


// New creates a new Hush Engine based on fasthttp.
func New(opts ...Option) *Engine {
	cfg := DefaultConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	if cfg.SoftMemoryLimit > 0 {
		debug.SetMemoryLimit(cfg.SoftMemoryLimit)
	}

	engine := &Engine{
		router:    newRouter(),
		container: make(map[reflect.Type]interface{}),
		config:    cfg,
	}
	engine.RouterGroup = &RouterGroup{
		engine: engine,
	}
	return engine
}

// Provide registers a singleton dependency.
func Provide[T any](e *Engine, instance T) {
	e.mu.Lock()
	defer e.mu.Unlock()
	typ := reflect.TypeOf((*T)(nil)).Elem()
	e.container[typ] = instance
}


// b2s converts a byte slice to string without allocation.
// WARNING: The returned string is only valid for the duration of the request.
// Do NOT store or pass to goroutines — fasthttp reuses the underlying buffer.
func b2s(b []byte) string {
	if len(b) == 0 {
		return ""
	}
	return unsafe.String(unsafe.SliceData(b), len(b))
}

// Handler conforms to fasthttp.RequestHandler
func (engine *Engine) Handler(ctx *fasthttp.RequestCtx) {
	method := b2s(ctx.Method())
	path := b2s(ctx.Path())

	c := contextPool.Get().(*Context)
	defer contextPool.Put(c)
	
	c.reset(ctx, engine)

	engine.mu.RLock()
	node := engine.router.get(method, path, c)
	mws := append([]HandlerFunc{}, engine.middlewares...)
	engine.mu.RUnlock()

	if node != nil {
		c.handlers = node.handlers
		c.Next()
	} else if method == fasthttp.MethodOptions {
		// Automatically handle CORS preflight if no specific OPTIONS route exists.
		// Note: If CORS middleware is already registered globally, it will intercept
		// and abort the request before reaching the 204 handler below.
		c.handlers = append(mws, func(ctx *Context) {
			ctx.Ctx.SetStatusCode(fasthttp.StatusNoContent)
		})
		c.Next()
	} else {
		// Run global middlewares even for 404s (so Logger/CORS still fire)
		c.handlers = append(mws, func(ctx *Context) {
			ctx.Ctx.Error("Not Found", fasthttp.StatusNotFound)
		})
		c.Next()
	}
}

// PrintRoutes logs all registered routes to the terminal if debug mode is enabled.
func (engine *Engine) PrintRoutes() {
	if !engine.config.Debug {
		return
	}
	for _, route := range engine.routes {
		log.Printf("[HUSH] %-7s %s", route.Method, route.Path)
	}
}

// applyConfig applies the framework configuration to the fasthttp.Server instance.
func (engine *Engine) applyConfig() {
	engine.mu.Lock()
	defer engine.mu.Unlock()
	
	if engine.server == nil {
		engine.server = &fasthttp.Server{
			Handler: engine.Handler,
		}
	}
	engine.server.MaxRequestBodySize = engine.config.MaxRequestBodySize
	engine.server.ReadTimeout = engine.config.ReadTimeout
	engine.server.WriteTimeout = engine.config.WriteTimeout
	engine.server.IdleTimeout = engine.config.IdleTimeout
	engine.server.Concurrency = engine.config.Concurrency
	engine.server.ReduceMemoryUsage = engine.config.ReduceMemoryUsage
	engine.server.Logger = engine.config.Logger
}

// Run starts the fasthttp server and listens for OS signals for graceful shutdown.
func (engine *Engine) Run(addr string) error {
	engine.PrintRoutes()
	engine.applyConfig()
	
	errCh := make(chan error, 1)
	go func() {
		errCh <- engine.server.ListenAndServe(addr)
	}()
	
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(quit)
	
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

// RunPrefork starts the server using SO_REUSEPORT, allowing multiple processes to bind to the same port.
func (engine *Engine) RunPrefork(addr string) error {
	engine.PrintRoutes()
	ln, err := reuseport.Listen("tcp4", addr)
	if err != nil {
		return err
	}
	
	defer func() {
		if r := recover(); r != nil {
			ln.Close()
			panic(r)
		}
	}()
	
	engine.applyConfig()
	
	errCh := make(chan error, 1)
	go func() {
		errCh <- engine.server.Serve(ln)
	}()
	
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(quit)
	
	select {
	case err := <-errCh:
		return err
	case sig := <-quit:
		log.Printf("Received signal: %v. Shutting down prefork server...", sig)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return engine.Shutdown(ctx)
	}
}

// RunTLS starts an HTTPS server and listens for OS signals for graceful shutdown.
func (engine *Engine) RunTLS(addr, certFile, keyFile string) error {
	engine.PrintRoutes()
	engine.applyConfig()
	
	errCh := make(chan error, 1)
	go func() {
		errCh <- engine.server.ListenAndServeTLS(addr, certFile, keyFile)
	}()
	
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(quit)
	
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
	engine.mu.RLock()
	server := engine.server
	engine.mu.RUnlock()

	if server == nil {
		return nil
	}
	
	errCh := make(chan error, 1)
	go func() {
		errCh <- server.Shutdown()
	}()
	
	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		log.Printf("Shutdown timeout reached. Exiting gracefully based on timeout...")
		return ctx.Err()
	}
}

// Serve is used to serve on a custom listener.
// Note: This method does NOT include OS signal handling or graceful shutdown.
// For production use with graceful shutdown, prefer Run or RunTLS.
func (engine *Engine) Serve(ln net.Listener) error {
	engine.PrintRoutes()
    engine.applyConfig()
	return engine.server.Serve(ln)
}
