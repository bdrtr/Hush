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

// Cache returns a middleware that caches GET responses.
// maxEntries prevents OOM by limiting the total number of cached routes (Random Eviction when full).
// maxResponseSize prevents caching extremely large files (e.g., 5*1024*1024 for 5MB limit).
func Cache(duration time.Duration, maxEntries int, maxResponseSize int) hush.HandlerFunc {
	var mu sync.RWMutex
	cache := make(map[string]cacheEntry)

	return func(c *hush.Context) {
		// Only cache GET requests. Skip others immediately.
		if string(c.Ctx.Method()) != fasthttp.MethodGet {
			c.Next()
			return
		}

		key := string(c.Ctx.RequestURI())
		
		mu.RLock()
		val, ok := cache[key]
		mu.RUnlock()
		
		if ok {
			if time.Now().Before(val.expires) {
				// Cache hit
				c.Ctx.SetContentTypeBytes(val.contentType)
				c.Ctx.SetStatusCode(val.statusCode)
				c.Ctx.Write(val.body)
				c.Abort() // Prevent next handlers from executing
				return
			}
			// Cache expired, delete it safely
			mu.Lock()
			// Double-check because another goroutine might have updated the cache between RUnlock and Lock
			val, ok = cache[key]
			if ok && time.Now().After(val.expires) {
				delete(cache, key)
			}
			mu.Unlock()
		}
		
		// Cache miss: Let the handlers run and capture the response
		c.Next()
		
		// Only cache successful GET requests
		if c.Ctx.Response.StatusCode() == fasthttp.StatusOK {
			bodyLen := len(c.Ctx.Response.Body())
			if maxResponseSize > 0 && bodyLen > maxResponseSize {
				return // Response too large, bypass caching
			}

			// Copy bytes because fasthttp reuses buffers across requests
			bodyCopy := make([]byte, bodyLen)
			copy(bodyCopy, c.Ctx.Response.Body())
			
			contentTypeCopy := make([]byte, len(c.Ctx.Response.Header.ContentType()))
			copy(contentTypeCopy, c.Ctx.Response.Header.ContentType())

			mu.Lock()
			// Random Eviction: Go map iteration order is random, so this naturally drops a random key
			if maxEntries > 0 && len(cache) >= maxEntries {
				for k := range cache {
					delete(cache, k)
					break
				}
			}
			
			cache[key] = cacheEntry{
				body:        bodyCopy,
				contentType: contentTypeCopy,
				statusCode:  c.Ctx.Response.StatusCode(),
				expires:     time.Now().Add(duration),
			}
			mu.Unlock()
		}
	}
}
