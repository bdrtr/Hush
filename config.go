package hush

import (
	"time"

	"github.com/valyala/fasthttp"
)

// Config holds the server configuration settings.
type Config struct {
	MaxRequestBodySize int
	ReadTimeout        time.Duration
	WriteTimeout       time.Duration
	IdleTimeout        time.Duration
	Concurrency        int
	ReduceMemoryUsage  bool
	Debug              bool
	SoftMemoryLimit    int64
	Logger             fasthttp.Logger
}

// Option represents a configuration option for the engine.
type Option func(*Config)

// WithMaxRequestBodySize sets the maximum allowed size for a request body.
func WithMaxRequestBodySize(size int) Option { return func(c *Config) { c.MaxRequestBodySize = size } }

// WithReadTimeout sets the maximum duration for reading the entire request.
func WithReadTimeout(t time.Duration) Option { return func(c *Config) { c.ReadTimeout = t } }

// WithWriteTimeout sets the maximum duration before timing out writes of the response.
func WithWriteTimeout(t time.Duration) Option { return func(c *Config) { c.WriteTimeout = t } }

// WithIdleTimeout sets the maximum amount of time to wait for the next request when keep-alive is enabled.
func WithIdleTimeout(t time.Duration) Option { return func(c *Config) { c.IdleTimeout = t } }

// WithConcurrency sets the maximum number of concurrent connections the server may serve.
func WithConcurrency(n int) Option { return func(c *Config) { c.Concurrency = n } }

// WithReduceMemoryUsage aggregates and delays small writes to reduce memory usage.
func WithReduceMemoryUsage(b bool) Option { return func(c *Config) { c.ReduceMemoryUsage = b } }

// WithSoftMemoryLimit configures the runtime's soft memory limit (GOMEMLIMIT).
// This instructs the Go GC to delay running until this limit is approached,
// eliminating micro-pauses and dramatically reducing P99 latency.
func WithSoftMemoryLimit(limit int64) Option { return func(c *Config) { c.SoftMemoryLimit = limit } }

// WithDebug enables or disables debug mode (e.g. printing route registration).
func WithDebug(b bool) Option { return func(c *Config) { c.Debug = b } }

// WithLogger sets a custom logger for the fasthttp server.
func WithLogger(l fasthttp.Logger) Option { return func(c *Config) { c.Logger = l } }

// DefaultConfig returns the secure default configuration.
func DefaultConfig() *Config {
	return &Config{
		MaxRequestBodySize: 10 * 1024 * 1024, // 10MB limit prevents large payload DoS
		ReadTimeout:        30 * time.Second,  // Protects against Slowloris (slow read)
		WriteTimeout:       30 * time.Second,  // Protects against slow clients
		IdleTimeout:        90 * time.Second,  // Keep-Alive connection timeout
		Concurrency:        256 * 1024,        // Fasthttp default (256k concurrent)
		Debug:              false,
	}
}
