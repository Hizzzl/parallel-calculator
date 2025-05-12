package auth

import (
	"encoding/json"
	"net/http"

	"parallel-calculator/internal/db"
)

// LoginRequest представляет запрос на вход
type LoginRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

// RegisterRequest представляет запрос на регистрацию
type RegisterRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

// AuthResponse представляет ответ на запрос аутентификации
type AuthResponse struct {
	Token string   `json:"token"`
	User  *db.User `json:"user"`
}

// Register обрабатывает запрос на регистрацию
func Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Ошибка при разборе JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Проверяем, что логин и пароль не пустые
	if req.Login == "" || req.Password == "" {
		http.Error(w, "Логин и пароль не могут быть пустыми", http.StatusBadRequest)
		return
	}

	// Создаем пользователя
	_, err := db.CreateUser(req.Login, req.Password)
	if err != nil {
		if err == db.ErrUserAlreadyExists {
			http.Error(w, "Пользователь с таким логином уже существует", http.StatusConflict)
			return
		}
		http.Error(w, "Ошибка при создании пользователя: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Устанавливаем статус 200 OK без дополнительного содержимого
	w.WriteHeader(http.StatusOK)
}

// Login обрабатывает запрос на вход пользователя (POST /api/v1/login)
func Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Ошибка при разборе JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Проверяем учетные данные
	user, err := db.AuthenticateUser(req.Login, req.Password)
	if err != nil {
		if err == db.ErrUserNotFound || err == db.ErrInvalidCredentials {
			http.Error(w, "Неверный логин или пароль", http.StatusUnauthorized)
			return
		}
		http.Error(w, "Ошибка при аутентификации: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Генерируем токен
	token, err := GenerateToken(user)
	if err != nil {
		http.Error(w, "Ошибка при создании токена: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Формируем ответ с токеном
	resp := struct {
		Token string `json:"token"`
	}{
		Token: token,
	}

	// Устанавливаем заголовок Content-Type и статус 200 OK
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}
