package handlers

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"github.com/gofiber/fiber/v2"
	"github.com/sol1corejz/goferrrmart/cmd/config"
	"github.com/sol1corejz/goferrrmart/internal/auth"
	"github.com/sol1corejz/goferrrmart/internal/logger"
	"github.com/sol1corejz/goferrrmart/internal/storage"
	"go.uber.org/zap"
	"net/http"
	"regexp"
	"time"
)

type Good struct {
	Description string  `json:"description"`
	Price       float64 `json:"price"`
}

type Order struct {
	OrderNumber string `json:"order"`
	OrderGoods  []Good `json:"goods"`
}

var luhnCheck = regexp.MustCompile(`^\d+$`)

func isValidLuhn(order string) bool {
	var sum int
	var double bool

	for i := len(order) - 1; i >= 0; i-- {
		n := int(order[i] - '0')

		if double {
			n *= 2
			if n > 9 {
				n -= 9
			}
		}
		sum += n
		double = !double
	}

	return sum%10 == 0
}

func CreateOrderHandler(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	select {
	case <-ctx.Done():
		logger.Log.Warn("Context canceled or timeout exceeded")
		return c.Status(fiber.StatusRequestTimeout).JSON(fiber.Map{
			"error": "Request timed out",
		})
	default:
		orderNumber := c.Body()

		token := c.Cookies("jwt")
		userID := auth.GetUserID(token)

		if !luhnCheck.Match(orderNumber) {
			logger.Log.Error("Invalid order number")
			return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
				"error": "Invalid order number",
			})
		}

		if !isValidLuhn(string(orderNumber)) {
			logger.Log.Error("Invalid order number")
			return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
				"error": "Invalid order number",
			})
		}

		order, err := storage.GetOrderByNumber(ctx, string(orderNumber))

		if err != nil {
			if !errors.Is(err, sql.ErrNoRows) {
				logger.Log.Error("Error checking order", zap.Error(err))
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "Error checking order",
				})
			}
		}

		if order.OrderNumber == string(orderNumber) && order.UserID == userID {
			logger.Log.Info("Order number already registered by this user")
			return c.Status(fiber.StatusOK).JSON(fiber.Map{
				"message": "Order already registered by this user",
			})
		}

		if order.OrderNumber != "" {
			logger.Log.Info("Order number already exists")
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"error": "Order number already exists",
			})
		}

		err = storage.CreateOrder(ctx, userID.String(), string(orderNumber))
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Error creating order",
			})
		}

		orderToPost := Order{
			OrderNumber: string(orderNumber),
			OrderGoods: []Good{
				{
					Description: "Чайник Bork",
					Price:       7000,
				},
			},
		}
		jsonData, _ := json.Marshal(orderToPost)

		resp, err := http.Post(config.AccrualSystemAddress, "application/json", bytes.NewBuffer(jsonData))
		defer resp.Body.Close()

		if err != nil {
			return err
		}

		return c.Status(fiber.StatusAccepted).JSON(fiber.Map{
			"message": "Order created",
		})
	}
}
