package handlers

import (
	"context"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/sol1corejz/goferrrmart/internal/logger"
	"github.com/sol1corejz/goferrrmart/internal/storage"
	"go.uber.org/zap"
	"time"
)

type BalanceResponse struct {
	Current   float64 `json:"current"`
	Withdrawn float64 `json:"withdrawn"`
}

func GetUserBalanceHandler(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	select {
	case <-ctx.Done():
		logger.Log.Warn("Context canceled or timeout exceeded")
		return c.Status(fiber.StatusRequestTimeout).JSON(fiber.Map{
			"error": "Request timed out",
		})
	default:
		userID := c.Locals("userID").(uuid.UUID)

		balance, err := storage.GetUserBalance(ctx, userID)

		if err != nil {
			logger.Log.Error("Error getting user orders", zap.Error(err))
			return c.SendStatus(fiber.StatusInternalServerError)
		}

		return c.Status(fiber.StatusOK).JSON(BalanceResponse{
			Current:   balance.CurrentBalance,
			Withdrawn: balance.WithdrawnTotal,
		})
	}
}
