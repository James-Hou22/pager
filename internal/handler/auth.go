package handler

import (
	"errors"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/James-Hou22/pager/internal/store"
)

// POST /auth/register
// Body: {"email":"...","password":"..."}
// Response 201: {"token":"..."}
func (h *Handler) authRegister(c *fiber.Ctx) error {
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid JSON"})
	}
	if body.Email == "" || body.Password == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "email and password are required"})
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(body.Password), 12)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
	}

	org, err := h.store.CreateOrganizer(c.Context(), body.Email, string(hash))
	if err != nil {
		if errors.Is(err, store.ErrConflict) {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "email already registered"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
	}

	token, err := h.signToken(org.ID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"token": token})
}

// POST /auth/login
// Body: {"email":"...","password":"..."}
// Response 200: {"token":"..."} or 401
func (h *Handler) authLogin(c *fiber.Ctx) error {
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid JSON"})
	}
	if body.Email == "" || body.Password == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "email and password are required"})
	}

	org, err := h.store.GetOrganizerByEmail(c.Context(), body.Email)
	if err != nil {
		// Return 401 whether the email doesn't exist or anything else fails,
		// to avoid leaking which emails are registered.
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid credentials"})
	}

	if err := bcrypt.CompareHashAndPassword([]byte(org.PasswordHash), []byte(body.Password)); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid credentials"})
	}

	token, err := h.signToken(org.ID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
	}

	return c.JSON(fiber.Map{"token": token})
}

// GET /auth/me
// Authorization: Bearer <token>
// Response 200: {"id":"...","email":"..."}
func (h *Handler) authMe(c *fiber.Ctx) error {
	organizerID, ok := c.Locals("organizer_id").(string)
	if !ok || organizerID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}
	org, err := h.store.GetOrganizerByID(c.Context(), organizerID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
	}
	return c.JSON(fiber.Map{"organizer_id": org.ID, "email": org.Email})
}

// signToken creates a signed JWT for the given organizer ID with a 72h expiry.
func (h *Handler) signToken(organizerID string) (string, error) {
	claims := jwt.MapClaims{
		"organizer_id": organizerID,
		"exp":          time.Now().Add(72 * time.Hour).Unix(),
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(h.jwtSecret)
}
