package middleware

import (
	"sync"
	"time"

	"github.com/bdrtr/hush"
	"github.com/valyala/fasthttp"
)

type cacheEntry struct {
	body        []byte
	contentType []byte
	statusCode  int
	expires     time.Time
}

// Cache returns a middleware that caches the response for the given duration based on the request URL.
func Cache(duration time.Duration) hush.HandlerFunc {
	var cache sync.Map // concurrent-safe map

	return func(c *hush.Context) {
		key := string(c.Ctx.RequestURI())
		
		if val, ok := cache.Load(key); ok {
			entry := val.(cacheEntry)
			if time.Now().Before(entry.expires) {
				// Cache hit
				c.Ctx.SetContentTypeBytes(entry.contentType)
				c.Ctx.SetStatusCode(entry.statusCode)
				c.Ctx.Write(entry.body)
				c.Abort() // Prevent next handlers from executing
				return
			}
			// Cache expired, delete it
			cache.Delete(key)
		}
		
		// Cache miss: Let the handlers run and capture the response
		c.Next()
		
		// Only cache successful GET requests
		if string(c.Ctx.Method()) == fasthttp.MethodGet && c.Ctx.Response.StatusCode() == fasthttp.StatusOK {
			// Copy bytes because fasthttp reuses buffers across requests
			bodyCopy := make([]byte, len(c.Ctx.Response.Body()))
			copy(bodyCopy, c.Ctx.Response.Body())
			
			contentTypeCopy := make([]byte, len(c.Ctx.Response.Header.ContentType()))
			copy(contentTypeCopy, c.Ctx.Response.Header.ContentType())

			cache.Store(key, cacheEntry{
				body:        bodyCopy,
				contentType: contentTypeCopy,
				statusCode:  c.Ctx.Response.StatusCode(),
				expires:     time.Now().Add(duration),
			})
		}
	}
}
