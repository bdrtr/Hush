package hush

import (
	"reflect"

	"github.com/goccy/go-json"
	"github.com/valyala/fasthttp"
)

// HandlerFunc is the type for Hush framework handlers.
type HandlerFunc func(*Context)

// Param represents a single URL parameter.
type Param struct {
	Key   string
	Value string
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
	c.handlers = nil
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
	if instance, ok := c.engine.container[typ]; ok {
		if typedInstance, ok := instance.(T); ok {
			return typedInstance
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

// JSON sends a JSON response with the given status code using fast goccy/go-json.
func (c *Context) JSON(code int, obj interface{}) {
	c.Ctx.SetContentType("application/json")
	c.Ctx.SetStatusCode(code)
	
	encoder := json.NewEncoder(c.Ctx)
	if err := encoder.Encode(obj); err != nil {
		c.Ctx.Error(err.Error(), fasthttp.StatusInternalServerError)
	}
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
