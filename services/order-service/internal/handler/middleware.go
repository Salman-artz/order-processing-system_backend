package handler

import (
	"strings"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/order-processing-system/pkg/jwtutil"
)

// JWTMiddleware validates the JWT token and extracts claims into the fiber context.
func JWTMiddleware(c *fiber.Ctx) error {
	authHeader := c.Get("Authorization")
	if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Missing or invalid Authorization header"})
	}

	tokenStr := strings.TrimPrefix(authHeader, "Bearer ")

	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "fallback-secret-for-dev"
	}
	cfg := jwtutil.Config{Secret: secret}

	claims, err := jwtutil.ValidateToken(cfg, tokenStr)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid or expired token"})
	}

	c.Locals("user_id", claims.UserID)
	c.Locals("email", claims.Email)
	c.Locals("role", claims.Role)

	return c.Next()
}

// RequireRole middleware ensures the user has a specific role.
func RequireRole(role jwtutil.Role) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userRole := c.Locals("role")
		if userRole != role {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Forbidden: insufficient permissions"})
		}
		return c.Next()
	}
}
