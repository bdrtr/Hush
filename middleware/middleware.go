package middleware

import (
	"crypto/rand"
	"encoding/hex"

	"github.com/bdrtr/hush"
	"github.com/valyala/fasthttp"
)

// CORS returns a middleware that handles Cross-Origin Resource Sharing.
func CORS(allowOrigins string) hush.HandlerFunc {
	return func(c *hush.Context) {
		c.Ctx.Response.Header.Set("Access-Control-Allow-Origin", allowOrigins)
		c.Ctx.Response.Header.Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Ctx.Response.Header.Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")

		if string(c.Ctx.Method()) == fasthttp.MethodOptions {
			c.AbortWithStatus(fasthttp.StatusNoContent)
			return
		}
		
		c.Next()
	}
}

// RequestID generates a unique ID for each request.
func RequestID() hush.HandlerFunc {
	return func(c *hush.Context) {
		reqID := string(c.Ctx.Request.Header.Peek("X-Request-ID"))
		if reqID == "" {
			bytes := make([]byte, 16)
			if _, err := rand.Read(bytes); err == nil {
				reqID = hex.EncodeToString(bytes)
			} else {
				reqID = "unknown"
			}
		}
		
		// Set in header so client gets it back
		c.Ctx.Response.Header.Set("X-Request-ID", reqID)
		
		// Set in context so handlers can use it for logging
		c.Set("request_id", reqID)
		
		c.Next()
	}
}

// Helmet sets standard security headers.
func Helmet() hush.HandlerFunc {
	return func(c *hush.Context) {
		c.Ctx.Response.Header.Set("X-XSS-Protection", "1; mode=block")
		c.Ctx.Response.Header.Set("X-Content-Type-Options", "nosniff")
		c.Ctx.Response.Header.Set("X-Frame-Options", "DENY")
		c.Ctx.Response.Header.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		
		c.Next()
	}
}
