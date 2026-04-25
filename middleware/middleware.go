package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/bdrtr/hush"
	"github.com/golang-jwt/jwt/v5"
	"github.com/valyala/fasthttp"
)

// CORS returns a middleware that handles Cross-Origin Resource Sharing.
func CORS(allowOrigins string, allowMethods ...string) hush.HandlerFunc {
	methods := "GET, POST, PUT, PATCH, DELETE, HEAD, OPTIONS"
	if len(allowMethods) > 0 && allowMethods[0] != "" {
		methods = allowMethods[0]
	}
	
	return func(c *hush.Context) {
		c.Ctx.Response.Header.Set("Access-Control-Allow-Origin", allowOrigins)
		c.Ctx.Response.Header.Set("Access-Control-Allow-Methods", methods)
		c.Ctx.Response.Header.Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
		
		// Guard: Never set Allow-Credentials to true if Allow-Origin is a wildcard.
		// This causes browser security errors and is an inherent security vulnerability.
		if allowOrigins != "*" {
			// In the future, if a config requires credentials, it is safe to set here.
			// c.Ctx.Response.Header.Set("Access-Control-Allow-Credentials", "true")
		}

		if string(c.Ctx.Method()) == fasthttp.MethodOptions {
			c.AbortWithStatus(fasthttp.StatusNoContent)
			return
		}
		
		c.Next()
	}
}

// Logger logs the details of each request using structured logging.
// It accepts an optional slog.Logger instance. If nil, slog.Default() is used.
func Logger(loggers ...*slog.Logger) hush.HandlerFunc {
	var logger *slog.Logger
	if len(loggers) > 0 && loggers[0] != nil {
		logger = loggers[0]
	} else {
		logger = slog.Default()
	}

	return func(c *hush.Context) {
		start := time.Now()
		c.Next()
		duration := time.Since(start)
		
		logger.Info("request processed", 
			slog.String("method", string(c.Ctx.Method())),
			slog.String("path", string(c.Ctx.Path())),
			slog.Int("status", c.Ctx.Response.StatusCode()),
			slog.Duration("duration", duration),
		)
	}
}

// Recovery catches any panics during request handling, logs the stack trace, and returns a 500 error gracefully.
func Recovery(loggers ...*slog.Logger) hush.HandlerFunc {
	var logger *slog.Logger
	if len(loggers) > 0 && loggers[0] != nil {
		logger = loggers[0]
	} else {
		logger = slog.Default()
	}

	return func(c *hush.Context) {
		defer func() {
			if err := recover(); err != nil {
				logger.Error("panic recovered", 
					slog.Any("error", err),
					slog.String("stack", string(debug.Stack())),
					slog.String("path", string(c.Ctx.Path())),
				)
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
				reqID = fmt.Sprintf("fallback-%d", time.Now().UnixNano())
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
		// Use actual TCP RemoteIP to prevent X-Forwarded-For spoofing attacks.
		// If behind a proxy like Cloudflare, the framework config should handle trusted proxies separately.
		ip := c.Ctx.RemoteIP().String()
		
		var count int
		
		// Use an anonymous function with defer for absolute panic/deadlock safety,
		// without holding the lock during c.Next()
		func() {
			mu.Lock()
			defer mu.Unlock()
			
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
				count = 1
			} else {
				data.count++
				count = data.count
			}
		}()
		
		if count > limit {
			c.Ctx.Error("Too Many Requests", fasthttp.StatusTooManyRequests)
			c.Abort()
			return
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
// Note: fasthttp handles timeouts at the server level via ReadTimeout/WriteTimeout.
func Timeout(timeout time.Duration) hush.HandlerFunc {
	return func(c *hush.Context) {
		// Pass the deadline downstream so handlers/DB calls can respect it via context.WithTimeout
		c.Set("timeout", timeout)
		c.Ctx.SetUserValue("deadline", time.Now().Add(timeout))
		c.Next()
	}
}
