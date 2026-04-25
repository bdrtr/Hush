<div align="center">
  <img src="hush_logo_product.png" width="200" height="200" alt="Hush Logo">
  <h1>🤫 Hush Framework</h1>
  <p><strong>A Next-Gen, High-Performance, Typesafe Go Web Framework</strong></p>
</div>

---

**Hush** (formerly Glow) is an ultra-fast, zero-allocation web framework built natively on Go 1.22+. Powered by the lightning-fast `valyala/fasthttp` under the hood and leveraging `goccy/go-json`, it is designed to achieve maximum RPS (Requests Per Second). Furthermore, it leverages modern Go Generics (`[T any]`) to provide a 100% Typesafe developer experience without the `interface{}` overhead.

## ✨ Key Features

1. **Extreme Performance:** Built on top of `fasthttp`.
2. **Zero-Allocation Routing:** Uses fixed-size arrays (`[10]Param`) and pointer-based path matching to ensure 0 bytes memory allocation during URL parameter parsing.
3. **Generics-Based Binding:** Bind JSON bodies and URL queries directly to your strict typed structs via `hush.BindBody[T](c)`. Powered by `goccy/go-json` for 300% faster JSON encoding/decoding.
4. **Typesafe Dependency Injection:** Built-in generic DI Container (`hush.Provide[T]` and `hush.Inject[T]`).

6. **OpenAPI & Swagger UI:** Auto-generates OpenAPI 3.0 schema and serves a built-in Swagger UI at `/docs`.
7. **Middleware & Security:** Comes pre-packaged with `Helmet()`, `CORS()`, and `RequestID()` middlewares.

## ⚡ Performance Benchmarks

Hush is engineered to push Go to its absolute physical limits using SIMD instructions, O(1) routing trees, and Zero-Allocation patterns.

### 1. Load Testing (Concurrent Throughput)
Tested with 1,000 concurrent workers against a static route (`hey -n 1000000 -c 1000 http://localhost:8080/`).

| Metric | Result |
| :--- | :--- |
| **Requests/sec (RPS)** | **283,348** |
| **P50 Latency** | 0.2 ms |
| **P99 Latency** | 26 ms |
| **Errors** | 0 |

### 2. Micro-Benchmarks (Routing & Memory)
Tested using `go test -bench=. -benchmem` running on an AMD Ryzen 7 8845HS processor.

| Operation | Speed (ns/op) | Memory Allocated | Allocs/op |
| :--- | :--- | :--- | :--- |
| **O(1) Static Route** | **25.96 ns** | **0 B/op** | **0** |
| **Param Route (`:id`)** | 39.47 ns | **0 B/op** | **0** |
| **Wildcard Route (`*path`)**| 42.50 ns | **0 B/op** | **0** |
| **JSON Serialization** | 386.80 ns | 661 B/op | 6 |

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
