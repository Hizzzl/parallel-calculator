package auth_test

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"parallel-calculator/internal/auth"
	"parallel-calculator/internal/config"
	"parallel-calculator/internal/db"
	"path/filepath"
	"testing"
)

func setupMiddlewareTest(t *testing.T) {
	// Инициализируем конфигурацию
	config.InitConfig("../../.env")
	config.AppConfig.JWTSecret = "test-secret-key"
	config.AppConfig.JWTExpirationMinutes = 60

	// Инициализируем тестовую базу данных в памяти с общим доступом
	db.DB, _ = sql.Open("sqlite3", "file:memdb1?mode=memory&cache=shared")

	// Применяем схему базы данных
	db.ApplySchema(filepath.Join("../../internal/db", "schema.sql"))
}

func createMiddlewareTestUser(t *testing.T) *db.User {
	user, err := db.CreateUser("testuser_middleware", "testpassword")
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}
	return user
}

// TestAuthMiddleware проверяет работу middleware для авторизации
func TestAuthMiddleware(t *testing.T) {
	setupMiddlewareTest(t)
	defer db.CloseDB()

	// Создаем тестового пользователя
	user := createMiddlewareTestUser(t)

	// Создаем токен
	token, err := auth.GenerateToken(user)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	tests := []struct {
		name           string
		authHeader     string
		expectedStatus int
		userInContext  bool
	}{
		{
			name:           "Valid token",
			authHeader:     "Bearer " + token,
			expectedStatus: http.StatusOK,
			userInContext:  true,
		},
		{
			name:           "Missing token",
			authHeader:     "",
			expectedStatus: http.StatusUnauthorized,
			userInContext:  false,
		},
		{
			name:           "Invalid token format",
			authHeader:     "Bearer invalid.token.string",
			expectedStatus: http.StatusUnauthorized,
			userInContext:  false,
		},
		{
			name:           "Wrong token prefix",
			authHeader:     "Basic " + token,
			expectedStatus: http.StatusUnauthorized,
			userInContext:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Создаем тестовый обработчик, который проверяет наличие пользователя в контексте
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				claims, ok := auth.GetUserFromContext(r.Context())
				if ok {
					if claims.UserID != user.ID {
						t.Errorf("Expected user ID %d, got %d", user.ID, claims.UserID)
					}
					w.WriteHeader(http.StatusOK)
				} else {
					// Этот код не должен выполняться для успешных тестов, т.к. они не должны проходить через middleware
					w.WriteHeader(http.StatusInternalServerError)
				}
			})

			// Создаем request recorder для захвата ответа
			req := httptest.NewRequest("GET", "/", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			
			rr := httptest.NewRecorder()

			// Применяем middleware
			middleware := auth.AuthMiddleware(handler)
			middleware.ServeHTTP(rr, req)

			// Проверяем статус ответа
			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status code %d, got %d", tt.expectedStatus, rr.Code)
			}
		})
	}
}

// TestGetUserFromContext проверяет извлечение пользователя из контекста
func TestGetUserFromContext(t *testing.T) {
	// Создаем тестовые данные
	testClaims := &auth.Claims{
		UserID: 1,
		Login:  "testuser",
	}

	// Тест на успешное извлечение
	t.Run("Valid context", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), auth.UserContextKey, testClaims)
		claims, ok := auth.GetUserFromContext(ctx)
		
		if !ok {
			t.Errorf("GetUserFromContext() returned not ok")
			return
		}
		
		if claims.UserID != testClaims.UserID {
			t.Errorf("GetUserFromContext() returned user ID = %v, want %v", claims.UserID, testClaims.UserID)
		}
		
		if claims.Login != testClaims.Login {
			t.Errorf("GetUserFromContext() returned login = %v, want %v", claims.Login, testClaims.Login)
		}
	})

	// Тест на отсутствие пользователя в контексте
	t.Run("Empty context", func(t *testing.T) {
		ctx := context.Background()
		_, ok := auth.GetUserFromContext(ctx)
		
		if ok {
			t.Errorf("GetUserFromContext() returned ok for empty context")
		}
	})

	// Тест на некорректный тип данных в контексте
	t.Run("Invalid context type", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), auth.UserContextKey, "not-a-claims-object")
		_, ok := auth.GetUserFromContext(ctx)
		
		if ok {
			t.Errorf("GetUserFromContext() returned ok for invalid context type")
		}
	})
}

// TestRequireAuth проверяет функцию RequireAuth
func TestRequireAuth(t *testing.T) {
	// Создаем тестовые данные
	testClaims := &auth.Claims{
		UserID: 1,
		Login:  "testuser",
	}

	// Тест на успешную авторизацию
	t.Run("Valid auth", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), auth.UserContextKey, testClaims)
		req := httptest.NewRequest("GET", "/", nil).WithContext(ctx)
		
		userID, err := auth.RequireAuth(req)
		
		if err != nil {
			t.Errorf("RequireAuth() returned error: %v", err)
			return
		}
		
		if userID != testClaims.UserID {
			t.Errorf("RequireAuth() returned user ID = %v, want %v", userID, testClaims.UserID)
		}
	})

	// Тест на отсутствие авторизации
	t.Run("No auth", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		
		_, err := auth.RequireAuth(req)
		
		if err != auth.ErrInvalidToken {
			t.Errorf("RequireAuth() returned error = %v, want %v", err, auth.ErrInvalidToken)
		}
	})
}
