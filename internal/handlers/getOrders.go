package handlers

import (
	"context"
	"github.com/gofiber/fiber/v2"
	"github.com/sol1corejz/goferrrmart/internal/auth"
	"github.com/sol1corejz/goferrrmart/internal/logger"
	"github.com/sol1corejz/goferrrmart/internal/storage"
	"go.uber.org/zap"
	"time"
)

type OrderResponse struct {
	Number     string    `json:"number"`
	Status     string    `json:"status"`
	Accrual    float64   `json:"accrual,omitempty"`
	UploadedAt time.Time `json:"uploaded_at"`
}

func GetOrdersHandler(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	select {
	case <-ctx.Done():
		logger.Log.Warn("Context canceled or timeout exceeded")
		return c.Status(fiber.StatusRequestTimeout).JSON(fiber.Map{
			"error": "Request timed out",
		})
	default:
		token := c.Cookies("jwt")

		userID, err := auth.GetUserID(token)
		if err != nil {
			logger.Log.Warn("Error getting user ID", zap.Error(err))
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Error getting user ID",
			})
		}

		orders, err := storage.GetUserOrders(ctx, userID)

		if err != nil {
			logger.Log.Error("Error getting user orders", zap.Error(err))
			return c.SendStatus(fiber.StatusInternalServerError)
		}

		if len(orders) == 0 {
			logger.Log.Info("No orders found")
			return c.SendStatus(fiber.StatusNoContent)
		}

		var response []OrderResponse
		for _, order := range orders {
			response = append(response, OrderResponse{
				Number:     order.OrderNumber,
				Status:     order.Status,
				Accrual:    order.Accrual,
				UploadedAt: order.UploadedAt,
			})
		}

		return c.Status(fiber.StatusOK).JSON(response)
	}
}
