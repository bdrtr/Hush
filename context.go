package hush

import (
	"bufio"
	"mime/multipart"
	"reflect"
	"sync"

	"github.com/bytedance/sonic"
	"github.com/fasthttp/websocket"
	"github.com/valyala/fasthttp"
)

// HandlerFunc is the type for Hush framework handlers.
type HandlerFunc func(*Context)

// Param represents a single URL parameter.
type Param struct {
	Key   string
	Value string
}

var contextPool = sync.Pool{
	New: func() interface{} {
		return &Context{}
	},
}

// Context holds the fasthttp request context and routing parameters.
// It is designed to be pooled to achieve zero-allocation per request.
type Context struct {
	Ctx        *fasthttp.RequestCtx
	Params     [10]Param // Fixed array for zero-allocation params
	paramCount int
	engine     *Engine // Reference to the main engine for DI
	handlers   []HandlerFunc
	index      int
}

// reset re-initializes the Context for a new request.
func (c *Context) reset(ctx *fasthttp.RequestCtx, engine *Engine) {
	c.Ctx = ctx
	c.engine = engine
	c.index = -1
	c.handlers = c.handlers[:0] // Preserve capacity for zero-allocation
	c.paramCount = 0 // Just reset count, array stays in memory without alloc
}

// Next executes the next handler in the middleware chain.
func (c *Context) Next() {
	c.index++
	for c.index < len(c.handlers) {
		c.handlers[c.index](c)
		c.index++
	}
}

// Abort prevents pending handlers from being called.
func (c *Context) Abort() {
	c.index = len(c.handlers) // Skip all remaining handlers
}

// AbortWithStatus calls Abort and writes the given status code.
func (c *Context) AbortWithStatus(code int) {
	c.Ctx.SetStatusCode(code)
	c.Abort()
}

// AbortWithJSON calls Abort and writes the JSON response.
func (c *Context) AbortWithJSON(code int, obj interface{}) {
	c.Abort()
	c.JSON(code, obj)
}

// AbortWithError calls Abort and writes the error message as a JSON response.
func (c *Context) AbortWithError(code int, err error) {
	c.AbortWithJSON(code, map[string]string{"error": err.Error()})
}

// Set stores a value for this request inside fasthttp's native UserValue storage.
func (c *Context) Set(key string, value interface{}) {
	c.Ctx.SetUserValue(key, value)
}

// Get returns the value for the given key.
func (c *Context) Get(key string) (interface{}, bool) {
	val := c.Ctx.UserValue(key)
	return val, val != nil
}

// Inject resolves a dependency from the Engine's DI container.
func Inject[T any](c *Context) T {
	var zero T
	if c.engine == nil || c.engine.container == nil {
		return zero
	}

	typ := reflect.TypeOf((*T)(nil)).Elem()
	
	// Fast path: Exact match
	if instance, ok := c.engine.container[typ]; ok {
		if typedInstance, ok := instance.(T); ok {
			return typedInstance
		}
	}
	
	// Slow path: Check if any registered type implements the interface
	if typ.Kind() == reflect.Interface {
		for regTyp, instance := range c.engine.container {
			if regTyp.Implements(typ) {
				if typedInstance, ok := instance.(T); ok {
					return typedInstance
				}
			}
		}
	}
	
	return zero
}

// Param returns the value of the URL parameter.
func (c *Context) Param(key string) string {
	for i := 0; i < c.paramCount; i++ {
		if c.Params[i].Key == key {
			return c.Params[i].Value
		}
	}
	return ""
}

// addParam is an internal method to append parameters during routing without allocation.
func (c *Context) addParam(key, value string) {
	if c.paramCount < len(c.Params) {
		c.Params[c.paramCount].Key = key
		c.Params[c.paramCount].Value = value
		c.paramCount++
	}
}

// Query returns the value of the given URL query parameter.
func (c *Context) Query(key string) string {
	return string(c.Ctx.QueryArgs().Peek(key))
}

// FormValue returns the value of the given form field.
func (c *Context) FormValue(key string) string {
	return string(c.Ctx.FormValue(key))
}

// FormFile returns the uploaded file by key.
func (c *Context) FormFile(key string) (*multipart.FileHeader, error) {
	return c.Ctx.FormFile(key)
}

// SSE sets up Server-Sent Events and executes the streamer function.
// The streamer should return an error if a write fails (e.g. client disconnects) to gracefully terminate.
func (c *Context) SSE(streamer func(w *bufio.Writer) error) {
	c.Ctx.SetContentType("text/event-stream")
	c.Ctx.Response.Header.Set("Cache-Control", "no-cache")
	c.Ctx.Response.Header.Set("Connection", "keep-alive")
	
	c.Ctx.SetBodyStreamWriter(func(w *bufio.Writer) {
		_ = streamer(w)
	})
}

// Upgrade upgrades the HTTP connection to a WebSocket connection securely.
func (c *Context) Upgrade(allowedOrigins []string, handler func(conn *websocket.Conn)) error {
	u := websocket.FastHTTPUpgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(ctx *fasthttp.RequestCtx) bool {
			if len(allowedOrigins) == 0 {
				return false // Default secure: block if no origins provided
			}
			origin := string(ctx.Request.Header.Peek("Origin"))
			if origin == "" {
				// Non-browser clients (mobile, server-to-server) might not send Origin
				return true
			}
			for _, o := range allowedOrigins {
				if o == "*" || o == origin {
					return true
				}
			}
			return false
		},
	}
	
	err := u.Upgrade(c.Ctx, handler)
	if err != nil {
		c.Ctx.Error("WebSocket Upgrade Failed", fasthttp.StatusBadRequest)
	}
	return err
}

// JSON sends a JSON response with the given status code using Sonic SIMD JSON.
func (c *Context) JSON(code int, obj interface{}) {
	c.Ctx.SetContentType("application/json")
	c.Ctx.SetStatusCode(code)

	bytes, err := sonic.Marshal(obj)
	if err != nil {
		c.Ctx.Error("Internal Server Error", fasthttp.StatusInternalServerError)
		return
	}
	c.Ctx.Write(bytes)
}

// Ok sends a 200 OK JSON response.
func (c *Context) Ok(obj interface{}) {
	c.JSON(fasthttp.StatusOK, obj)
}

// Created sends a 201 Created JSON response.
func (c *Context) Created(obj interface{}) {
	c.JSON(fasthttp.StatusCreated, obj)
}

// BadRequest sends a 400 Bad Request JSON error.
func (c *Context) BadRequest(err string) {
	c.JSON(fasthttp.StatusBadRequest, map[string]string{"error": err})
}

// NotFound sends a 404 Not Found JSON error.
func (c *Context) NotFound(err string) {
	c.JSON(fasthttp.StatusNotFound, map[string]string{"error": err})
}

// File serves a static file to the client.
func (c *Context) File(filepath string) {
	fasthttp.ServeFile(c.Ctx, filepath)
}
