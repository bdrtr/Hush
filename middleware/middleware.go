package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"

	"github.com/bdrtr/hush"
)

// CORS returns a middleware that handles Cross-Origin Resource Sharing.
func CORS(allowOrigins string) hush.HandlerFunc {
	return func(c *hush.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", allowOrigins)
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		
		c.Next()
	}
}

// RequestID generates a unique ID for each request.
func RequestID() hush.HandlerFunc {
	return func(c *hush.Context) {
		reqID := c.Request.Header.Get("X-Request-ID")
		if reqID == "" {
			bytes := make([]byte, 16)
			if _, err := rand.Read(bytes); err == nil {
				reqID = hex.EncodeToString(bytes)
			} else {
				reqID = "unknown"
			}
		}
		
		// Set in header so client gets it back
		c.Writer.Header().Set("X-Request-ID", reqID)
		
		// Set in context so handlers can use it for logging
		c.Set("request_id", reqID)
		
		c.Next()
	}
}

// Helmet sets standard security headers.
func Helmet() hush.HandlerFunc {
	return func(c *hush.Context) {
		c.Writer.Header().Set("X-XSS-Protection", "1; mode=block")
		c.Writer.Header().Set("X-Content-Type-Options", "nosniff")
		c.Writer.Header().Set("X-Frame-Options", "DENY")
		c.Writer.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		// HSTS can be added here as well
		
		c.Next()
	}
}

// BasicAuth provides simple username/password validation.
func BasicAuth(username, password string) hush.HandlerFunc {
	return func(c *hush.Context) {
		u, p, ok := c.Request.BasicAuth()
		if !ok || u != username || p != password {
			c.Writer.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
			c.AbortWithJSON(http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
			return
		}
		c.Set("user", u)
		c.Next()
	}
}
