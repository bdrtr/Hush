# 📖 Hush Framework Wiki

Welcome to the comprehensive guide for the **Hush Framework**! This wiki will teach you everything from basic routing to extreme performance optimizations.

---

## 📑 Table of Contents
1. [Routing & Groups](#1-routing--groups)
2. [Context & Data Binding](#2-context--data-binding)
3. [Middleware & Observability](#3-middleware--observability)
4. [Dependency Injection (DI)](#4-dependency-injection-di)
5. [OpenAPI & Swagger](#5-openapi--swagger)
6. [Testing Utilities](#6-testing-utilities)
7. [Extreme Performance Tuning](#7-extreme-performance-tuning)

---

## 1. Routing & Groups

Hush uses a custom Radix Tree based router that is extremely strict and zero-allocation.

### Basic Routes
```go
app := hush.New()

app.GET("/hello", func(c *hush.Context) {
    c.Ok(map[string]string{"message": "World"})
})
```

### Dynamic Parameters
URL parameters are parsed with **0 byte allocations** using fixed-size arrays.
```go
app.GET("/users/:id", func(c *hush.Context) {
    id := c.Param("id") // Access without allocations
    c.Ok(map[string]string{"user_id": id})
})
```

### Wildcards (Catch-All)
Perfect for serving dynamic files or SPA routing.
```go
app.GET("/assets/*filepath", func(c *hush.Context) {
    filepath := c.Param("filepath")
    c.File("./public/" + filepath)
})
```

### Router Groups
Groups allow you to isolate routes and apply middlewares to specific sub-trees safely.
```go
api := app.Group("/api/v1")
api.Use(middleware.Auth()) // Applies only to /api/v1/*

api.GET("/status", func(c *hush.Context) { ... })
```

### Static Files
Hush provides a highly optimized static file server backed by `fasthttp.FS`.
```go
app.Static("/uploads", "./storage/uploads")
```

---

## 2. Context & Data Binding

We leverage **Go Generics** and `bytedance/sonic` to provide a 100% Typesafe data binding experience without `interface{}` casting or reflection penalties.

### JSON Body Binding & Validation
Hush integrates `go-playground/validator/v10` automatically.
```go
type CreateUserReq struct {
    Email string `json:"email" validate:"required,email"`
    Age   int    `json:"age" validate:"gte=18"`
}

app.POST("/users", func(c *hush.Context) {
    // req is strictly typed as *CreateUserReq
    req, err := hush.BindBody[CreateUserReq](c)
    if err != nil {
        // Validation errors and 400 Bad Request are handled automatically by BindBody
        return 
    }
    
    // Fully typesafe access!
    c.Created(map[string]interface{}{"email": req.Email})
})
```

### Query Parameter Binding
```go
type Pagination struct {
    Page  int `query:"page" validate:"min=1"`
    Limit int `query:"limit" validate:"min=10,max=100"`
}

app.GET("/posts", func(c *hush.Context) {
    query, err := hush.BindQuery[Pagination](c)
    if err != nil { return }
    
    // query.Page is an int!
})
```

### WebSockets & Server-Sent Events (SSE)
Hush makes streaming protocols simple.

**Server-Sent Events (SSE):**
```go
app.GET("/stream", func(c *hush.Context) {
    c.SSE(func(w *bufio.Writer) error {
        for {
            w.WriteString("data: update\n\n")
            w.Flush()
            time.Sleep(1 * time.Second)
        }
    })
})
```

**WebSockets:**
```go
app.GET("/ws", func(c *hush.Context) {
    err := c.Upgrade([]string{"*"}, func(conn *websocket.Conn) {
        defer conn.Close()
        conn.WriteMessage(websocket.TextMessage, []byte("Hello WS!"))
    })
})
```

---

## 3. Middleware & Observability

Middleware allows you to intercept requests, run security checks, or pass state down the chain. Hush provides a powerful built-in ecosystem.

> [!TIP]
> **Middleware Ordering Matters:** Always place `Recovery()` first, followed by `RequestID()` and `Logger()`.

### Built-in Middlewares
```go
import "github.com/bdrtr/hush/middleware"

app := hush.New()

// 1. Recovery catches panics and returns 500 cleanly
app.Use(middleware.Recovery())

// 2. Generates unique X-Request-ID for tracing
app.Use(middleware.RequestID())

// 3. Structured Logging (now includes request_id!)
app.Use(middleware.Logger())

// 4. Zero-allocation atomic metrics tracking
app.Use(middleware.Stats())

// 5. Security headers (replaces manual setups)
app.Use(middleware.Helmet())

// 6. JWT Authentication
app.Use(middleware.JWT("my-secret-key"))

// 7. Rate Limiter (Token bucket, Per-IP)
app.Use(middleware.RateLimit(100, time.Minute))

// 8. OOM-Safe Caching (Cache GET requests for 5 mins, max 1000 routes)
app.Use(middleware.Cache(5 * time.Minute, 1000, 5*1024*1024))
```

### Passing State
Middlewares can pass state downstream via `c.Set()`.
```go
func AuthMiddleware() hush.HandlerFunc {
    return func(c *hush.Context) {
        c.Set("userID", 42) // Set state
        c.Next()            // Proceed to handler
    }
}

app.GET("/me", func(c *hush.Context) {
    id, _ := c.Get("userID") // Get state
    c.Ok(map[string]interface{}{"id": id})
})
```

### Exposing Metrics
You can expose the live `Stats()` metrics via an endpoint:
```go
app.GET("/api/stats", func(c *hush.Context) {
    c.Ok(middleware.GetStats()) 
})
```

---

## 4. Dependency Injection (DI)

Instead of passing database connections or services via global variables or complex struct initializations, Hush provides a built-in, generic, typesafe DI container.

### 1. Provide (Startup)
```go
type DBService interface { GetUser() string }
type Postgres struct{}
func (p *Postgres) GetUser() string { return "Alice" }

func main() {
    app := hush.New()
    var db DBService = &Postgres{}
    
    // Inject the service into the container
    hush.Provide[DBService](app, db) 
}
```

### 2. Inject (Runtime)
Retrieve your services safely inside any route handler.
```go
app.GET("/user", func(c *hush.Context) {
    db := hush.Inject[DBService](c)
    if db != nil {
        c.Ok(db.GetUser())
    }
})
```

---

## 5. OpenAPI & Swagger

Hush automatically generates an OpenAPI 3.0 specification from your generic routes! 

### Documenting Routes
Use the `With*` fluent methods to attach metadata. Because of Go generics limitations, Body/Query types use a wrapper function.

```go
hush.WithBody[CreateUserReq](
    hush.WithResponse[UserResponse](
        app.POST("/users", CreateUserHandler).
            WithSummary("Creates a new user in the system").
            WithTags("Users", "Auth"),
    ),
)
```

### Serving the UI
Attach the Swagger UI to a path. Hush uses `sync.Once` to cache the schema generation instantly.
```go
app.ServeSwaggerUI("/docs")
```
Navigate to `/docs` to see your beautifully documented, interactive API!

---

## 6. Testing Utilities

Because Hush uses `valyala/fasthttp` rather than standard `net/http`, testing requires mock context generation. Hush makes this trivial.

```go
func TestUserRoute(t *testing.T) {
    // 1. Create a mock context (returns Context and Cleanup function)
    c, cleanup := hush.NewTestContext("POST", "/users")
    defer cleanup() // CRITICAL: Always defer cleanup to return context to the memory pool!
    
    // 2. Set mock JSON body
    c.Ctx.Request.Header.SetContentType("application/json")
    c.Ctx.Request.SetBody([]byte(`{"email": "test@test.com", "age": 25}`))
    
    // 3. Call your handler directly
    UserHandler(c)
    
    // 4. Assert
    if c.Ctx.Response.StatusCode() != 201 {
        t.Errorf("Expected 201, got %d", c.Ctx.Response.StatusCode())
    }
}
```

---

## 7. Extreme Performance Tuning

Hush is designed to bypass standard Go bottlenecks. 

### O(1) Static Routing
You don't need to do anything to enable this. Hush automatically detects routes without parameters (`:id` or `*path`) and registers them in an O(1) `cespare/xxhash/v2` map. This completely bypasses the Radix tree, matching static paths in ~25ns with **0 Bytes** memory allocation.

### GOMEMLIMIT (Soft Memory Limit)
Set a soft memory limit to instruct the Go GC to stay asleep until memory usage approaches your defined limit. By keeping the GC asleep, you eliminate P99 latency spikes and micro-pauses under high load.

```go
app := hush.New(
    // 512MB Soft Limit
    hush.WithSoftMemoryLimit(512 * 1024 * 1024),
    
    // Aggressive Concurrency
    hush.WithConcurrency(256 * 1024),
)
```
> [!WARNING]
> Ensure your physical server or container has enough RAM to comfortably hold this limit. If your application exceeds physical RAM, the OS OOM killer will terminate the process!

### Prefork (SO_REUSEPORT)
In production, run Hush in Prefork mode to utilize all CPU cores effectively by allowing multiple processes to bind to the same port.
```go
// Starts the server utilizing SO_REUSEPORT
app.RunPrefork(":8080")
```
