package orchestrator

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"parallel-calculator/internal/auth"
	"parallel-calculator/internal/db"
	"parallel-calculator/internal/logger"
	"strconv"

	"github.com/gorilla/mux"
)

type CalculateRequest struct {
	Expression string `json:"expression"`
}

type CalculateResponse struct {
	ID int64 `json:"id"`
}

func HandleCalculate(w http.ResponseWriter, r *http.Request) {
	var request CalculateRequest
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		logger.LogERROR(fmt.Sprintf("Failed to decode calculate request: %v", err))
		http.Error(w, "Неверный формат запроса", http.StatusUnprocessableEntity)
		return
	}

	logger.LogINFO(fmt.Sprintf("Received calculate request: %v", request.Expression))

	// Получаем ID пользователя из запроса с JWT-токеном
	userID, err := GetUserIDFromToken(r)
	if err != nil {
		logger.LogERROR(fmt.Sprintf("Authentication error: %v", err))
		http.Error(w, "Ошибка аутентификации", http.StatusUnauthorized)
		return
	}

	// Обрабатываем выражение
	id, err := ProcessExpression(request.Expression, userID)
	if err != nil {
		if err == ErrInvalidExpression {
			logger.LogERROR(fmt.Sprintf("Invalid expression: %v", err))
			http.Error(w, "Неверное выражение", http.StatusUnprocessableEntity)
			return
		}
		logger.LogERROR(fmt.Sprintf("Failed to process expression: %v", err))
		http.Error(w, "Внутренняя ошибка сервера", http.StatusInternalServerError)
		return
	}

	// Возвращаем ID выражения как он в БД
	response := CalculateResponse{
		ID: *id,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

type ExpressionResponse struct {
	ID     int64   `json:"id"`
	Status string  `json:"status"`
	Result float64 `json:"result"`
}

// GetUserIDFromToken извлекает ID пользователя из JWT-токена
func GetUserIDFromToken(r *http.Request) (int64, error) {
	// Извлекаем токен из заголовка
	tokenString, err := auth.ExtractTokenFromHeader(r)
	if err != nil {
		return 0, err
	}

	// Валидируем токен и получаем claims
	claims, err := auth.ValidateToken(tokenString)
	if err != nil {
		return 0, err
	}

	return claims.UserID, nil
}

func HandleGetExpressions(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	logger.LogINFO("Received get expressions request")

	// Получаем ID пользователя из токена
	userID, err := GetUserIDFromToken(r)
	if err != nil {
		logger.LogERROR(fmt.Sprintf("Authentication error: %v", err))
		http.Error(w, "Ошибка аутентификации", http.StatusUnauthorized)
		return
	}

	// Получаем выражения пользователя из базы данных
	expressions, err := db.GetUserExpressions(userID)
	if err != nil {
		logger.LogERROR(fmt.Sprintf("Failed to get expressions: %v", err))
		http.Error(w, "Внутренняя ошибка сервера", http.StatusInternalServerError)
		return
	}

	expressionsResponse := make([]ExpressionResponse, len(expressions))
	for i, expr := range expressions {
		response := ExpressionResponse{
			ID:     expr.ID,
			Status: expr.Status,
		}
		// Добавляем результат, если он существует
		if expr.Result != nil {
			response.Result = *expr.Result
		}
		expressionsResponse[i] = response
	}

	err = json.NewEncoder(w).Encode(expressionsResponse)
	if err != nil {
		logger.LogERROR(fmt.Sprintf("Failed to encode expressions response: %v", err))
		http.Error(w, "Внутренняя ошибка сервера", http.StatusInternalServerError)
		return
	}
}

func HandleGetExpressionByID(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	idStr := vars["id"] // получаем параметр id из URL

	logger.LogINFO(fmt.Sprintf("Received get expression by id request: %v", idStr))
	id, err := strconv.Atoi(idStr)
	if err != nil {
		logger.LogERROR(fmt.Sprintf("Invalid expression ID format: %v", err))
		http.Error(w, "Неверный формат ID", http.StatusBadRequest)
		return
	}

	logger.LogINFO(fmt.Sprintf("Processing get expression by id request: %v", id))

	expression, err := db.GetExpressionByID(int64(id))
	if err != nil {
		if errors.Is(err, ErrExpressionNotFound) {
			logger.LogERROR(fmt.Sprintf("Expression not found: %v", id))
			http.Error(w, "Выражение не найдено", http.StatusNotFound)
			return
		}
		logger.LogERROR(fmt.Sprintf("Failed to get expression by id: %v", err))
		http.Error(w, "Внутренняя ошибка сервера", http.StatusInternalServerError)
		return
	}

	expressionResponse := ExpressionResponse{
		ID:     expression.ID,
		Status: expression.Status,
	}

	// Добавляем результат только если он существует
	if expression.Result != nil {
		expressionResponse.Result = *expression.Result
	}

	err = json.NewEncoder(w).Encode(expressionResponse)
	if err != nil {
		logger.LogERROR(fmt.Sprintf("Failed to encode expression response: %v", err))
		http.Error(w, "Внутренняя ошибка сервера", http.StatusInternalServerError)
		return
	}
}

type TaskResult struct {
	ID     int64   `json:"id"`
	Result float64 `json:"result"`
	Error  string  `json:"error"`
}
