package handler

import (
	"errors"
	"strings"

	"github.com/James-Hou22/pager/internal/store"
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

	if err := h.store.AddToWaitlist(c.Context(), email); err != nil {
		if errors.Is(err, store.ErrConflict) {
			return c.SendStatus(fiber.StatusOK)
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
	}

	return c.SendStatus(fiber.StatusOK)
}
