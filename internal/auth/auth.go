package auth

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"parallel-calculator/internal/config"
	"parallel-calculator/internal/db"
)

var (
	ErrInvalidToken      = errors.New("недействительный токен")
	ErrExpiredToken      = errors.New("истекший токен")
	ErrMissingAuthHeader = errors.New("отсутствует заголовок Authorization")
	ErrInvalidAuthHeader = errors.New("недействительный формат заголовка Authorization")
)

// Claims представляет собой утверждения JWT
type Claims struct {
	UserID int64  `json:"user_id"`
	Login  string `json:"login"`
	jwt.RegisteredClaims
}

// GenerateToken создает JWT токен для пользователя
func GenerateToken(user *db.User) (string, error) {
	// Время истечения токена из конфигурации
	expirationTime := time.Now().Add(time.Duration(config.AppConfig.JWTExpirationMinutes) * time.Minute)

	claims := &Claims{
		UserID: user.ID,
		Login:  user.Login,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	tokenString, err := token.SignedString([]byte(config.AppConfig.JWTSecret))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// ValidateToken проверяет токен и возвращает утверждения, если токен действителен
func ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("неожиданный метод подписи: %v", token.Header["alg"])
		}
		return []byte(config.AppConfig.JWTSecret), nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, ErrInvalidToken
	}

	// Извлекаем утверждения
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

// ExtractTokenFromHeader извлекает токен из заголовка Authorization
func ExtractTokenFromHeader(r *http.Request) (string, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return "", ErrMissingAuthHeader
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		return "", ErrInvalidAuthHeader
	}

	return parts[1], nil
}

// GetUserFromToken получает пользователя на основе токена
func GetUserFromToken(tokenString string) (*db.User, error) {
	claims, err := ValidateToken(tokenString)
	if err != nil {
		return nil, err
	}

	user, err := db.GetUserByID(claims.UserID)
	if err != nil {
		return nil, err
	}

	return user, nil
}
