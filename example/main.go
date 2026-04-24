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

type GenericResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

func Logger() hush.HandlerFunc {
	return func(c *hush.Context) {
		fmt.Printf("[LOG] %s %s\n", c.Ctx.Method(), c.Ctx.Path())
		c.Next()
	}
}

func main() {
	app := hush.New()
	
	db := essence.New()
	defer db.Close()
	hush.Provide[*essence.EssenceDB](app, db)
	
	app.ServeSwaggerUI("/docs")

	app.Use(middleware.Helmet())
	app.Use(middleware.RequestID())
	app.Use(middleware.CORS("*"))
	app.Use(middleware.RateLimit(100, 60)) // 100 requests per minute
	app.Use(Logger())

	app.GET("/ping", func(c *hush.Context) {
		c.Ok(map[string]string{"message": "pong"})
	}).WithSummary("Health check endpoint").WithTags("System")

	geo := app.Group("/geo")
	
	hush.WithBody[LocationRequest](
		geo.POST("/update", func(c *hush.Context) {
			req, err := hush.BindBody[LocationRequest](c)
			if err != nil {
				c.BadRequest(err.Error())
				return
			}
			
			db := hush.Inject[*essence.EssenceDB](c)
			if db != nil {
				db.UpdateLocation(req.UserID, req.Lat, req.Lon, 10.0)
			}
			
			c.Ok(GenericResponse{Status: "success", Message: "Location updated"})
		}).WithSummary("Update user location").WithTags("Geo"),
	).WithResponse[GenericResponse]()
	
	geo.GET("/nearby/:user_id", func(c *hush.Context) {
		userID := c.Param("user_id")
		db := hush.Inject[*essence.EssenceDB](c)
		var matches []essence.Match
		if db != nil {
			matches = db.GetNearbyUsers(userID, 5.0, 10)
		}
		c.Ok(map[string]interface{}{
			"user_id": userID,
			"matches": matches,
		})
	}).WithSummary("Get nearby users").WithTags("Geo")

	log.Println("Hush (Fasthttp) running on http://localhost:8080")
	log.Fatal(app.Run(":8080"))
}
