package orchestrator

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"parallel-calculator/internal/config"
	"parallel-calculator/internal/logger"
	"strconv"
	"time"

	"github.com/gorilla/mux"
)

type CalculateRequest struct {
	Expression string `json:"expression"`
}

type CalculateResponse struct {
	ID uint32 `json:"id"`
}

func HandleCalculate(w http.ResponseWriter, r *http.Request) {
	var request CalculateRequest
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	logger.LogINFO(fmt.Sprintf("Received calculate request: %v", request.Expression))

	id, err := ProcessExpression(request.Expression)
	if err != nil {
		if err == ErrInvalidExpression {
			http.Error(w, err.Error(), http.StatusUnprocessableEntity)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := CalculateResponse{
		ID: id,
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
	ID     uint32  `json:"id"`
	Status string  `json:"status"`
	Result float64 `json:"result"`
}

func HandleGetExpressions(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	logger.LogINFO("Received get expressions request")
	full_expressions, err := ManagerInstance.GetExpressions()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	expressionsResponse := make([]ExpressionResponse, len(full_expressions))
	for i, expression := range full_expressions {
		expressionsResponse[i] = ExpressionResponse{
			ID:     expression.id,
			Status: expression.status,
			Result: expression.result,
		}
	}

	err = json.NewEncoder(w).Encode(expressionsResponse)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
		http.Error(w, "Неверный id", http.StatusInternalServerError)
		return
	}

	logger.LogINFO(fmt.Sprintf("Received get expression by id request: %v", id))

	expression, err := ManagerInstance.GetExpressionById(uint32(id))
	if err != nil {
		if errors.Is(err, ErrExpressionNotFound) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	expressionResponse := ExpressionResponse{
		ID:     expression.id,
		Status: expression.status,
		Result: expression.result,
	}

	err = json.NewEncoder(w).Encode(expressionResponse)
	if err != nil {
		if errors.Is(err, ErrExpressionNotFound) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

type TaskResponseArgs struct {
	ID            uint32        `json:"id"`
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

	taskId, err := ManagerInstance.NextTask()
	if err != nil {
		if errors.Is(err, ErrQueueIsEmpty) {
			logger.LogINFO("Queue is empty")
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		logger.LogERROR(fmt.Sprintf("Failed to get task from manager: %v", err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	logger.LogINFO(fmt.Sprintf("Successfully got task from manager. Received task id: %v", taskId))

	expression, err := ManagerInstance.GetExpressionById(taskId)
	if err != nil {
		logger.LogERROR(fmt.Sprintf("Failed to get expression by id from manager: %v", err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	logger.LogINFO(fmt.Sprintf("Successfully got expression by id from manager. Received expression: %v", expression))

	var operationTime time.Duration
	switch expression.operator {
	case "+":
		operationTime = config.AppConfig.TimeAddition
	case "-":
		operationTime = config.AppConfig.TimeSubtraction
	case "*":
		operationTime = config.AppConfig.TimeMultiplication
	case "/":
		operationTime = config.AppConfig.TimeDivision
	}

	taskArgs := TaskResponseArgs{
		ID:            taskId,
		LeftValue:     <-expression.leftValue,
		RightValue:    <-expression.rightValue,
		Operator:      expression.operator,
		OperationTime: operationTime,
	}
	// Закрываем, так как больше не будет пользоваться этими каналами
	close(expression.leftValue)
	close(expression.rightValue)

	task := TaskResponse{
		Task: taskArgs,
	}

	logger.LogINFO(fmt.Sprintf("Successfully created task response. Task: %v", task))
	err = json.NewEncoder(w).Encode(task)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

type TaskResult struct {
	ID     uint32  `json:"id"`
	Result float64 `json:"result"`
	Error  string  `json:"error"`
}

func HandlePostTaskResult(w http.ResponseWriter, r *http.Request) {
	logger.LogINFO("Received post task result request")
	var result TaskResult
	err := json.NewDecoder(r.Body).Decode(&result)
	if err != nil {
		logger.LogERROR(fmt.Sprintf("Failed to decode task result: %v", err))
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}
	defer r.Body.Close()

	logger.LogINFO(fmt.Sprintf("Successfully decoded task result: %v. Task id: %v", result, result.ID))

	err = ProcessExpressionResult(result)
	if err != nil {
		if errors.Is(err, ErrExpressionNotFound) {
			logger.LogERROR(fmt.Sprintf("Failed to find expression by id: %v", err.Error()))
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		logger.LogERROR(fmt.Sprintf("Failed to update expression result: %v", err.Error()))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	logger.LogINFO(fmt.Sprintf("Successfully updated expression result. Result: %v. Task id: %v", result.Result, result.ID))
	w.WriteHeader(http.StatusOK)
}
