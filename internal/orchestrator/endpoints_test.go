package orchestrator_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"parallel-calculator/internal/auth"
	"parallel-calculator/internal/config"
	"parallel-calculator/internal/db"
	"parallel-calculator/internal/orchestrator"
	"strconv"
	"testing"

	"github.com/gorilla/mux"

	_ "github.com/mattn/go-sqlite3" // Необходимо для работы с SQLite
)

// createTestUser создает тестового пользователя и возвращает его ID и токен
func createTestUser(t *testing.T) (int64, string) {
	user, err := db.CreateUser("testuser", "password123")
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	// Настраиваем конфигурацию JWT для тестов
	if config.AppConfig.JWTSecret == "" {
		config.AppConfig.JWTSecret = "test-secret-key"
		config.AppConfig.JWTExpirationMinutes = 60
	}

	// Генерируем JWT-токен для пользователя
	token, err := auth.GenerateToken(user)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	return user.ID, token
}

// TestHandleCalculate проверяет корректную обработку запроса на вычисление
func TestHandleCalculate(t *testing.T) {
	// Инициализируем конфигурацию
	config.InitConfig("../../.env")
	// Устанавливаем базу данных в памяти для тестов

	// Инициализируем базу данных, передаем путь к директории с schema.sql
	err := db.InitDB("../../internal/db")
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}

	// Настраиваем тестовое окружение
	defer db.CleanupDB()

	// Создаем тестового пользователя
	userID, token := createTestUser(t)

	// Тестовые случаи
	tests := []struct {
		name           string
		body           string
		expectedStatus int
		withToken      bool
		invalidToken   bool
	}{
		{
			name:           "Valid expression",
			body:           `{"expression": "2+2"}`,
			expectedStatus: http.StatusCreated,
			withToken:      true,
		},
		{
			name:           "Invalid expression format",
			body:           `{"expression": "2+"}`,
			expectedStatus: http.StatusUnprocessableEntity,
			withToken:      true,
		},
		{
			name:           "Missing token",
			body:           `{"expression": "2+2"}`,
			expectedStatus: http.StatusUnauthorized,
			withToken:      false,
		},
		{
			name:           "Invalid token",
			body:           `{"expression": "2+2"}`,
			expectedStatus: http.StatusUnauthorized,
			withToken:      true,
			invalidToken:   true,
		},
		{
			name:           "Invalid JSON",
			body:           `{"expression": 123}`,
			expectedStatus: http.StatusUnprocessableEntity,
			withToken:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Подготавливаем запрос
			req := httptest.NewRequest("POST", "/calculate", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")

			// Добавляем токен, если нужно
			if tt.withToken {
				tokenString := token
				if tt.invalidToken {
					tokenString = "invalid-token"
				}
				req.Header.Set("Authorization", "Bearer "+tokenString)
			}

			// Подготавливаем ResponseRecorder для получения ответа
			rr := httptest.NewRecorder()

			// Вызываем тестируемый обработчик
			orchestrator.HandleCalculate(rr, req)

			// Проверяем код ответа
			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status code %d, got %d", tt.expectedStatus, rr.Code)
			}

			// Для успешных запросов проверяем содержимое ответа
			if tt.expectedStatus == http.StatusCreated {
				var response orchestrator.CalculateResponse
				if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}

				if response.ID <= 0 {
					t.Errorf("Expected positive ID, got %d", response.ID)
				}

				// Проверяем, что выражение записано в базу данных
				expr, err := db.GetExpressionByID(response.ID)
				if err != nil {
					t.Fatalf("Failed to get expression from DB: %v", err)
				}

				if expr.UserID != userID {
					t.Errorf("Expected expression to be associated with user ID %d, got %d", userID, expr.UserID)
				}
			}
		})
	}
}

// TestHandleGetExpressions проверяет получение списка выражений пользователя
func TestHandleGetExpressions(t *testing.T) {
	// Инициализируем конфигурацию
	config.InitConfig("../../.env")
	// Устанавливаем базу данных в памяти для тестов

	// Инициализируем базу данных, передаем путь к директории с schema.sql
	err := db.InitDB("../../internal/db")
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}

	defer db.CleanupDB()

	// Создаем тестового пользователя
	userID, token := createTestUser(t)

	// Создаем несколько тестовых выражений для пользователя
	expressions := []string{"1+1", "2*2", "3-1"}
	for _, expr := range expressions {
		_, err := orchestrator.ProcessExpression(expr, userID)
		if err != nil {
			t.Fatalf("Failed to create test expression: %v", err)
		}
	}

	// Тестовые случаи
	tests := []struct {
		name           string
		expectedStatus int
		withToken      bool
		invalidToken   bool
		expectedCount  int
	}{
		{
			name:           "Get user expressions",
			expectedStatus: http.StatusOK,
			withToken:      true,
			expectedCount:  len(expressions),
		},
		{
			name:           "Missing token",
			expectedStatus: http.StatusUnauthorized,
			withToken:      false,
		},
		{
			name:           "Invalid token",
			expectedStatus: http.StatusUnauthorized,
			withToken:      true,
			invalidToken:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Подготавливаем запрос
			req := httptest.NewRequest("GET", "/expressions", nil)

			// Добавляем токен, если нужно
			if tt.withToken {
				tokenString := token
				if tt.invalidToken {
					tokenString = "invalid-token"
				}
				req.Header.Set("Authorization", "Bearer "+tokenString)
			}

			// Подготавливаем ResponseRecorder для получения ответа
			rr := httptest.NewRecorder()

			// Вызываем тестируемый обработчик
			orchestrator.HandleGetExpressions(rr, req)

			// Проверяем код ответа
			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status code %d, got %d", tt.expectedStatus, rr.Code)
			}

			// Для успешных запросов проверяем содержимое ответа
			if tt.expectedStatus == http.StatusOK {
				var response []orchestrator.ExpressionResponse
				if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}

				if len(response) != tt.expectedCount {
					t.Errorf("Expected %d expressions, got %d", tt.expectedCount, len(response))
				}
			}
		})
	}
}

// TestHandleGetExpressionByID проверяет получение выражения по ID
func TestHandleGetExpressionByID(t *testing.T) {
	// Инициализируем конфигурацию
	config.InitConfig("../../.env")
	// Устанавливаем базу данных в памяти для тестов

	// Инициализируем базу данных, передаем путь к директории с schema.sql
	err := db.InitDB("../../internal/db")
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}

	defer db.CleanupDB()

	// Создаем тестового пользователя
	userID, token := createTestUser(t)

	// Создаем тестовое выражение
	exprID, err := orchestrator.ProcessExpression("5+5", userID)
	if err != nil {
		t.Fatalf("Failed to create test expression: %v", err)
	}

	// Создаем маршрутизатор Gorilla Mux для тестирования
	router := mux.NewRouter()
	router.HandleFunc("/expressions/{id}", orchestrator.HandleGetExpressionByID)

	// Тестовые случаи
	tests := []struct {
		name           string
		id             string
		expectedStatus int
		withToken      bool
		invalidToken   bool
	}{
		{
			name:           "Get valid expression",
			id:             fmt.Sprintf("%d", *exprID),
			expectedStatus: http.StatusOK,
			withToken:      true,
		},
		{
			name:           "Invalid ID format",
			id:             "invalid",
			expectedStatus: http.StatusBadRequest,
			withToken:      true,
		},
		{
			name:           "Expression not found",
			id:             "999",
			expectedStatus: http.StatusNotFound,
			withToken:      true,
		},
		{
			name:           "Missing token",
			id:             fmt.Sprintf("%d", *exprID),
			expectedStatus: http.StatusUnauthorized,
			withToken:      false,
		},
		{
			name:           "Invalid token",
			id:             fmt.Sprintf("%d", *exprID),
			expectedStatus: http.StatusUnauthorized,
			withToken:      true,
			invalidToken:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Создаем путь для запроса
			path := fmt.Sprintf("/expressions/%s", tt.id)

			// Подготавливаем запрос
			req := httptest.NewRequest("GET", path, nil)

			// Добавляем токен, если нужно
			if tt.withToken {
				tokenString := token
				if tt.invalidToken {
					tokenString = "invalid-token"
				}
				req.Header.Set("Authorization", "Bearer "+tokenString)
			}

			// Прикрепляем переменные маршрута
			var vars = map[string]string{
				"id": tt.id,
			}
			req = mux.SetURLVars(req, vars)

			// Подготавливаем ResponseRecorder для получения ответа
			rr := httptest.NewRecorder()

			// Вызываем тестируемый обработчик
			orchestrator.HandleGetExpressionByID(rr, req)

			// Проверяем код ответа
			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status code %d, got %d", tt.expectedStatus, rr.Code)
			}

			// Для успешных запросов проверяем содержимое ответа
			if tt.expectedStatus == http.StatusOK {
				var response orchestrator.ExpressionResponse
				if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}

				idInt, _ := strconv.ParseInt(tt.id, 10, 64)
				if response.ID != idInt {
					t.Errorf("Expected expression ID %d, got %d", idInt, response.ID)
				}
			}
		})
	}
}

// TestGetUserIDFromToken проверяет извлечение ID пользователя из JWT-токена
func TestGetUserIDFromToken(t *testing.T) {
	// Инициализируем конфигурацию
	config.InitConfig("../../.env")
	// Устанавливаем базу данных в памяти для тестов

	// Инициализируем базу данных, передаем путь к директории с schema.sql
	err := db.InitDB("../../internal/db")
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}

	defer db.CleanupDB()

	// Создаем тестового пользователя
	userID, token := createTestUser(t)

	// Тестовые случаи
	tests := []struct {
		name        string
		token       string
		expectedID  int64
		expectedErr bool
	}{
		{
			name:       "Valid token",
			token:      token,
			expectedID: userID,
		},
		{
			name:        "Invalid token",
			token:       "invalid-token",
			expectedErr: true,
		},
		{
			name:        "Empty token",
			token:       "",
			expectedErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Создаем HTTP запрос с токеном
			req := httptest.NewRequest("GET", "/", nil)
			if tt.token != "" {
				req.Header.Set("Authorization", "Bearer "+tt.token)
			}

			// Извлекаем ID пользователя из токена
			id, err := orchestrator.GetUserIDFromToken(req)

			// Проверяем ошибку
			if (err != nil) != tt.expectedErr {
				t.Errorf("Expected error: %v, got error: %v", tt.expectedErr, err != nil)
			}

			// Проверяем ID пользователя
			if !tt.expectedErr && id != tt.expectedID {
				t.Errorf("Expected user ID %d, got %d", tt.expectedID, id)
			}
		})
	}
}
