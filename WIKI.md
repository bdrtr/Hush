# 📖 Hush Framework Wiki

Welcome to the comprehensive guide for the Hush Framework!

## Table of Contents
1. [Routing & Groups](#routing--groups)
2. [Data Binding & Validation](#data-binding--validation)
3. [Middleware & State Management](#middleware--state-management)
4. [Dependency Injection (DI)](#dependency-injection-di)
5. [Testing Utilities](#testing-utilities)
6. [OpenAPI & Swagger](#openapi--swagger)
7. [Extreme Optimizations](#extreme-optimizations)

---

## Routing & Groups
Hush uses a custom Radix Tree based router. It supports static routes, dynamic parameters, and isolated router groups.

```go
app := hush.New()

// Basic Route
app.GET("/hello", func(c *hush.Context) {
    c.Ok(map[string]string{"msg": "World"})
})

// Dynamic Parameters
app.GET("/users/:id", func(c *hush.Context) {
    id := c.Param("id")
    c.Ok(map[string]string{"user_id": id})
})

// Route Groups
api := app.Group("/api/v1")
api.GET("/status", func(c *hush.Context) { ... })
```

---

## Data Binding & Validation
We use Go Generics to prevent runtime panics and interface casting.

### JSON Body
```go
type Request struct {
    Email string `json:"email" validate:"required"`
}

app.POST("/login", func(c *hush.Context) {
    req, err := hush.BindBody[Request](c) // req is *Request
    if err != nil {
        c.BadRequest(err.Error())
        return
    }
    // Access safely: req.Email
})
```

---

## Middleware & State Management
Middleware allows you to intercept requests, run security checks, or inject states.

### Aborting Requests
If a middleware fails (e.g., Auth), you must abort the chain:
```go
func Auth() hush.HandlerFunc {
    return func(c *hush.Context) {
        if c.Request.Header.Get("Token") != "123" {
            c.AbortWithJSON(401, map[string]string{"error": "Unauthorized"})
            return
        }
        c.Set("userID", 99) // Pass state
        c.Next()
    }
}
```

### Accessing State
In your final route, you can retrieve the state set by middlewares:
```go
app.GET("/profile", func(c *hush.Context) {
    userID, exists := c.Get("userID")
    // ...
})
```

---

## Dependency Injection (DI)
Instead of passing database connections via global variables, Hush has a built-in Typesafe DI container.

### 1. Provide (Startup)
Inject your dependencies when starting the server:
```go
type DB interface { GetUser() string }
type Postgres struct{}
func (p *Postgres) GetUser() string { return "Alice" }

func main() {
    app := hush.New()
    var db DB = &Postgres{}
    hush.Provide[DB](app, db) // Provide interface or struct
    

}
```

### 2. Inject (Runtime)
Retrieve it from any HTTP handler safely:
```go
app.GET("/user", func(c *hush.Context) {
    db := hush.Inject[DB](c)
    if db != nil {
        c.Ok(db.GetUser())
    }
})
```

---

## Testing Utilities
Hush makes unit testing handlers extremely easy.

### Unit Testing Handlers
Because Hush uses `valyala/fasthttp` under the hood, you test by mocking `*fasthttp.RequestCtx`.

```go
func TestUserRoute(t *testing.T) {
    c := hush.NewTestContext("GET", "/users/1")
    
    // Call handler directly
    UserHandler(c)
    
    if c.Ctx.Response.StatusCode() != 200 {
        t.Errorf("Expected 200, got %d", c.Ctx.Response.StatusCode())
    }
}
```

---

## OpenAPI & Swagger
Hush generates OpenAPI 3.0 specification automatically. To serve the Swagger UI, simply attach the endpoint:

```go
app.ServeSwaggerUI("/docs")
```
Navigate to `http://localhost:8080/docs` in your browser.

---

## Extreme Optimizations

Hush provides out-of-the-box features to push your application's performance to the absolute physical limits of your server by minimizing Garbage Collector (GC) pressure and bypassing traditional router tree traversals.

### 1. O(1) Static Routing (Zero Allocation)
You don't need to do anything to enable this. Hush automatically detects routes without parameters (`:id` or `*path`) and registers them in an O(1) hash map using `cespare/xxhash/v2`. This completely bypasses the Radix tree routing logic, matching static paths in ~25ns with exactly **0 Bytes** of memory allocation.

### 2. GOMEMLIMIT (Soft Memory Limit)
In Go 1.19+, you can set a soft memory limit. This instructs the Go GC to stay asleep until the memory usage approaches your defined limit. By keeping the GC asleep, you eliminate P99 latency spikes and micro-pauses under high concurrent load.

```go
func main() {
    // Tell Hush to configure a 512MB Soft Memory Limit
    app := hush.New(
        hush.WithSoftMemoryLimit(512 * 1024 * 1024),
    )
    
    // ...
}
```
**Warning:** Ensure your physical server or container has enough RAM to comfortably hold this limit. If your application exceeds the physical RAM, the operating system's OOM killer will terminate the process.
