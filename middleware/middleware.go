package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/bdrtr/hush"
	"github.com/golang-jwt/jwt/v5"
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

type clientData struct {
	count     int
	windowEnd time.Time
}

// RateLimit implements a per-IP fixed-window rate limiter without leaking goroutines.
func RateLimit(limit int, window time.Duration) hush.HandlerFunc {
	var mu sync.Mutex
	clients := make(map[string]*clientData)
	
	return func(c *hush.Context) {
		ip := string(c.Ctx.Request.Header.Peek("X-Forwarded-For"))
		if ip != "" {
			if commaIdx := strings.IndexByte(ip, ','); commaIdx != -1 {
				ip = ip[:commaIdx]
			}
		} else {
			ip = c.Ctx.RemoteIP().String()
		}
		
		mu.Lock()
		now := time.Now()
		
		// Lazy sweep to prevent OOM
		if len(clients) > 10000 {
			for k, v := range clients {
				if now.After(v.windowEnd) {
					delete(clients, k)
				}
			}
		}

		data, exists := clients[ip]
		if !exists || now.After(data.windowEnd) {
			clients[ip] = &clientData{
				count:     1,
				windowEnd: now.Add(window),
			}
			mu.Unlock()
		} else {
			data.count++
			count := data.count
			mu.Unlock()
			
			if count > limit {
				c.Ctx.Error("Too Many Requests", fasthttp.StatusTooManyRequests)
				c.Abort()
				return
			}
		}
		c.Next()
	}
}

// JWT checks for a Bearer token in Authorization header and verifies its signature.
func JWT(secret string) hush.HandlerFunc {
	return func(c *hush.Context) {
		auth := string(c.Ctx.Request.Header.Peek("Authorization"))
		if auth == "" || len(auth) < 7 || auth[:7] != "Bearer " {
			c.Ctx.Error("Missing or invalid token", fasthttp.StatusUnauthorized)
			c.Abort()
			return
		}
		
		tokenString := auth[7:]
		
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(secret), nil
		})
		
		if err != nil || !token.Valid {
			c.Ctx.Error("Unauthorized: Invalid token signature", fasthttp.StatusUnauthorized)
			c.Abort()
			return
		}

		if claims, ok := token.Claims.(jwt.MapClaims); ok {
			c.Set("jwt_claims", claims)
		}
		
		c.Set("jwt_token", tokenString)
		c.Next()
	}
}

// Timeout sets a strict deadline for the request.
// It uses fasthttp's native SetDeadline which safely aborts network connections
// if the request exceeds the duration, avoiding goroutine leaks.
func Timeout(timeout time.Duration) hush.HandlerFunc {
	return func(c *hush.Context) {
		c.Ctx.SetDeadline(time.Now().Add(timeout))
		c.Next()
	}
}
