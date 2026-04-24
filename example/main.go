package main

import (
	"fmt"
	"log"

	"github.com/bdrtr/hush"
	"github.com/bdrtr/hush/ext/essence"
	"github.com/bdrtr/hush/middleware"
)

type LocationRequest struct {
	UserID string  `json:"user_id" validate:"required"`
	Lat    float64 `json:"lat" validate:"required"`
	Lon    float64 `json:"lon" validate:"required"`
}

func Logger() hush.HandlerFunc {
	return func(c *hush.Context) {
		fmt.Printf("[LOG] %s %s\n", c.Ctx.Method(), c.Ctx.Path())
		c.Next()
	}
}

func main() {
	app := hush.New()
	
	// Phase 4: Essence DB Dependency Injection
	// (Note: To compile this, the Rust Essence core must be built in heartbeat-project)
	db := essence.New()
	defer db.Close()
	hush.Provide[*essence.EssenceDB](app, db)
	
	// Serve Swagger UI at /docs
	app.ServeSwaggerUI("/docs")

	// Global Middlewares
	app.Use(middleware.Helmet())
	app.Use(middleware.RequestID())
	app.Use(middleware.CORS("*"))
	app.Use(Logger())

	// Static File Serving
	app.Static("/assets", "./public")

	// Basic route
	app.GET("/ping", func(c *hush.Context) {
		reqID, _ := c.Get("request_id")
		c.Ok(map[string]interface{}{
			"message": "pong",
			"req_id":  reqID,
		})
	})

	// Spatial API Group
	geo := app.Group("/geo")
	
	geo.POST("/update", func(c *hush.Context) {
		req, err := hush.BindBody[LocationRequest](c)
		if err != nil {
			c.BadRequest(err.Error())
			return
		}
		
		db := hush.Inject[*essence.EssenceDB](c)
		if db != nil {
			db.UpdateLocation(req.UserID, req.Lat, req.Lon, 10.0) // 10.0 alt
		}
		
		c.Ok(map[string]string{"status": "Location updated"})
	})
	
	geo.GET("/nearby/:user_id", func(c *hush.Context) {
		userID := c.Param("user_id")
		
		db := hush.Inject[*essence.EssenceDB](c)
		var matches []essence.Match
		if db != nil {
			matches = db.GetNearbyUsers(userID, 5.0, 10) // 5km radius, max 10
		}
		
		c.Ok(map[string]interface{}{
			"user_id": userID,
			"matches": matches,
		})
	})

	log.Println("Fasthttp Server running on http://localhost:8080")
	log.Fatal(app.Run(":8080"))
}
