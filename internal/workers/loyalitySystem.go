package workers

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/sol1corejz/goferrrmart/cmd/config"
	"github.com/sol1corejz/goferrrmart/internal/auth"
	"github.com/sol1corejz/goferrrmart/internal/logger"
	"github.com/sol1corejz/goferrrmart/internal/models"
	"github.com/sol1corejz/goferrrmart/internal/storage"
	"go.uber.org/zap"
	"io"
	"net/http"
	"strconv"
	"time"

	_ "github.com/lib/pq"
)

type LoyaltyResponse struct {
	Order   string  `json:"order"`
	Status  string  `json:"status"`
	Accrual float64 `json:"accrual,omitempty"`
}

const WorkerInterval = 5 * time.Second

func InitLoyaltySystem() {
	go startWorker()

	logger.Log.Info("Loyalty system worker started")
}

func startWorker() {
	ticker := time.NewTicker(WorkerInterval)
	for range ticker.C {
		checkOrdersForProcessing()
	}
}

func checkOrdersForProcessing() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	select {
	case <-ctx.Done():
		logger.Log.Info("Cancelling orders")
	default:
		orders, err := storage.GetAllUnprocessedOrders(ctx)

		if err != nil {
			logger.Log.Error("Error getting orders", zap.Error(err))
			return
		}

		for _, order := range orders {
			logger.Log.Info("Checking order:", zap.String("orderNumber", order.OrderNumber))
			loyaltyResp, err := queryLoyaltySystem(order.OrderNumber)
			if err != nil {
				logger.Log.Error("Failed to query loyalty system for order", zap.String("orderNumber", order.OrderNumber), zap.Error(err))
				continue
			}

			updateOrderStatus(order.ID, loyaltyResp)
		}

	}
}

func queryLoyaltySystem(orderNumber string) (LoyaltyResponse, error) {
	url := fmt.Sprintf("%s%s%s", config.AccrualSystemAddress, "/api/orders/", orderNumber)
	logger.Log.Info("Querying loyalty system", zap.String("url", url))
	resp, err := http.Get(url)
	if err != nil {
		return LoyaltyResponse{}, fmt.Errorf("HTTP request failed: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var loyaltyResp LoyaltyResponse
	err = json.Unmarshal(body, &loyaltyResp)
	if err != nil {
		logger.Log.Error("Failed to decode response", zap.Error(err))
		return LoyaltyResponse{}, fmt.Errorf("failed to decode response: %w", err)
	}

	return loyaltyResp, nil
}

func updateOrderStatus(orderID int, loyaltyResp LoyaltyResponse) {
	var newStatus string

	switch loyaltyResp.Status {
	case "PROCESSED":
		newStatus = models.PROCESSED
	case "INVALID":
		newStatus = models.INVALID
	case "PROCESSING":
		newStatus = models.PROCESSING
	default:
		newStatus = models.REGISTERED
	}

	accrual := loyaltyResp.Accrual

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	select {
	case <-ctx.Done():
		logger.Log.Info("Cancel updating orders")
	default:
		err := storage.UpdateOrder(ctx, orderID, newStatus, accrual, auth.UserID)
		if err != nil {
			logger.Log.Error("Failed to update orders", zap.Error(err))
		}

		logger.Log.Info("Order updated", zap.String("orderID", strconv.Itoa(orderID)))
	}
}
