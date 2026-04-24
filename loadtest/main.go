package main

import (
	"log"

	"github.com/bdrtr/hush"
)

func main() {
	// Enable extreme GC optimizations for load test
	app := hush.New(
		hush.WithSoftMemoryLimit(100 * 1024 * 1024), // 100MB soft limit
		hush.WithConcurrency(20000), // Handle up to 20k concurrent connections
	)

	// Static route O(1) xxhash test
	app.GET("/", func(c *hush.Context) {
		c.Ok("Hello from Hush!")
	})

	// JSON response test
	app.GET("/api/data", func(c *hush.Context) {
		c.JSON(200, map[string]string{
			"status": "success",
			"framework": "hush",
		})
	})

	log.Println("Starting load test server on :8080")
	if err := app.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}
