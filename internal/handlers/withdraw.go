package handlers

import (
	"context"
	"database/sql"
	"errors"
	"github.com/sol1corejz/goferrrmart/internal/auth"
	"github.com/sol1corejz/goferrrmart/internal/logger"
	"github.com/sol1corejz/goferrrmart/internal/storage"
	"go.uber.org/zap"
	"time"
)

type WithdrawRequest struct {
	Order string  `json:"order" validate:"required"`
	Sum   float64 `json:"sum" validate:"required"`
}

func WithdrawHandler(c *fiber.Ctx) error {
	var request WithdrawRequest
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	select {
	case <-ctx.Done():
		logger.Log.Warn("Context canceled or timeout exceeded")
		return c.Status(fiber.StatusRequestTimeout).JSON(fiber.Map{
			"error": "Request timed out",
		})
	default:
		token := c.Cookies("jwt")

		userID := auth.GetUserID(token)

		if err := c.BodyParser(&request); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid request body",
			})
		}

		balance, err := storage.GetUserBalance(ctx, userID)
		if err != nil {
			logger.Log.Error("Error getting user balance", zap.Error(err))
			return c.SendStatus(fiber.StatusInternalServerError)
		}

		if balance.CurrentBalance < request.Sum {
			return c.Status(fiber.StatusPaymentRequired).JSON(fiber.Map{
				"error": "Insufficient funds",
			})
		}

		_, err = storage.GetOrderByNumber(ctx, request.Order)

		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				logger.Log.Warn("No such order found", zap.Error(err))
				return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
					"error": "Order does not exist",
				})
			}
			logger.Log.Error("Database error while fetching order", zap.Error(err))
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Internal server error",
			})
		}

		err = storage.CreateWithdrawal(ctx, userID, request.Order, request.Sum)
		if err != nil {
			logger.Log.Error("Error creating withdrawal", zap.Error(err))
			return c.SendStatus(fiber.StatusInternalServerError)
		}

		logger.Log.Info("Withdrawal created successfully", zap.String("userID", userID.String()), zap.String("order", request.Order), zap.Float64("sum", request.Sum))
		return c.SendStatus(fiber.StatusOK)
	}
}

type WithdrawalsResponse struct {
	Order       string    `json:"order"`
	Sum         float64   `json:"sum"`
	ProcessedAt time.Time `json:"processed_at"`
}

func GetWithdrawalsHandler(c *fiber.Ctx) error {
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

		userID := auth.GetUserID(token)

		withdrawals, err := storage.GetUserWithdrawals(ctx, userID)

		if err != nil {
			logger.Log.Error("Error getting user withdrawals", zap.Error(err))
			return c.SendStatus(fiber.StatusInternalServerError)
		}

		if len(withdrawals) == 0 {
			logger.Log.Info("No withdrawals found")
			return c.SendStatus(fiber.StatusNoContent)
		}

		var response []WithdrawalsResponse
		for _, withdrawal := range withdrawals {
			response = append(response, WithdrawalsResponse{
				Order:       withdrawal.OrderNumber,
				Sum:         withdrawal.Sum,
				ProcessedAt: withdrawal.ProcessedAt,
			})
		}

		return c.Status(fiber.StatusOK).JSON(response)
	}
}
