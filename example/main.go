package main

import (
	"fmt"
	"log"

	"github.com/bdrtr/hush"
	"github.com/bdrtr/hush/middleware"
)

type UserRequest struct {
	Name  string `json:"name" validate:"required"`
	Email string `json:"email" validate:"required"`
}

type UserResponse struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

// Database represents a fake database interface
type Database interface {
	GetUserName(id string) string
}

type PostgresDB struct{}

func (db *PostgresDB) GetUserName(id string) string {
	return "Alice injected from DB"
}

func Logger() hush.HandlerFunc {
	return func(c *hush.Context) {
		fmt.Printf("[LOG] %s %s\n", c.Request.Method, c.Request.URL.Path)
		c.Next()
	}
}

func AuthMiddleware() hush.HandlerFunc {
	return func(c *hush.Context) {
		token := c.Request.Header.Get("Authorization")
		if token != "secret" {
			c.AbortWithJSON(401, map[string]string{"error": "Unauthorized"})
			return
		}
		// Store user data in Context
		c.Set("user", "admin")
		c.Next()
	}
}

func main() {
	app := hush.New()
	
	// Phase 3: Dependency Injection
	var db Database = &PostgresDB{}
	hush.Provide[Database](app, db)
	
	// Serve Swagger UI at /docs
	app.ServeSwaggerUI("/docs")

	// Phase 2: Global Security Middlewares
	app.Use(middleware.Helmet())
	app.Use(middleware.RequestID())
	app.Use(middleware.CORS("*"))
	app.Use(Logger())

	// Phase 2: Static File Serving
	// Make sure a "public" folder exists if you want to test this
	app.Static("/assets", "./public")

	// Basic route
	app.GET("/ping", func(c *hush.Context) {
		reqID, _ := c.Get("request_id")
		c.Ok(map[string]interface{}{
			"message": "pong",
			"req_id":  reqID,
		})
	})

	// Route Group with Middleware
	api := app.Group("/api/v1")
	api.Use(AuthMiddleware())
	
	api.GET("/users/:id", func(c *hush.Context) {
		id := c.Param("id")
		user, _ := c.Get("user")
		
		// Phase 3: Inject Database Dependency
		db := hush.Inject[Database](c)
		name := "Unknown"
		if db != nil {
			name = db.GetUserName(id)
		}
		
		c.Ok(map[string]interface{}{
			"message": "User found", 
			"id": id,
			"accessed_by": user,
			"db_result": name,
		})
	})
	
	api.POST("/users", func(c *hush.Context) {
		req, err := hush.BindBody[UserRequest](c)
		if err != nil {
			c.BadRequest(err.Error())
			return
		}
		
		res := UserResponse{
			ID:    1,
			Name:  req.Name,
			Email: req.Email,
		}
		c.Created(res)
	})

	log.Println("Server running on http://localhost:8080")
	// Graceful shutdown and TLS are supported via app.Shutdown and app.RunTLS
	log.Fatal(app.Run(":8080"))
}
