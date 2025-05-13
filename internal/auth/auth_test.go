package auth_test

import (
	"database/sql"
	"net/http"
	"parallel-calculator/internal/auth"
	"parallel-calculator/internal/config"
	"parallel-calculator/internal/db"
	"path/filepath"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func setupTest(t *testing.T) {
	// Инициализируем конфигурацию
	config.InitConfig("../../.env")
	config.AppConfig.JWTSecret = "test-secret-key"
	config.AppConfig.JWTExpirationMinutes = 60

	// Инициализируем тестовую базу данных в памяти с общим доступом
	db.DB, _ = sql.Open("sqlite3", "file:memdb1?mode=memory&cache=shared")

	// Применяем схему базы данных
	db.ApplySchema(filepath.Join("../../internal/db", "schema.sql"))
}

func createTestUser(t *testing.T) *db.User {
	user, err := db.CreateUser("testuser", "testpassword")
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}
	return user
}

// TestGenerateToken проверяет создание JWT токена
func TestGenerateToken(t *testing.T) {
	setupTest(t)
	defer db.CloseDB()

	// Создаем тестового пользователя
	user := createTestUser(t)

	// Генерируем токен
	token, err := auth.GenerateToken(user)
	if err != nil {
		t.Errorf("GenerateToken() error = %v", err)
		return
	}

	// Проверяем, что токен не пустой
	if token == "" {
		t.Errorf("GenerateToken() returned empty token")
		return
	}

	// Валидируем сгенерированный токен
	claims, err := auth.ValidateToken(token)
	if err != nil {
		t.Errorf("ValidateToken() error = %v", err)
		return
	}

	// Проверяем, что утверждения содержат правильный ID пользователя и логин
	if claims.UserID != user.ID {
		t.Errorf("Generated token contains wrong user ID. got = %v, want = %v", claims.UserID, user.ID)
	}

	if claims.Login != user.Login {
		t.Errorf("Generated token contains wrong login. got = %v, want = %v", claims.Login, user.Login)
	}
}

// TestValidateToken проверяет валидацию JWT токена
func TestValidateToken(t *testing.T) {
	setupTest(t)
	defer db.CloseDB()

	user := createTestUser(t)

	tests := []struct {
		name        string
		tokenFunc   func() string
		wantErr     bool
		expectedErr error
	}{
		{
			name: "Valid token",
			tokenFunc: func() string {
				token, _ := auth.GenerateToken(user)
				return token
			},
			wantErr: false,
		},
		{
			name: "Expired token",
			tokenFunc: func() string {
				// Создаем истекший токен
				expirationTime := time.Now().Add(-time.Minute) // 1 минута назад
				claims := &auth.Claims{
					UserID: user.ID,
					Login:  user.Login,
					RegisteredClaims: jwt.RegisteredClaims{
						ExpiresAt: jwt.NewNumericDate(expirationTime),
						IssuedAt:  jwt.NewNumericDate(time.Now().Add(-time.Hour)),
					},
				}
				token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
				tokenString, _ := token.SignedString([]byte(config.AppConfig.JWTSecret))
				return tokenString
			},
			wantErr:     true,
			expectedErr: auth.ErrExpiredToken,
		},
		{
			name: "Invalid token signature",
			tokenFunc: func() string {
				// Создаем токен с неправильной подписью
				expirationTime := time.Now().Add(time.Hour)
				claims := &auth.Claims{
					UserID: user.ID,
					Login:  user.Login,
					RegisteredClaims: jwt.RegisteredClaims{
						ExpiresAt: jwt.NewNumericDate(expirationTime),
						IssuedAt:  jwt.NewNumericDate(time.Now()),
					},
				}
				token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
				tokenString, _ := token.SignedString([]byte("wrong-secret"))
				return tokenString
			},
			wantErr:     true,
			expectedErr: auth.ErrInvalidToken,
		},
		{
			name: "Malformed token",
			tokenFunc: func() string {
				return "malformed.token.string"
			},
			wantErr:     true,
			expectedErr: auth.ErrInvalidToken,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token := tt.tokenFunc()
			claims, err := auth.ValidateToken(token)

			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateToken() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				if err != tt.expectedErr {
					t.Errorf("ValidateToken() expected error = %v, got = %v", tt.expectedErr, err)
				}
				return
			}

			// Для валидного токена проверяем, что claims содержат правильные данные
			if claims.UserID != user.ID {
				t.Errorf("ValidateToken() claims have wrong user ID. got = %v, want = %v", claims.UserID, user.ID)
			}

			if claims.Login != user.Login {
				t.Errorf("ValidateToken() claims have wrong login. got = %v, want = %v", claims.Login, user.Login)
			}
		})
	}
}

// TestExtractTokenFromHeader проверяет извлечение токена из заголовка
func TestExtractTokenFromHeader(t *testing.T) {
	tests := []struct {
		name            string
		authHeaderValue string
		wantToken       string
		wantErr         bool
		expectedErr     error
	}{
		{
			name:            "Valid Bearer token",
			authHeaderValue: "Bearer valid-token-123",
			wantToken:       "valid-token-123",
			wantErr:         false,
		},
		{
			name:            "Missing Authorization header",
			authHeaderValue: "",
			wantToken:       "",
			wantErr:         true,
			expectedErr:     auth.ErrMissingAuthHeader,
		},
		{
			name:            "Invalid format - no Bearer",
			authHeaderValue: "token-123",
			wantToken:       "",
			wantErr:         true,
			expectedErr:     auth.ErrInvalidAuthHeader,
		},
		{
			name:            "Invalid format - no space",
			authHeaderValue: "Bearertoken-123",
			wantToken:       "",
			wantErr:         true,
			expectedErr:     auth.ErrInvalidAuthHeader,
		},
		{
			name:            "Invalid format - wrong prefix",
			authHeaderValue: "Basic token-123",
			wantToken:       "",
			wantErr:         true,
			expectedErr:     auth.ErrInvalidAuthHeader,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Создаем фиктивный HTTP запрос
			req, _ := http.NewRequest("GET", "/", nil)
			if tt.authHeaderValue != "" {
				req.Header.Set("Authorization", tt.authHeaderValue)
			}

			token, err := auth.ExtractTokenFromHeader(req)

			if (err != nil) != tt.wantErr {
				t.Errorf("ExtractTokenFromHeader() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				if err != tt.expectedErr {
					t.Errorf("ExtractTokenFromHeader() expected error = %v, got = %v", tt.expectedErr, err)
				}
				return
			}

			if token != tt.wantToken {
				t.Errorf("ExtractTokenFromHeader() = %v, want %v", token, tt.wantToken)
			}
		})
	}
}

// TestGetUserFromToken проверяет получение пользователя по токену
func TestGetUserFromToken(t *testing.T) {
	setupTest(t)
	defer db.CloseDB()

	// Создаем тестового пользователя
	user := createTestUser(t)

	tests := []struct {
		name        string
		tokenFunc   func() string
		wantErr     bool
		expectedErr error
	}{
		{
			name: "Valid token",
			tokenFunc: func() string {
				token, _ := auth.GenerateToken(user)
				return token
			},
			wantErr: false,
		},
		{
			name: "Invalid token",
			tokenFunc: func() string {
				return "invalid.token.string"
			},
			wantErr:     true,
			expectedErr: auth.ErrInvalidToken,
		},
		{
			name: "Expired token",
			tokenFunc: func() string {
				// Создаем истекший токен
				expirationTime := time.Now().Add(-time.Minute) // 1 минута назад
				claims := &auth.Claims{
					UserID: user.ID,
					Login:  user.Login,
					RegisteredClaims: jwt.RegisteredClaims{
						ExpiresAt: jwt.NewNumericDate(expirationTime),
						IssuedAt:  jwt.NewNumericDate(time.Now().Add(-time.Hour)),
					},
				}
				token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
				tokenString, _ := token.SignedString([]byte(config.AppConfig.JWTSecret))
				return tokenString
			},
			wantErr:     true,
			expectedErr: auth.ErrExpiredToken,
		},
		{
			name: "Non-existent user ID",
			tokenFunc: func() string {
				// Создаем токен с несуществующим ID пользователя
				expirationTime := time.Now().Add(time.Hour)
				claims := &auth.Claims{
					UserID: 999999, // Несуществующий ID
					Login:  "nonexistent",
					RegisteredClaims: jwt.RegisteredClaims{
						ExpiresAt: jwt.NewNumericDate(expirationTime),
						IssuedAt:  jwt.NewNumericDate(time.Now()),
					},
				}
				token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
				tokenString, _ := token.SignedString([]byte(config.AppConfig.JWTSecret))
				return tokenString
			},
			wantErr: true,
			// Ожидаем ошибку от db.GetUserByID, но точный текст зависит от реализации
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token := tt.tokenFunc()
			fetchedUser, err := auth.GetUserFromToken(token)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetUserFromToken() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				if tt.expectedErr != nil && err != tt.expectedErr {
					t.Errorf("GetUserFromToken() expected error = %v, got = %v", tt.expectedErr, err)
				}
				return
			}

			// Для валидного токена проверяем, что получен правильный пользователь
			if fetchedUser.ID != user.ID {
				t.Errorf("GetUserFromToken() returned user with wrong ID. got = %v, want = %v", fetchedUser.ID, user.ID)
			}

			if fetchedUser.Login != user.Login {
				t.Errorf("GetUserFromToken() returned user with wrong login. got = %v, want = %v", fetchedUser.Login, user.Login)
			}
		})
	}
}
