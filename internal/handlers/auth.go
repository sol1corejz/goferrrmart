package handlers

import (
	"context"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/sol1corejz/goferrrmart/internal/auth" // Путь к вашему auth пакету
	"github.com/sol1corejz/goferrrmart/internal/logger"
	"github.com/sol1corejz/goferrrmart/internal/storage" // Путь к вашему пакету работы с базой данных
	"github.com/sol1corejz/goferrrmart/internal/tokenstorage"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"time"
)

type RegisterRequest struct {
	Login    string `json:"login" validate:"required"`
	Password string `json:"password" validate:"required"`
}

func RegisterHandler(c *fiber.Ctx) error {
	var request RegisterRequest
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	select {
	case <-ctx.Done():
		logger.Log.Warn("Context canceled or timeout exceeded")
		return c.Status(fiber.StatusRequestTimeout).JSON(fiber.Map{
			"error": "Request timed out",
		})
	default:
		if err := c.BodyParser(&request); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid request body",
			})
		}

		existingUser, err := storage.GetUserByLogin(ctx, request.Login)
		if err != nil {
			logger.Log.Error("Error while querying user: ", zap.Error(err))
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Internal server error",
			})
		}

		if existingUser.ID.String() != uuid.Nil.String() {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"error": "User already exists",
			})
		}

		userID := uuid.New()
		token, err := auth.GenerateToken(userID)
		if err != nil {
			logger.Log.Error("Error generating token: ", zap.Error(err))
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Internal server error",
			})
		}

		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(request.Password), bcrypt.DefaultCost)
		if err != nil {
			logger.Log.Error("Error hashing password: ", zap.Error(err))
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Internal server error",
			})
		}

		err = storage.CreateUser(ctx, userID.String(), request.Login, string(hashedPassword))
		if err != nil {
			logger.Log.Error("Error creating user: ", zap.Error(err))
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Internal server error",
			})
		}

		tokenstorage.AddToken(token)

		auth.UserID = userID

		c.Cookie(&fiber.Cookie{
			Name:     "jwt",
			Value:    token,
			Expires:  time.Now().Add(auth.TokenExp),
			HTTPOnly: true,
		})

		c.Set("Authorization", "Bearer "+token)

		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"message": "User registered successfully",
		})
	}
}

func LoginHandler(c *fiber.Ctx) error {
	var request RegisterRequest
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	select {
	case <-ctx.Done():
		logger.Log.Warn("Context canceled or timeout exceeded")
		return c.Status(fiber.StatusRequestTimeout).JSON(fiber.Map{
			"error": "Request timed out",
		})
	default:
		if err := c.BodyParser(&request); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid request body",
			})
		}

		existingUser, err := storage.GetUserByLogin(ctx, request.Login)
		if err != nil {
			logger.Log.Error("Error while querying user: ", zap.Error(err))
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Internal server error",
			})
		}

		if existingUser.ID.String() == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Wrong login or password",
			})
		}

		err = bcrypt.CompareHashAndPassword([]byte(existingUser.PasswordHash), []byte(request.Password))
		if err != nil {
			logger.Log.Error("Error while comparing hash: ", zap.Error(err))
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Wrong login or password",
			})
		}

		token, err := auth.GenerateToken(existingUser.ID)
		if err != nil {
			logger.Log.Error("Error generating token: ", zap.Error(err))
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Internal server error",
			})
		}

		tokenstorage.AddToken(token)

		auth.UserID = existingUser.ID

		c.Cookie(&fiber.Cookie{
			Name:     "jwt",
			Value:    token,
			Expires:  time.Now().Add(auth.TokenExp),
			HTTPOnly: true,
		})

		c.Set("Authorization", "Bearer "+token)

		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"message": "User authorized successfully",
		})
	}
}
