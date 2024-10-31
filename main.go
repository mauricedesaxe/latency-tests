package main

import (
	latency_simulations "go-on-rails/latency-simulations"
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
)

// This is the entry point for the application.
// Here we initialize & start the Fiber app. We also add routes
// from other modules here.
// Don't put too much logic here, just enough to get the app running.

func main() {
	log.Println("Starting server on port 3000")
	app := fiber.New()
	app.Use(logger.New())

	// routes
	app.Static("/", "./public")
	latency_simulations.AddRoutes(app)

	err := app.Listen(":3000")
	if err != nil {
		log.Println("Error starting server")
		log.Println(err)
	}
}
