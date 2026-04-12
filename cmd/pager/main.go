package main

import (
	"context"
	"log"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"

	"github.com/James-Hou22/pager/internal/handler"
	"github.com/James-Hou22/pager/internal/push"
	"github.com/James-Hou22/pager/internal/store"
)

func main() {
	//Load local env first. local env doesn't exist in prod so it will load .env
	//set environment variables don't get overwritten which is why this works.
	_ = godotenv.Load(".env.local")
	_ = godotenv.Load(".env")

	redisAddr := os.Getenv("REDIS_URL")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	dbURL := os.Getenv("POSTGRES_URL")
	if dbURL == "" {
		log.Fatal("POSTGRES_URL must be set")
	}
	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		log.Fatalf("pgxpool.New: %v", err)
	}
	defer pool.Close()
	if err := pool.Ping(context.Background()); err != nil {
		log.Fatalf("postgres ping: %v", err)
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("JWT_SECRET must be set")
	}

	rdb := store.NewRedisClient(redisAddr)
	s := store.New(rdb, pool)
	f, err := push.New(s)
	if err != nil {
		log.Fatalf("push: %v", err)
	}
	h := handler.New(s, f, []byte(jwtSecret))

	app := fiber.New()

	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	// Expose the VAPID public key so the attendee page can subscribe.
	app.Get("/vapid-public-key", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"key": os.Getenv("VAPID_PUBLIC_KEY")})
	})

	// API routes
	h.Register(app)

	// Static assets (JS, CSS, sw.js, manifest.json, etc.)
	app.Static("/", "./web/static")

	// Attendee SPA — lightweight bundle for /event/:id routes
	app.Get("/event/*", func(c *fiber.Ctx) error {
		return c.SendFile("./web/static/attendee.html")
	})

	// Organizer SPA catch-all
	app.Get("*", func(c *fiber.Ctx) error {
		return c.SendFile("./web/static/index.html")
	})

	log.Fatal(app.Listen(":" + port))
}
