<div align="center">
  <h1>🤫 Hush Framework</h1>
  <p><strong>A Next-Gen, Zero-Dependency, Typesafe Go Web Framework</strong></p>
</div>

---

**Hush** (formerly Glow) is an ultra-fast, entirely zero-dependency web framework built natively on Go 1.22+. It leverages modern Go Generics (`[T any]`) to eliminate the need for `interface{}` casting, providing a 100% Typesafe developer experience.

## ✨ Key Features

1. **Zero-Allocation Context:** Uses `sync.Pool` under the hood to recycle request contexts, ensuring virtually zero heap allocations per request.
2. **Generics-Based Binding:** Bind JSON bodies and URL queries directly to your strict typed structs via `hush.BindBody[T](c)`. No type-assertions needed.
3. **Reflection Validator:** Automatically enforces validation rules (like `validate:"required"`) based on struct tags.
4. **Typesafe Dependency Injection:** Built-in generic DI Container (`hush.Provide[T]` and `hush.Inject[T]`). Say goodbye to complex global variables!
5. **OpenAPI & Swagger UI:** Auto-generates OpenAPI 3.0 schema and serves a built-in Swagger UI at `/docs`.
6. **Middleware & Security:** Comes pre-packaged with zero-dependency `Helmet()`, `CORS()`, `RequestID()`, and `BasicAuth()` middlewares.
7. **Production Ready:** Native support for Graceful Shutdown and HTTP/2 (TLS).

## 🚀 Quick Start

### Installation
```bash
go get github.com/bdrtr/hush
```

### Basic Example
```go
package main

import (
	"log"
	"github.com/bdrtr/hush"
)

type UserRequest struct {
	Name  string `json:"name" validate:"required"`
	Email string `json:"email" validate:"required"`
}

func main() {
	app := hush.New()

	// Generic Body Binding & Validation
	app.POST("/users", func(c *hush.Context) {
		req, err := hush.BindBody[UserRequest](c)
		if err != nil {
			c.BadRequest(err.Error())
			return
		}
		
		c.Created(map[string]string{
			"message": "User " + req.Name + " created successfully!",
		})
	})

	log.Fatal(app.Run(":8080"))
}
```

## 📖 Documentation
Please see the [WIKI.md](./WIKI.md) for detailed documentation, advanced routing, dependency injection examples, and testing strategies.

## 📝 License
Written with 🦀 and 🐹. Open-source and free to use.
