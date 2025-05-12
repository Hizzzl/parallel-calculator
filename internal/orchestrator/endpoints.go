package orchestrator

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"parallel-calculator/internal/auth"
	"parallel-calculator/internal/config"
	"parallel-calculator/internal/db"
	"parallel-calculator/internal/logger"
	"strconv"
	"time"

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

type TaskResponseArgs struct {
	ID            int64         `json:"id"`
	LeftValue     float64       `json:"arg1"`
	RightValue    float64       `json:"arg2"`
	Operator      string        `json:"operation"`
	OperationTime time.Duration `json:"operation_time"`
}

type TaskResponse struct {
	Task TaskResponseArgs `json:"task"`
}

func HandleGetTask(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	logger.LogINFO("Received get task request")

	// Получаем одну готовую к обработке операцию из БД
	readyOp, err := db.GetReadyOperation()
	if err != nil {
		logger.LogERROR(fmt.Sprintf("Failed to get ready operation from DB: %v", err))
		http.Error(w, "Внутренняя ошибка сервера", http.StatusInternalServerError)
		return
	}

	// Если нет готовых операций, возвращаем 404 Not Found
	if readyOp == nil {
		w.WriteHeader(http.StatusNotFound)
		logger.LogINFO("No ready operations found")
		return
	}

	// Обновляем статус операции на "processing"
	err = db.UpdateOperationStatus(readyOp.ID, db.StatusProcessing)
	if err != nil {
		logger.LogERROR(fmt.Sprintf("Failed to update operation status: %v", err))
		http.Error(w, "Внутренняя ошибка сервера", http.StatusInternalServerError)
		return
	}

	// Получаем значения операндов
	var leftValue, rightValue float64

	// Получаем значения аргументов
	leftValue = 0.0
	if readyOp.LeftValue != nil {
		leftValue = *readyOp.LeftValue
	}

	rightValue = 0.0
	if readyOp.RightValue != nil {
		rightValue = *readyOp.RightValue
	}

	logger.LogINFO(fmt.Sprintf("Operation args: left=%v, right=%v", leftValue, rightValue))

	// Определяем время операции в соответствии с конфигурацией
	var operationTime time.Duration
	switch readyOp.Operator {
	case "+":
		operationTime = config.AppConfig.TimeAddition
	case "-":
		operationTime = config.AppConfig.TimeSubtraction
	case "*":
		operationTime = config.AppConfig.TimeMultiplication
	case "/":
		operationTime = config.AppConfig.TimeDivision
	default:
		// Для других операторов используем время умножения как базовое
		operationTime = config.AppConfig.TimeMultiplication
	}
	logger.LogINFO(fmt.Sprintf("Operation time: %v", operationTime))

	// Формируем задачу для обработчика
	taskArgs := TaskResponseArgs{
		ID:            readyOp.ID,
		LeftValue:     leftValue,
		RightValue:    rightValue,
		Operator:      readyOp.Operator,
		OperationTime: operationTime,
	}

	task := TaskResponse{
		Task: taskArgs,
	}

	logger.LogINFO(fmt.Sprintf("Successfully created task response. Task: %v", task))
	err = json.NewEncoder(w).Encode(task)
	if err != nil {
		logger.LogERROR(fmt.Sprintf("Failed to encode task response: %v", err))
		http.Error(w, "Внутренняя ошибка сервера", http.StatusInternalServerError)
	}
}

type TaskResult struct {
	ID     int64   `json:"id"`
	Result float64 `json:"result"`
	Error  string  `json:"error"`
}

func HandlePostTaskResult(w http.ResponseWriter, r *http.Request) {
	logger.LogINFO("Received post task result request")
	var result TaskResult
	err := json.NewDecoder(r.Body).Decode(&result)
	if err != nil {
		logger.LogERROR(fmt.Sprintf("Failed to decode task result: %v", err))
		http.Error(w, "Неверный формат запроса", http.StatusUnprocessableEntity)
		return
	}
	defer r.Body.Close()

	logger.LogINFO(fmt.Sprintf("Successfully decoded task result: %v. Task id: %v", result, result.ID))

	// Обрабатываем результат операции
	err = ProcessExpressionResult(result)
	if err != nil {
		logger.LogERROR(fmt.Sprintf("Failed to process task result: %v", err))
		http.Error(w, "Внутренняя ошибка сервера", http.StatusInternalServerError)
		return
	}

	// Проверяем, является ли операция корневой для выражения
	op, err := db.GetOperationByID(result.ID)
	if err != nil {
		logger.LogERROR(fmt.Sprintf("Failed to get operation: %v", err))
		http.Error(w, "Внутренняя ошибка сервера", http.StatusInternalServerError)
		return
	}

	// Если операция корневая, обновляем результат всего выражения
	if op.IsRootExpression {
		logger.LogINFO("Root operation completed, updating expression result")
		err = db.SetExpressionResult(op.ExpressionID, result.Result)
		if err != nil {
			logger.LogERROR(fmt.Sprintf("Failed to update expression result: %v", err))
			http.Error(w, "Внутренняя ошибка сервера", http.StatusInternalServerError)
			return
		}

		// Обновляем статус выражения на "completed"
		err = db.UpdateExpressionStatus(op.ExpressionID, db.StatusCompleted)
		if err != nil {
			logger.LogERROR(fmt.Sprintf("Failed to update expression status: %v", err))
			http.Error(w, "Внутренняя ошибка сервера", http.StatusInternalServerError)
			return
		}
	}

	logger.LogINFO(fmt.Sprintf("Successfully processed task result. Task ID: %v, Result: %v", result.ID, result.Result))
	w.WriteHeader(http.StatusOK)
}
