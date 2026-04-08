package main

import (
	"log"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/joho/godotenv"

	"github.com/James-Hou22/pager/internal/handler"
	"github.com/James-Hou22/pager/internal/push"
	"github.com/James-Hou22/pager/internal/store"
)

func main() {
	// Load .env if present; ignore error when file doesn't exist.
	_ = godotenv.Load()

	redisAddr := os.Getenv("REDIS_URL")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	rdb := store.NewRedisClient(redisAddr)
	s := store.New(rdb)
	f, err := push.New(s)
	if err != nil {
		log.Fatalf("push: %v", err)
	}
	h := handler.New(s, f)

	app := fiber.New()

	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	// Expose the VAPID public key so the attendee page can subscribe.
	app.Get("/vapid-public-key", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"key": os.Getenv("VAPID_PUBLIC_KEY")})
	})

	// Serve the React SPA for organizer and attendee routes.
	app.Get("/organizer", func(c *fiber.Ctx) error {
		return c.SendFile("./web/static/index.html")
	})

	// Serve attendee page at /channel/:id (HTML) — must come before the API routes.
	app.Get("/channel/:id", func(c *fiber.Ctx) error {
		// If the client wants JSON (API call), fall through to the API handler.
		if c.Accepts("text/html") == "text/html" {
			return c.SendFile("./web/static/index.html")
		}
		return c.Next()
	})

	h.Register(app)

	// Static assets — sw.js, manifest.json, icons, etc.
	app.Static("/", "./web/static")

	log.Fatal(app.Listen(":" + port))
}
