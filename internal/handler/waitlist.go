package handler

import (
	"os"
	"strings"

	"github.com/gofiber/fiber/v2"
)

// POST /waitlist
// Body: {"email":"..."}
func (h *Handler) joinWaitlist(c *fiber.Ctx) error {
	var body struct {
		Email string `json:"email"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid JSON"})
	}

	email := strings.TrimSpace(body.Email)
	if email == "" || !strings.Contains(email, "@") {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "valid email required"})
	}

	h.waitlistMu.Lock()
	defer h.waitlistMu.Unlock()

	f, err := os.OpenFile(h.waitlistPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
	}
	defer f.Close()

	if _, err := f.WriteString(email + "\n"); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
	}

	return c.SendStatus(fiber.StatusOK)
}
