package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"log"
	"sync"
	"time"

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

// Logger logs the details of each request including method, path, status, and duration.
func Logger() hush.HandlerFunc {
	return func(c *hush.Context) {
		start := time.Now()
		c.Next()
		duration := time.Since(start)
		
		log.Printf("[%s] %s | Status: %d | %v", c.Ctx.Method(), c.Ctx.Path(), c.Ctx.Response.StatusCode(), duration)
	}
}

// Recovery catches any panics during request handling and returns a 500 error gracefully.
func Recovery() hush.HandlerFunc {
	return func(c *hush.Context) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("[Recovery] panic recovered: %v", err)
				c.AbortWithStatus(fasthttp.StatusInternalServerError)
			}
		}()
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
		c.Ctx.Response.Header.Set("X-Request-ID", reqID)
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

// RateLimit implements a simple fixed-window rate limiter.
func RateLimit(limit int, window time.Duration) hush.HandlerFunc {
	var mu sync.Mutex
	clients := make(map[string]int)
	
	go func() {
		for {
			time.Sleep(window)
			mu.Lock()
			clients = make(map[string]int)
			mu.Unlock()
		}
	}()
	
	return func(c *hush.Context) {
		ip := c.Ctx.RemoteIP().String()
		mu.Lock()
		clients[ip]++
		count := clients[ip]
		mu.Unlock()
		
		if count > limit {
			c.AbortWithJSON(fasthttp.StatusTooManyRequests, map[string]string{"error": "Rate limit exceeded"})
			return
		}
		c.Next()
	}
}

// JWT checks for a Bearer token in Authorization header.
// Real-world usage would verify the signature with a secret.
func JWT(secret string) hush.HandlerFunc {
	return func(c *hush.Context) {
		auth := string(c.Ctx.Request.Header.Peek("Authorization"))
		if auth == "" || len(auth) < 7 || auth[:7] != "Bearer " {
			c.AbortWithJSON(fasthttp.StatusUnauthorized, map[string]string{"error": "Missing or invalid token"})
			return
		}
		
		token := auth[7:]
		// Validation logic would go here
		c.Set("jwt_token", token)
		c.Next()
	}
}
