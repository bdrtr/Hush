<div align="center">
  <img src="hush_logo_product.png" width="200" height="200" alt="Hush Logo">
  <h1>🤫 Hush Framework</h1>
  <p><strong>A Next-Gen, Ultra-Fast, Typesafe Go Web Framework</strong></p>

  [![Go Reference](https://pkg.go.dev/badge/github.com/bdrtr/hush.svg)](https://pkg.go.dev/github.com/bdrtr/hush)
  [![Go Version](https://img.shields.io/badge/go-1.23+-blue.svg)](https://golang.org)
  [![License](https://img.shields.io/badge/License-MIT-green.svg)](https://opensource.org/licenses/MIT)
</div>

---

**Hush** is an uncompromisingly fast, zero-allocation web framework built natively on Go 1.23+. Powered by the lightning-fast `valyala/fasthttp` and `bytedance/sonic` (SIMD JSON) under the hood, it is engineered to achieve absolute maximum Requests Per Second (RPS). 

It abandons `interface{}` overhead completely, leveraging **Go Generics** (`[T any]`) to provide a 100% Typesafe developer experience.

## ✨ Why Hush?

- 🚀 **Extreme Performance:** Built on `fasthttp`, making it up to 10x faster than `net/http`.
- 🧠 **Zero-Allocation Routing:** Uses custom Radix Trees, fixed-size parameter arrays (`[10]Param`), and O(1) Hash Map lookups for static routes resulting in **0 Bytes** memory allocation.
- 🛡️ **Bulletproof Security:** Ships with built-in `Helmet`, `CORS`, `JWT`, `RateLimit`, and automatic XSS protections.
- 🧩 **Typesafe Generics:** Bind JSON and Queries directly to structs with `hush.BindBody[T](c)`. No reflection penalties, no `interface{}` casting.
- 📊 **Built-in Observability:** Zero-allocation `sync/atomic` metrics (`Stats()`) and contextual structured logging (`Logger()`, `RequestID()`).
- 📘 **Auto OpenAPI 3.0:** Generates Swagger documentation on-the-fly and serves the UI out of the box.

## ⚡ Performance

Hush pushes Go to its physical limits using SIMD instructions, strict data race prevention, and GC-friendly memory pooling.

### Micro-Benchmarks (Routing & Memory)
Tested on an AMD Ryzen 7 8845HS.

| Operation | Speed (ns/op) | Memory Allocated | Allocs/op |
| :--- | :--- | :--- | :--- |
| **O(1) Static Route** | **25.96 ns** | **0 B/op** | **0** |
| **Param Route (`:id`)** | 39.47 ns | **0 B/op** | **0** |
| **Wildcard Route (`*path`)**| 42.50 ns | **0 B/op** | **0** |

## 🚀 Quick Start

### Installation
```bash
go get github.com/bdrtr/hush
```

### Hello World (with Typesafe Binding & Swagger)
```go
package main

import (
	"log"
	"github.com/bdrtr/hush"
)

type UserRequest struct {
	Name  string `json:"name" validate:"required"`
	Email string `json:"email" validate:"required,email"`
}

func main() {
	app := hush.New()

	// 1. Add Observability
	app.Use(hush.Logger())

	// 2. Typesafe Route
	hush.WithBody[UserRequest](
		app.POST("/users", func(c *hush.Context) {
			// Generics-based JSON Binding + Validation
			req, err := hush.BindBody[UserRequest](c)
			if err != nil {
				return // BindBody handles the 400 Bad Request response automatically
			}
			
			c.Created(map[string]string{
				"message": "Welcome, " + req.Name + "!",
			})
		}).WithSummary("Create a new user"),
	)

	// 3. Serve Auto-Generated Swagger UI
	app.ServeSwaggerUI("/docs")

	// 4. Start Server
	log.Println("Server running on http://localhost:8080")
	log.Fatal(app.Run(":8080"))
}
```

## 📖 Documentation

For advanced routing, dependency injection, middleware crafting, and performance tuning, please read the [Official Wiki](./WIKI.md).

## 📝 License
Written with 🦀 and 🐹. Open-source and free to use.
