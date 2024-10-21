package main

import (
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/sol1corejz/goferrrmart/cmd/config"
	"github.com/sol1corejz/goferrrmart/internal/handlers"
	"github.com/sol1corejz/goferrrmart/internal/logger"
	"github.com/sol1corejz/goferrrmart/internal/middleware"
	"github.com/sol1corejz/goferrrmart/internal/storage"
	"github.com/sol1corejz/goferrrmart/internal/workers"
	"go.uber.org/zap"
)

func main() {
	config.ParseFlags()

	if err := logger.Initialize(config.LogLevel); err != nil {
		logger.Log.Fatal("Failed to initialize logger", zap.Error(err))
	}

	if err := storage.Init(); err != nil {
		logger.Log.Error("Failed to init storage", zap.Error(err))
		return
	}

	workers.InitLoyaltySystem()

	if err := run(); err != nil {
		logger.Log.Fatal("Failed to run server", zap.Error(err))
	}
}

func run() error {
	app := fiber.New()
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowMethods: "GET,POST,OPTIONS",
	}))

	app.Post("/api/user/register", handlers.RegisterHandler)
	app.Post("/api/user/login", handlers.LoginHandler)

	authRoutes := app.Group("/api/user", middleware.AuthMiddleware)
	authRoutes.Get("/orders", handlers.GetOrdersHandler)
	authRoutes.Post("/orders", handlers.CreateOrderHandler)
	authRoutes.Get("/balance", handlers.GetUserBalanceHandler)
	authRoutes.Post("/balance/withdraw", handlers.WithdrawHandler)
	authRoutes.Get("/withdrawals", handlers.GetWithdrawalsHandler)

	logger.Log.Info("Running server", zap.String("address", config.RunAddress))
	return app.Listen(config.RunAddress)
}
