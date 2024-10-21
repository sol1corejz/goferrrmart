package middleware

import (
	"github.com/sol1corejz/goferrrmart/internal/logger"
	"github.com/sol1corejz/goferrrmart/internal/tokenstorage"
)

func AuthMiddleware(c *fiber.Ctx) error {
	token := c.Cookies("jwt")

	if token == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	isValid := tokenstorage.CheckToken(token)

	if !isValid {
		logger.Log.Error("Token validation failed")
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid or expired token",
		})
	}

	return c.Next()
}
