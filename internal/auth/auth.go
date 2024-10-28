package auth

import (
	"errors"
	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"github.com/sol1corejz/goferrrmart/internal/logger"
	"go.uber.org/zap"
	"time"
)

type Claims struct {
	jwt.RegisteredClaims
	UserID uuid.UUID
}

var UserID uuid.UUID

const TokenExp = time.Hour * 3
const SecretKey = "supersecretkey"

func GenerateToken(userID uuid.UUID) (string, error) {

	tokenString, err := BuildJWTString(userID)

	if err != nil {
		return "", err
	}

	return tokenString, nil
}

func BuildJWTString(userID uuid.UUID) (string, error) {

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(TokenExp)),
		},

		UserID: userID,
	})

	tokenString, err := token.SignedString([]byte(SecretKey))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

func GetUserID(tokenString string) (uuid.UUID, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
		return []byte(SecretKey), nil
	})

	if err != nil {
		logger.Log.Warn("Error parsing token:", zap.Error(err))
		return uuid.Nil, errors.New("invalid token")
	}

	if !token.Valid {
		logger.Log.Info("Token is not valid")
		return uuid.Nil, errors.New("token is not valid")
	}

	if claims.UserID == uuid.Nil {
		logger.Log.Warn("Parsed UserID is nil")
		return uuid.Nil, errors.New("user ID is nil")
	}

	logger.Log.Info("Token is valid")
	return claims.UserID, nil
}
