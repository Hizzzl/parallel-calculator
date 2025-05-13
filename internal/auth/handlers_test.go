package auth_test

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"parallel-calculator/internal/auth"
	"parallel-calculator/internal/config"
	"parallel-calculator/internal/db"
	"path/filepath"
	"testing"
)

func setupHandlersTest(t *testing.T) {
	// Инициализируем конфигурацию
	config.InitConfig("../../.env")
	config.AppConfig.JWTSecret = "test-secret-key"
	config.AppConfig.JWTExpirationMinutes = 60

	// Инициализируем тестовую базу данных в памяти с общим доступом
	db.DB, _ = sql.Open("sqlite3", "file:memdb1?mode=memory&cache=shared")

	// Применяем схему базы данных
	db.ApplySchema(filepath.Join("../../internal/db", "schema.sql"))
}

// TestRegister проверяет обработчик регистрации
func TestRegister(t *testing.T) {
	setupHandlersTest(t)
	defer db.CloseDB()

	tests := []struct {
		name         string
		reqBody      map[string]string
		expectedCode int
	}{
		{
			name: "Valid registration",
			reqBody: map[string]string{
				"login":    "newuser",
				"password": "newpassword",
			},
			expectedCode: http.StatusOK,
		},
		{
			name: "Empty login",
			reqBody: map[string]string{
				"login":    "",
				"password": "newpassword",
			},
			expectedCode: http.StatusBadRequest,
		},
		{
			name: "Empty password",
			reqBody: map[string]string{
				"login":    "newuser2",
				"password": "",
			},
			expectedCode: http.StatusBadRequest,
		},
		{
			name: "Duplicate user",
			reqBody: map[string]string{
				"login":    "duplicate_user",
				"password": "password123",
			},
			expectedCode: http.StatusOK, // First request succeeds
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Создаем JSON тело запроса
			jsonBody, _ := json.Marshal(tt.reqBody)

			// Создаем HTTP запрос
			req := httptest.NewRequest("POST", "/api/v1/register", bytes.NewBuffer(jsonBody))
			req.Header.Set("Content-Type", "application/json")
			
			// Создаем recorder для захвата ответа
			rr := httptest.NewRecorder()

			// Вызываем обработчик
			auth.Register(rr, req)

			// Проверяем статус ответа
			if rr.Code != tt.expectedCode {
				t.Errorf("Handler returned wrong status code: got %v want %v, response: %s",
					rr.Code, tt.expectedCode, rr.Body.String())
			}
		})
	}

	// Специальный тест для проверки дубликата (второй запрос должен вернуть ошибку)
	t.Run("Duplicate user (second attempt)", func(t *testing.T) {
		jsonBody, _ := json.Marshal(map[string]string{
			"login":    "duplicate_user",
			"password": "password123",
		})

		req := httptest.NewRequest("POST", "/api/v1/register", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		
		rr := httptest.NewRecorder()
		auth.Register(rr, req)

		if rr.Code != http.StatusConflict {
			t.Errorf("Handler returned wrong status code: got %v want %v, response: %s",
				rr.Code, http.StatusConflict, rr.Body.String())
		}
	})
}

// TestLogin проверяет обработчик авторизации
func TestLogin(t *testing.T) {
	setupHandlersTest(t)
	defer db.CloseDB()

	// Создаем тестового пользователя
	_, err := db.CreateUser("testlogin", "testpassword")
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	tests := []struct {
		name         string
		reqBody      map[string]string
		expectedCode int
		checkToken   bool
	}{
		{
			name: "Valid login",
			reqBody: map[string]string{
				"login":    "testlogin",
				"password": "testpassword",
			},
			expectedCode: http.StatusOK,
			checkToken:   true,
		},
		{
			name: "Invalid login",
			reqBody: map[string]string{
				"login":    "nonexistent",
				"password": "password",
			},
			expectedCode: http.StatusUnauthorized,
			checkToken:   false,
		},
		{
			name: "Wrong password",
			reqBody: map[string]string{
				"login":    "testlogin",
				"password": "wrongpassword",
			},
			expectedCode: http.StatusUnauthorized,
			checkToken:   false,
		},
		{
			name: "Empty fields",
			reqBody: map[string]string{
				"login": "",
				"password": "",
			},
			expectedCode: http.StatusUnauthorized, // Пустые поля вызывают ошибку авторизации
			checkToken:   false,
		},
		{
			name: "Invalid JSON format", // Будет использовать специальный запрос с некорректным JSON
			reqBody: nil,
			expectedCode: http.StatusBadRequest,
			checkToken:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req *http.Request
			
			if tt.name == "Invalid JSON format" {
				// Для теста некорректного JSON отправляем невалидный JSON
				invalidJSON := []byte(`{"login":"test", "password" INVALID JSON}`)
				req = httptest.NewRequest("POST", "/api/v1/login", bytes.NewBuffer(invalidJSON))
			} else {
				// Создаем JSON тело запроса
				jsonBody, _ := json.Marshal(tt.reqBody)
				
				// Создаем HTTP запрос
				req = httptest.NewRequest("POST", "/api/v1/login", bytes.NewBuffer(jsonBody))
			}
			
			req.Header.Set("Content-Type", "application/json")
			
			// Создаем recorder для захвата ответа
			rr := httptest.NewRecorder()

			// Вызываем обработчик
			auth.Login(rr, req)

			// Проверяем статус ответа
			if rr.Code != tt.expectedCode {
				t.Errorf("Handler returned wrong status code: got %v want %v, response: %s",
					rr.Code, tt.expectedCode, rr.Body.String())
			}

			// Если ожидается успешный ответ, проверяем токен в ответе
			if tt.checkToken && rr.Code == http.StatusOK {
				var resp struct {
					Token string `json:"token"`
				}
				
				if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
					t.Errorf("Failed to parse response JSON: %v", err)
					return
				}
				
				if resp.Token == "" {
					t.Errorf("Empty token in response")
				}
				
				// Проверяем, что токен действителен
				claims, err := auth.ValidateToken(resp.Token)
				if err != nil {
					t.Errorf("Invalid token in response: %v", err)
					return
				}
				
				if claims.Login != tt.reqBody["login"] {
					t.Errorf("Token contains wrong login: got %v want %v", claims.Login, tt.reqBody["login"])
				}
			}
		})
	}
}
