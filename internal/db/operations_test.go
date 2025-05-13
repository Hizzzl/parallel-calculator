package db

import (
	"database/sql"
	"path/filepath"
	"testing"
)

func InitTest(t *testing.T) {
	DB, _ = sql.Open("sqlite3", ":memory:")

	ApplySchema(filepath.Join("../../internal/db", "schema.sql"))
}

// TestCreateOperation проверяет создание операции
func TestCreateOperation(t *testing.T) {
	InitTest(t)

	defer CleanupDB()

	// Создаем тестового пользователя
	user, err := CreateUser("testuser", "testpass")
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	// Создаем тестовое выражение
	expr, err := CreateExpression(user.ID, "5+5")
	if err != nil {
		t.Fatalf("Failed to create test expression: %v", err)
	}

	// Тестовые случаи для CreateOperation
	tests := []struct {
		name          string
		expressionID  int64
		parentOpID    *int64
		operator      string
		leftValue     *float64
		rightValue    *float64
		isRoot        bool
		childPosition *string
		status        string
		expectError   bool
	}{
		{
			name:          "Root operation with left value",
			expressionID:  expr.ID,
			parentOpID:    nil,
			operator:      "+",
			leftValue:     floatPtr(5.0),
			rightValue:    nil,
			isRoot:        true,
			childPosition: nil,
			status:        StatusPending,
			expectError:   false,
		},
		{
			name:          "Child operation with both values",
			expressionID:  expr.ID,
			parentOpID:    intPtr(1), // ID первой операции (предполагается, что будет создана)
			operator:      "*",
			leftValue:     floatPtr(2.0),
			rightValue:    floatPtr(3.0),
			isRoot:        false,
			childPosition: strPtr("left"),
			status:        StatusReady,
			expectError:   false,
		},
	}

	// Создаем первую операцию для теста (чтобы использовать её ID во втором тесткейсе)
	firstOp, err := CreateOperation(
		expr.ID,
		nil,
		"+",
		floatPtr(5.0),
		nil,
		true,
		nil,
		StatusPending,
	)
	if err != nil {
		t.Fatalf("Failed to create first operation: %v", err)
	}

	// Обновляем parentOpID для второго тесткейса
	tests[1].parentOpID = &firstOp.ID

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			op, err := CreateOperation(
				tt.expressionID,
				tt.parentOpID,
				tt.operator,
				tt.leftValue,
				tt.rightValue,
				tt.isRoot,
				tt.childPosition,
				tt.status,
			)

			if (err != nil) != tt.expectError {
				t.Errorf("CreateOperation() error = %v, expectError = %v", err, tt.expectError)
				return
			}

			if !tt.expectError {
				if op == nil {
					t.Errorf("CreateOperation() returned nil operation")
					return
				}

				// Проверяем, что поля операции установлены правильно
				if op.ExpressionID != tt.expressionID {
					t.Errorf("ExpressionID = %v, want %v", op.ExpressionID, tt.expressionID)
				}

				if op.Operator != tt.operator {
					t.Errorf("Operator = %v, want %v", op.Operator, tt.operator)
				}

				if op.IsRootExpression != tt.isRoot {
					t.Errorf("IsRootExpression = %v, want %v", op.IsRootExpression, tt.isRoot)
				}

				if op.Status != tt.status {
					t.Errorf("Status = %v, want %v", op.Status, tt.status)
				}
			}
		})
	}
}

// TestSetOperationResult проверяет установку результата операции
func TestSetOperationResult(t *testing.T) {
	// Инициализируем конфигурацию

	InitTest(t)

	defer CleanupDB()

	// Создаем тестового пользователя
	user, err := CreateUser("testuser", "testpass")
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	// Создаем тестовое выражение
	expr, err := CreateExpression(user.ID, "5+5")
	if err != nil {
		t.Fatalf("Failed to create test expression: %v", err)
	}

	// Создаем операцию для тестирования
	op, err := CreateOperation(
		expr.ID,
		nil,
		"+",
		floatPtr(5.0),
		floatPtr(5.0),
		true,
		nil,
		StatusPending,
	)
	if err != nil {
		t.Fatalf("Failed to create test operation: %v", err)
	}

	// Тестируем установку результата
	result := 10.0
	err = SetOperationResult(op.ID, result)
	if err != nil {
		t.Errorf("SetOperationResult() error = %v", err)
	}

	// Проверяем, что результат был установлен
	updatedOp, err := GetOperationByID(op.ID)
	if err != nil {
		t.Errorf("GetOperationByID() error = %v", err)
		return
	}

	if updatedOp.Result == nil || *updatedOp.Result != result {
		t.Errorf("Operation result = %v, want %v", updatedOp.Result, result)
	}
}

// TestSetOperationError проверяет установку ошибки операции
func TestSetOperationError(t *testing.T) {
	// Инициализируем конфигурацию
	InitTest(t)

	defer CleanupDB()

	// Создаем тестового пользователя
	user, err := CreateUser("testuser", "testpass")
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	// Создаем тестовое выражение
	expr, err := CreateExpression(user.ID, "5+5")
	if err != nil {
		t.Fatalf("Failed to create test expression: %v", err)
	}

	// Создаем операцию для тестирования
	op, err := CreateOperation(
		expr.ID,
		nil,
		"+",
		floatPtr(5.0),
		floatPtr(5.0),
		true,
		nil,
		StatusPending,
	)
	if err != nil {
		t.Fatalf("Failed to create test operation: %v", err)
	}

	// Тестируем установку ошибки
	errorMsg := "Test error message"
	err = SetOperationError(op.ID, errorMsg)
	if err != nil {
		t.Errorf("SetOperationError() error = %v", err)
	}

	// Проверяем, что ошибка была установлена и статус изменен
	updatedOp, err := GetOperationByID(op.ID)
	if err != nil {
		t.Errorf("GetOperationByID() error = %v", err)
		return
	}

	if updatedOp.Status != StatusError {
		t.Errorf("Operation status = %v, want %v", updatedOp.Status, StatusError)
	}

	if updatedOp.ErrorMessage == nil || *updatedOp.ErrorMessage != errorMsg {
		t.Errorf("Operation error message = %v, want %v", updatedOp.ErrorMessage, errorMsg)
	}
}

// TestCancelOperationsByExpressionID проверяет отмену операций по expressionID
func TestCancelOperationsByExpressionID(t *testing.T) {
	// Инициализируем конфигурацию
	InitTest(t)

	defer CleanupDB()

	// Создаем тестового пользователя
	user, err := CreateUser("testuser", "testpass")
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	// Создаем тестовое выражение
	expr, err := CreateExpression(user.ID, "5+5+5")
	if err != nil {
		t.Fatalf("Failed to create test expression: %v", err)
	}

	// Создаем несколько операций для выражения
	op1, err := CreateOperation(expr.ID, nil, "+", floatPtr(5.0), nil, true, nil, StatusPending)
	if err != nil {
		t.Fatalf("Failed to create operation 1: %v", err)
	}

	_, err = CreateOperation(expr.ID, &op1.ID, "+", floatPtr(5.0), floatPtr(5.0), false, strPtr("left"), StatusPending)
	if err != nil {
		t.Fatalf("Failed to create operation 2: %v", err)
	}

	_, err = CreateOperation(expr.ID, &op1.ID, "+", floatPtr(5.0), nil, false, strPtr("right"), StatusPending)
	if err != nil {
		t.Fatalf("Failed to create operation 3: %v", err)
	}

	// Отменяем операции
	err = CancelOperationsByExpressionID(expr.ID)
	if err != nil {
		t.Errorf("CancelOperationsByExpressionID() error = %v", err)
	}

	// Проверяем, что все операции были отменены
	operations, err := GetOperationsByExpressionID(expr.ID)
	if err != nil {
		t.Errorf("GetOperationsByExpressionID() error = %v", err)
		return
	}

	for _, op := range operations {
		if op.Status != StatusCanceled {
			t.Errorf("Operation status = %v, want %v", op.Status, StatusCanceled)
		}
	}
}

// TestUpdateOperationStatus проверяет обновление статуса операции
func TestUpdateOperationStatus(t *testing.T) {
	// Инициализируем конфигурацию
	InitTest(t)

	defer CleanupDB()

	// Создаем тестового пользователя
	user, err := CreateUser("testuser", "testpass")
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	// Создаем тестовое выражение
	expr, err := CreateExpression(user.ID, "5+5")
	if err != nil {
		t.Fatalf("Failed to create test expression: %v", err)
	}

	// Создаем операцию для тестирования
	op, err := CreateOperation(
		expr.ID,
		nil,
		"+",
		floatPtr(5.0),
		floatPtr(5.0),
		true,
		nil,
		StatusPending,
	)
	if err != nil {
		t.Fatalf("Failed to create test operation: %v", err)
	}

	// Тестируем обновление статуса
	newStatus := StatusReady
	err = UpdateOperationStatus(op.ID, newStatus)
	if err != nil {
		t.Errorf("UpdateOperationStatus() error = %v", err)
	}

	// Проверяем, что статус был обновлен
	updatedOp, err := GetOperationByID(op.ID)
	if err != nil {
		t.Errorf("GetOperationByID() error = %v", err)
		return
	}

	if updatedOp.Status != newStatus {
		t.Errorf("Operation status = %v, want %v", updatedOp.Status, newStatus)
	}
}

// TestGetReadyOperation проверяет получение одной готовой операции
func TestGetReadyOperation(t *testing.T) {
	// Инициализируем конфигурацию
	InitTest(t)

	defer CleanupDB()

	// Создаем тестового пользователя
	user, err := CreateUser("testuser", "testpass")
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	// Создаем тестовое выражение
	expr, err := CreateExpression(user.ID, "5+5")
	if err != nil {
		t.Fatalf("Failed to create test expression: %v", err)
	}

	// Создаем операции с разными статусами
	_, err = CreateOperation(expr.ID, nil, "+", floatPtr(5.0), floatPtr(5.0), true, nil, StatusPending)
	if err != nil {
		t.Fatalf("Failed to create operation 1: %v", err)
	}

	op2, err := CreateOperation(expr.ID, nil, "*", floatPtr(2.0), floatPtr(3.0), false, nil, StatusReady)
	if err != nil {
		t.Fatalf("Failed to create operation 2: %v", err)
	}

	_, err = CreateOperation(expr.ID, nil, "-", floatPtr(10.0), floatPtr(5.0), false, nil, StatusCompleted)
	if err != nil {
		t.Fatalf("Failed to create operation 3: %v", err)
	}

	// Получаем одну готовую операцию
	readyOp, err := GetReadyOperation()
	if err != nil {
		t.Errorf("GetReadyOperation() error = %v", err)
		return
	}

	// Проверяем, что мы получили операцию
	if readyOp == nil {
		t.Errorf("GetReadyOperation() returned nil operation, expected an operation with status = %v", StatusReady)
		return
	}

	// Проверяем, что операция имеет статус StatusReady
	if readyOp.Status != StatusReady {
		t.Errorf("Operation status = %v, want %v", readyOp.Status, StatusReady)
	}

	// По LIMIT 1 в SQL-запросе функция должна вернуть первую готовую операцию,
	// в нашем случае это должна быть op2
	if readyOp.ID != op2.ID {
		t.Errorf("GetReadyOperation() returned operation with ID = %d, want %d", readyOp.ID, op2.ID)
	}
}

// TestGetOperationInfo проверяет получение информации об операции
func TestGetOperationInfo(t *testing.T) {
	// Инициализируем конфигурацию
	InitTest(t)

	defer CleanupDB()

	// Создаем тестового пользователя
	user, err := CreateUser("testuser", "testpass")
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	// Создаем тестовое выражение
	expr, err := CreateExpression(user.ID, "5+5")
	if err != nil {
		t.Fatalf("Failed to create test expression: %v", err)
	}

	// Создаем операцию для тестирования
	op, err := CreateOperation(
		expr.ID,
		nil,
		"+",
		floatPtr(5.0),
		floatPtr(5.0),
		true,
		nil,
		StatusPending,
	)
	if err != nil {
		t.Fatalf("Failed to create test operation: %v", err)
	}

	// Тестируем получение информации об операции
	info, err := GetOperationInfo(op.ID)
	if err != nil {
		t.Errorf("GetOperationInfo() error = %v", err)
		return
	}

	// Проверяем, что информация об операции совпадает с ожидаемой
	if info.ID != op.ID {
		t.Errorf("Operation ID = %v, want %v", info.ID, op.ID)
	}

	if info.ExpressionID != op.ExpressionID {
		t.Errorf("Expression ID = %v, want %v", info.ExpressionID, op.ExpressionID)
	}

	if info.IsRootExpression != op.IsRootExpression {
		t.Errorf("IsRootExpression = %v, want %v", info.IsRootExpression, op.IsRootExpression)
	}

	if info.Status != op.Status {
		t.Errorf("Status = %v, want %v", info.Status, op.Status)
	}

	// Проверяем получение информации о несуществующей операции
	_, err = GetOperationInfo(999999)
	if err == nil {
		t.Errorf("GetOperationInfo() for non-existent operation did not return error")
	}
}

// TestUpdateOperation проверяет обновление операции
func TestUpdateOperation(t *testing.T) {
	// Инициализируем конфигурацию
	InitTest(t)

	defer CleanupDB()

	// Создаем тестового пользователя
	user, err := CreateUser("testuser", "testpass")
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	// Создаем тестовое выражение
	expr, err := CreateExpression(user.ID, "5+5")
	if err != nil {
		t.Fatalf("Failed to create test expression: %v", err)
	}

	// Создаем операцию для тестирования
	op, err := CreateOperation(
		expr.ID,
		nil,
		"+",
		floatPtr(5.0),
		floatPtr(5.0),
		true,
		nil,
		StatusPending,
	)
	if err != nil {
		t.Fatalf("Failed to create test operation: %v", err)
	}

	// Изменяем параметры операции
	op.Status = StatusReady
	op.LeftValue = floatPtr(10.0)
	op.RightValue = floatPtr(15.0)
	op.Operator = "*"
	error := strPtr("Test error message")
	op.ErrorMessage = error

	// Тестируем обновление операции
	err = UpdateOperation(op)
	if err != nil {
		t.Errorf("UpdateOperation() error = %v", err)
		return
	}

	// Получаем обновленную операцию и проверяем параметры
	updatedOp, err := GetOperationByID(op.ID)
	if err != nil {
		t.Errorf("GetOperationByID() error = %v", err)
		return
	}

	// Проверяем, что операция была обновлена корректно
	if updatedOp.Status != op.Status {
		t.Errorf("Status = %v, want %v", updatedOp.Status, op.Status)
	}

	if *updatedOp.LeftValue != *op.LeftValue {
		t.Errorf("LeftValue = %v, want %v", *updatedOp.LeftValue, *op.LeftValue)
	}

	if *updatedOp.RightValue != *op.RightValue {
		t.Errorf("RightValue = %v, want %v", *updatedOp.RightValue, *op.RightValue)
	}

	if updatedOp.Operator != op.Operator {
		t.Errorf("Operator = %v, want %v", updatedOp.Operator, op.Operator)
	}

	if *updatedOp.ErrorMessage != *op.ErrorMessage {
		t.Errorf("ErrorMessage = %v, want %v", *updatedOp.ErrorMessage, *op.ErrorMessage)
	}

	// Проверяем обновление несуществующей операции
	invalidOp := &Operation{
		ID:       999999,
		Status:   StatusError,
		Operator: "+",
	}

	err = UpdateOperation(invalidOp)
	// Ожидаем, что хотя операция не существует, ошибки не будет
	if err != nil {
		t.Errorf("UpdateOperation() for non-existent operation error = %v", err)
	}
}

// TestUpdateOperationLeftValue проверяет обновление левого операнда операции
func TestUpdateOperationLeftValue(t *testing.T) {
	// Инициализируем конфигурацию
	InitTest(t)

	defer CleanupDB()

	// Создаем тестового пользователя
	user, err := CreateUser("testuser", "testpass")
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	// Создаем тестовое выражение
	expr, err := CreateExpression(user.ID, "5+5")
	if err != nil {
		t.Fatalf("Failed to create test expression: %v", err)
	}

	// Создаем операцию для тестирования
	initialLeftValue := 5.0
	op, err := CreateOperation(
		expr.ID,
		nil,
		"+",
		floatPtr(initialLeftValue),
		floatPtr(5.0),
		true,
		nil,
		StatusPending,
	)
	if err != nil {
		t.Fatalf("Failed to create test operation: %v", err)
	}

	// Обновляем левый операнд
	newLeftValue := 10.0
	err = UpdateOperationLeftValue(op.ID, newLeftValue)
	if err != nil {
		t.Errorf("UpdateOperationLeftValue() error = %v", err)
		return
	}

	// Получаем обновленную операцию и проверяем значение левого операнда
	updatedOp, err := GetOperationByID(op.ID)
	if err != nil {
		t.Errorf("GetOperationByID() error = %v", err)
		return
	}

	if updatedOp.LeftValue == nil {
		t.Errorf("LeftValue is nil, want %v", newLeftValue)
		return
	}

	if *updatedOp.LeftValue != newLeftValue {
		t.Errorf("LeftValue = %v, want %v", *updatedOp.LeftValue, newLeftValue)
	}

	// Проверяем обновление несуществующей операции
	err = UpdateOperationLeftValue(999999, 42.0)
	// Ожидаем, что хотя операция не существует, ошибки не будет
	if err != nil {
		t.Errorf("UpdateOperationLeftValue() for non-existent operation error = %v", err)
	}
}

// TestUpdateOperationRightValue проверяет обновление правого операнда операции
func TestUpdateOperationRightValue(t *testing.T) {
	// Инициализируем конфигурацию
	InitTest(t)

	defer CleanupDB()

	// Создаем тестового пользователя
	user, err := CreateUser("testuser", "testpass")
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	// Создаем тестовое выражение
	expr, err := CreateExpression(user.ID, "5+5")
	if err != nil {
		t.Fatalf("Failed to create test expression: %v", err)
	}

	// Создаем операцию для тестирования
	initialRightValue := 5.0
	op, err := CreateOperation(
		expr.ID,
		nil,
		"+",
		floatPtr(10.0),
		floatPtr(initialRightValue),
		true,
		nil,
		StatusPending,
	)
	if err != nil {
		t.Fatalf("Failed to create test operation: %v", err)
	}

	// Обновляем правый операнд
	newRightValue := 15.0
	err = UpdateOperationRightValue(op.ID, newRightValue)
	if err != nil {
		t.Errorf("UpdateOperationRightValue() error = %v", err)
		return
	}

	// Получаем обновленную операцию и проверяем значение правого операнда
	updatedOp, err := GetOperationByID(op.ID)
	if err != nil {
		t.Errorf("GetOperationByID() error = %v", err)
		return
	}

	if updatedOp.RightValue == nil {
		t.Errorf("RightValue is nil, want %v", newRightValue)
		return
	}

	if *updatedOp.RightValue != newRightValue {
		t.Errorf("RightValue = %v, want %v", *updatedOp.RightValue, newRightValue)
	}

	// Проверяем обновление несуществующей операции
	err = UpdateOperationRightValue(999999, 42.0)
	// Ожидаем, что хотя операция не существует, ошибки не будет
	if err != nil {
		t.Errorf("UpdateOperationRightValue() for non-existent operation error = %v", err)
	}
}

// Вспомогательные функции для создания указателей на базовые типы
func intPtr(i int64) *int64 {
	return &i
}

func floatPtr(f float64) *float64 {
	return &f
}

func strPtr(s string) *string {
	return &s
}
