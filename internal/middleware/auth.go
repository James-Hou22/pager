package middleware

import (
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

// Auth returns a Fiber middleware that validates a Bearer JWT and stores the
// organizer_id claim in c.Locals("organizer_id"). Returns 401 on any failure.
func Auth(secret []byte) fiber.Handler {
	return func(c *fiber.Ctx) error {
		header := c.Get("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "missing or invalid authorization header"})
		}
		tokenStr := strings.TrimPrefix(header, "Bearer ")

		token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (any, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return secret, nil
		})
		if err != nil || !token.Valid {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid or expired token"})
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid token claims"})
		}

		organizerID, ok := claims["organizer_id"].(string)
		if !ok || organizerID == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid token claims"})
		}

		c.Locals("organizer_id", organizerID)
		return c.Next()
	}
}
