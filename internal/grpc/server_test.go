package grpc

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"parallel-calculator/internal/config"
	"parallel-calculator/internal/db"
	"parallel-calculator/proto"

	_ "github.com/mattn/go-sqlite3"
)

// TestOrchestratorService_GetTask проверяет обработку запроса на получение задачи
func TestOrchestratorService_GetTask(t *testing.T) {
	// Инициализируем конфигурацию
	config.InitConfig("../../.env")

	// Инициализируем тестовую базу данных в памяти
	db.DB, _ = sql.Open("sqlite3", "file:memdb1?mode=memory&cache=shared")
	db.ApplySchema(filepath.Join("../../internal/db", "schema.sql"))

	defer db.CloseDB()

	// Создаем тестовый сервис
	service := &OrchestratorService{}

	// Тестируем различные сценарии
	t.Run("No ready tasks", func(t *testing.T) {
		// Создаем пользователя
		user, err := db.CreateUser("test_noops", "password")
		if err != nil {
			t.Fatalf("Failed to create test user: %v", err)
		}

		// Создаем выражение, но не операции
		_, err = db.CreateExpression(user.ID, "1+1")
		if err != nil {
			t.Fatalf("Failed to create test expression: %v", err)
		}

		// Должен вернуть пустой ответ
		resp, err := service.GetTask(context.Background(), &proto.GetTaskRequest{})

		if err != nil {
			t.Fatalf("GetTask failed: %v", err)
		}

		if resp.HasTask {
			t.Error("Expected no task, but got one")
		}
	})

	t.Run("Ready task exists", func(t *testing.T) {
		// Создаем пользователя
		user, err := db.CreateUser("test_with_ops", "password")
		if err != nil {
			t.Fatalf("Failed to create test user: %v", err)
		}

		// Создаем выражение и операцию
		expr, err := db.CreateExpression(user.ID, "2*3")
		if err != nil {
			t.Fatalf("Failed to create test expression: %v", err)
		}

		leftValue := 2.0
		rightValue := 3.0
		var parentID *int64 = nil
		op, err := db.CreateOperation(
			expr.ID,
			parentID,
			"*",
			&leftValue,
			&rightValue,
			false,
			nil,
			db.StatusReady,
		)
		if err != nil {
			t.Fatalf("Failed to create test operation: %v", err)
		}

		// Получаем задачу
		resp, err := service.GetTask(context.Background(), &proto.GetTaskRequest{})

		if err != nil {
			t.Fatalf("GetTask failed: %v", err)
		}

		if !resp.HasTask {
			t.Error("Expected to get a task, but got none")
			return
		}

		if resp.Id != uint32(op.ID) {
			t.Errorf("Expected task ID %d, got %d", op.ID, resp.Id)
		}

		if resp.Operator != "*" {
			t.Errorf("Expected operator '*', got '%s'", resp.Operator)
		}

		if resp.LeftValue != leftValue {
			t.Errorf("Expected left value %f, got %f", leftValue, resp.LeftValue)
		}

		if resp.RightValue != rightValue {
			t.Errorf("Expected right value %f, got %f", rightValue, resp.RightValue)
		}

		// Проверяем, что статус обновился на "в процессе"
		updatedOp, err := db.GetOperationByID(op.ID)
		if err != nil {
			t.Fatalf("Failed to get updated operation: %v", err)
		}

		if updatedOp.Status != db.StatusProcessing {
			t.Errorf("Expected status to be %s, got %s", db.StatusProcessing, updatedOp.Status)
		}
	})
}

// TestOrchestratorService_SendTaskResult проверяет обработку результатов задач
func TestOrchestratorService_SendTaskResult(t *testing.T) {
	// Инициализируем конфигурацию
	config.InitConfig("../../.env")

	// Инициализируем тестовую базу данных в памяти
	db.DB, _ = sql.Open("sqlite3", "file:memdb1?mode=memory&cache=shared")
	db.ApplySchema(filepath.Join("../../internal/db", "schema.sql"))

	defer db.CloseDB()

	// Создаем тестовый сервис
	service := &OrchestratorService{}

	// Тестируем различные сценарии
	t.Run("Successful result", func(t *testing.T) {
		// Создаем пользователя
		user, err := db.CreateUser("test_result", "password")
		if err != nil {
			t.Fatalf("Failed to create test user: %v", err)
		}

		// Создаем выражение и операцию
		expr, err := db.CreateExpression(user.ID, "2*3")
		if err != nil {
			t.Fatalf("Failed to create test expression: %v", err)
		}

		leftValue := 2.0
		rightValue := 3.0
		var parentID *int64 = nil
		op, err := db.CreateOperation(
			expr.ID,
			parentID,
			"*",
			&leftValue,
			&rightValue,
			true, // Корневая операция
			nil,
			db.StatusProcessing,
		)
		if err != nil {
			t.Fatalf("Failed to create test operation: %v", err)
		}

		// Отправляем результат
		result := 6.0 // 2 * 3 = 6
		resp, err := service.SendTaskResult(context.Background(), &proto.TaskResultRequest{
			Id:     uint32(op.ID),
			Result: result,
			Error:  "",
		})

		if err != nil {
			t.Fatalf("SendTaskResult failed: %v", err)
		}

		if !resp.Success {
			t.Errorf("Expected success, but got error: %s", resp.Error)
		}

		// Проверяем, что операция обновилась
		updatedOp, err := db.GetOperationByID(op.ID)
		if err != nil {
			t.Fatalf("Failed to get updated operation: %v", err)
		}

		if updatedOp.Status != db.StatusCompleted {
			t.Errorf("Expected status to be %s, got %s", db.StatusCompleted, updatedOp.Status)
		}

		if updatedOp.Result == nil {
			t.Fatal("Expected result to be set, but it's nil")
		}

		if *updatedOp.Result != result {
			t.Errorf("Expected result %f, got %f", result, *updatedOp.Result)
		}

		// Проверяем, что выражение обновилось (так как операция была корневой)
		updatedExpr, err := db.GetExpressionByID(expr.ID)
		if err != nil {
			t.Fatalf("Failed to get updated expression: %v", err)
		}

		if updatedExpr.Status != db.StatusCompleted {
			t.Errorf("Expected expression status to be %s, got %s", db.StatusCompleted, updatedExpr.Status)
		}

		if updatedExpr.Result == nil {
			t.Fatal("Expected expression result to be set, but it's nil")
		}

		if *updatedExpr.Result != result {
			t.Errorf("Expected expression result %f, got %f", result, *updatedExpr.Result)
		}
	})

	t.Run("Error result", func(t *testing.T) {
		// Создаем пользователя
		user, err := db.CreateUser("test_error", "password")
		if err != nil {
			t.Fatalf("Failed to create test user: %v", err)
		}

		// Создаем выражение и операцию
		expr, err := db.CreateExpression(user.ID, "2/0")
		if err != nil {
			t.Fatalf("Failed to create test expression: %v", err)
		}

		leftValue := 2.0
		rightValue := 0.0
		var parentID *int64 = nil
		op, err := db.CreateOperation(
			expr.ID,
			parentID,
			"/",
			&leftValue,
			&rightValue,
			true, // Корневая операция
			nil,
			db.StatusProcessing,
		)
		if err != nil {
			t.Fatalf("Failed to create test operation: %v", err)
		}

		// Отправляем результат с ошибкой
		errorMsg := "division by zero"
		resp, err := service.SendTaskResult(context.Background(), &proto.TaskResultRequest{
			Id:     uint32(op.ID),
			Result: 0,
			Error:  errorMsg,
		})

		if err != nil {
			t.Fatalf("SendTaskResult failed: %v", err)
		}

		if !resp.Success {
			t.Errorf("Expected success even for error result, but got error: %s", resp.Error)
		}

		// Проверяем, что операция обновилась
		updatedOp, err := db.GetOperationByID(op.ID)
		if err != nil {
			t.Fatalf("Failed to get updated operation: %v", err)
		}

		if updatedOp.Status != db.StatusCanceled {
			t.Errorf("Expected status to be %s, got %s", db.StatusCanceled, updatedOp.Status)
		}

		// Проверяем, что выражение обновилось со статусом ошибки
		updatedExpr, err := db.GetExpressionByID(expr.ID)
		if err != nil {
			t.Fatalf("Failed to get updated expression: %v", err)
		}

		if updatedExpr.Status != db.StatusError {
			t.Errorf("Expected expression status to be %s, got %s", db.StatusError, updatedExpr.Status)
		}

		if updatedExpr.ErrorMessage == nil {
			t.Fatal("Expected expression error message to be set, but it's nil")
		}

		if *updatedExpr.ErrorMessage != errorMsg {
			t.Errorf("Expected expression error message '%s', got '%s'", errorMsg, *updatedExpr.ErrorMessage)
		}
	})
}

// TestStartGRPCServer проверяет запуск gRPC сервера
func TestStartGRPCServer(t *testing.T) {
	// Инициализируем конфигурацию
	config.InitConfig("../../.env")

	// Используем временный порт для теста
	port := ":0"

	// Запускаем сервер
	server, err := StartGRPCServer(port)
	if err != nil {
		t.Fatalf("Failed to start gRPC server: %v", err)
	}

	// Проверяем, что сервер запустился
	if server == nil {
		t.Error("Server is nil")
	}

	// Останавливаем сервер
	server.Stop()

	// Даем время на graceful shutdown
	time.Sleep(100 * time.Millisecond)
}
