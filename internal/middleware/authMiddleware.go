package middleware

import (
	"github.com/gofiber/fiber/v2"
	"github.com/sol1corejz/goferrrmart/internal/auth"
)

func AuthMiddleware(c *fiber.Ctx) error {
	// Получение токена из cookies
	tokenString := c.Cookies("jwt")
	if tokenString == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	// Проверка токена и извлечение UserID
	userID, err := auth.GetUserID(tokenString)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid or expired token",
		})
	}

	// Сохранение userID в контексте для использования в последующих обработчиках
	c.Locals("userID", userID)

	return c.Next()
}
