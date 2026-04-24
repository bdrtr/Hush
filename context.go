package hush

import (
	"encoding/json"
	"net/http"
	"sync"
)

// HandlerFunc is the type for Hush framework handlers.
type HandlerFunc func(*Context)

// responseWriter wraps http.ResponseWriter to capture the status code.
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// Context holds the request, response, and routing parameters.
// It is designed to be pooled to achieve zero-allocation per request.
type Context struct {
	Request  *http.Request
	Writer   http.ResponseWriter
	rw       responseWriter // Embedded wrapper
	Params   map[string]string // URL parameters
	Keys     map[string]interface{} // Key-value store for middlewares
	engine   *Engine                // Reference to the main engine for DI
	handlers []HandlerFunc
	index    int
}

// sync.Pool for Context
var contextPool = sync.Pool{
	New: func() interface{} {
		return &Context{
			Params: make(map[string]string),
			Keys:   make(map[string]interface{}),
		}
	},
}

// reset re-initializes the Context for a new request.
func (c *Context) reset(w http.ResponseWriter, r *http.Request, engine *Engine) {
	c.rw.ResponseWriter = w
	c.rw.statusCode = http.StatusOK // Default
	c.Writer = &c.rw
	c.Request = r
	c.engine = engine
	c.index = -1
	c.handlers = nil
	// Reset maps without allocating new ones
	for k := range c.Params {
		delete(c.Params, k)
	}
	for k := range c.Keys {
		delete(c.Keys, k)
	}
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
	c.Writer.WriteHeader(code)
	c.Abort()
}

// AbortWithJSON calls Abort and writes the JSON response.
func (c *Context) AbortWithJSON(code int, obj interface{}) {
	c.Abort()
	c.JSON(code, obj)
}

// Set stores a value for this request.
func (c *Context) Set(key string, value interface{}) {
	c.Keys[key] = value
}

// Get returns the value for the given key.
func (c *Context) Get(key string) (interface{}, bool) {
	val, ok := c.Keys[key]
	return val, ok
}

// Inject resolves a dependency from the Engine's DI container.
func Inject[T any](c *Context) T {
	var zero T
	if c.engine == nil || c.engine.container == nil {
		return zero
	}
	
	typ := reflect.TypeOf((*T)(nil)).Elem()
	if instance, ok := c.engine.container[typ]; ok {
		if typedInstance, ok := instance.(T); ok {
			return typedInstance
		}
	}
	return zero
}

// Param returns the value of the URL parameter.
func (c *Context) Param(key string) string {
	return c.Params[key]
}

// JSON sends a JSON response with the given status code.
func (c *Context) JSON(code int, obj interface{}) {
	c.Writer.Header().Set("Content-Type", "application/json")
	c.Writer.WriteHeader(code)
	encoder := json.NewEncoder(c.Writer)
	if err := encoder.Encode(obj); err != nil {
		http.Error(c.Writer, err.Error(), http.StatusInternalServerError)
	}
}

// Ok sends a 200 OK JSON response.
func (c *Context) Ok(obj interface{}) {
	c.JSON(http.StatusOK, obj)
}

// Created sends a 201 Created JSON response.
func (c *Context) Created(obj interface{}) {
	c.JSON(http.StatusCreated, obj)
}

// BadRequest sends a 400 Bad Request JSON error.
func (c *Context) BadRequest(err string) {
	c.JSON(http.StatusBadRequest, map[string]string{"error": err})
}

// NotFound sends a 404 Not Found JSON error.
func (c *Context) NotFound(err string) {
	c.JSON(http.StatusNotFound, map[string]string{"error": err})
}

// File serves a static file to the client.
func (c *Context) File(filepath string) {
	http.ServeFile(c.Writer, c.Request, filepath)
}

// Stream sends a streaming response.
func (c *Context) Stream(contentType string, reader func(w http.ResponseWriter) bool) {
	c.Writer.Header().Set("Content-Type", contentType)
	c.Writer.WriteHeader(http.StatusOK)
	for reader(c.Writer) {
		if f, ok := c.Writer.(http.Flusher); ok {
			f.Flush()
		}
	}
}
