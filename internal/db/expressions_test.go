package db

import (
	"testing"
)

// TestCreateExpression проверяет создание выражения
func TestCreateExpression(t *testing.T) {
	// Инициализируем конфигурацию
	InitTest(t)

	defer CleanupDB()

	// Создаем тестового пользователя
	user, err := CreateUser("testuser", "testpass")
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}
	userID := user.ID

	// Тестовые случаи для CreateExpression
	tests := []struct {
		name        string
		userID      int64
		expression  string
		status      string
		expectError bool
	}{
		{
			name:        "Valid expression",
			userID:      userID,
			expression:  "5+5",
			status:      StatusPending,
			expectError: false,
		},
		{
			name:        "Long expression",
			userID:      userID,
			expression:  "5+5*2-10/2",
			status:      StatusPending,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr, err := CreateExpression(tt.userID, tt.expression)
			if (err != nil) != tt.expectError {
				t.Errorf("CreateExpression() error = %v, expectError = %v", err, tt.expectError)
				return
			}

			if !tt.expectError {
				if expr.ID <= 0 {
					t.Errorf("CreateExpression() returned invalid ID %d", expr.ID)
					return
				}

				// Проверяем, что выражение было создано
				savedExpr, err := GetExpressionByID(expr.ID)
				if err != nil {
					t.Errorf("Failed to get created expression: %v", err)
					return
				}

				if savedExpr.Expression != tt.expression {
					t.Errorf("Expression = %v, want %v", savedExpr.Expression, tt.expression)
				}

				if savedExpr.Status != tt.status {
					t.Errorf("Status = %v, want %v", savedExpr.Status, tt.status)
				}

				if savedExpr.UserID != tt.userID {
					t.Errorf("UserID = %v, want %v", savedExpr.UserID, tt.userID)
				}
			}
		})
	}
}

// TestSetExpressionResult проверяет установку результата выражения
func TestSetExpressionResult(t *testing.T) {
	// Инициализируем конфигурацию
	InitTest(t)

	defer CleanupDB()

	// Создаем тестового пользователя и выражение
	user, err := CreateUser("testuser", "testpass")
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	expr, err := CreateExpression(user.ID, "5+5")
	if err != nil {
		t.Fatalf("Failed to create test expression: %v", err)
	}

	// Тестируем установку результата
	result := 10.0
	err = SetExpressionResult(expr.ID, result)
	if err != nil {
		t.Errorf("SetExpressionResult() error = %v", err)
	}

	// Проверяем, что результат был установлен и статус изменен
	updatedExpr, err := GetExpressionByID(expr.ID)
	if err != nil {
		t.Errorf("GetExpressionByID() error = %v", err)
		return
	}

	if updatedExpr.Status != StatusCompleted {
		t.Errorf("Expression status = %v, want %v", updatedExpr.Status, StatusCompleted)
	}

	if updatedExpr.Result == nil || *updatedExpr.Result != result {
		t.Errorf("Expression result = %v, want %v", updatedExpr.Result, result)
	}
}

// TestSetExpressionError проверяет установку ошибки выражения
func TestSetExpressionError(t *testing.T) {
	// Инициализируем конфигурацию
	InitTest(t)

	defer CleanupDB()

	// Создаем тестового пользователя и выражение
	user, err := CreateUser("testuser", "testpass")
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	expr, err := CreateExpression(user.ID, "5+5")
	if err != nil {
		t.Fatalf("Failed to create test expression: %v", err)
	}

	// Тестируем установку ошибки
	errorMsg := "Test error message"
	err = SetExpressionError(expr.ID, errorMsg)
	if err != nil {
		t.Errorf("SetExpressionError() error = %v", err)
	}

	// Проверяем, что ошибка была установлена и статус изменен
	updatedExpr, err := GetExpressionByID(expr.ID)
	if err != nil {
		t.Errorf("GetExpressionByID() error = %v", err)
		return
	}

	if updatedExpr.Status != StatusError {
		t.Errorf("Expression status = %v, want %v", updatedExpr.Status, StatusError)
	}

	if updatedExpr.ErrorMessage == nil || *updatedExpr.ErrorMessage != errorMsg {
		t.Errorf("Expression error message = %v, want %v", updatedExpr.ErrorMessage, errorMsg)
	}
}

// TestUpdateExpressionStatus проверяет обновление статуса выражения
func TestUpdateExpressionStatus(t *testing.T) {
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

	// Изначально выражение должно иметь статус StatusPending
	if expr.Status != StatusPending {
		t.Errorf("Initial status should be %s, got %s", StatusPending, expr.Status)
	}

	// Тестируем обновление статуса
	newStatus := StatusCompleted
	err = UpdateExpressionStatus(expr.ID, newStatus)
	if err != nil {
		t.Errorf("UpdateExpressionStatus() error = %v", err)
	}

	// Получаем обновленное выражение и проверяем статус
	updatedExpr, err := GetExpressionByID(expr.ID)
	if err != nil {
		t.Errorf("GetExpressionByID() error = %v", err)
		return
	}

	if updatedExpr.Status != newStatus {
		t.Errorf("Expression status = %v, want %v", updatedExpr.Status, newStatus)
	}

	// Проверяем обработку ошибки при обновлении несуществующего выражения
	err = UpdateExpressionStatus(999999, StatusError)
	// Здесь мы не ожидаем ошибки, так как DB.Exec не вернет ошибку, если не найдено строк для обновления
	if err != nil {
		t.Errorf("UpdateExpressionStatus() for non-existent expression error = %v", err)
	}
}

// TestGetUserExpressions проверяет получение выражений пользователя
func TestGetUserExpressions(t *testing.T) {
	// Инициализируем конфигурацию
	InitTest(t)

	defer CleanupDB()

	// Создаем двух тестовых пользователей
	user1, err := CreateUser("testuser1", "testpass1")
	if err != nil {
		t.Fatalf("Failed to create test user 1: %v", err)
	}
	userID1 := user1.ID

	user2, err := CreateUser("testuser2", "testpass2")
	if err != nil {
		t.Fatalf("Failed to create test user 2: %v", err)
	}
	userID2 := user2.ID

	// Создаем несколько выражений для первого пользователя
	for i := 0; i < 3; i++ {
		_, err := CreateExpression(userID1, "5+5")
		if err != nil {
			t.Fatalf("Failed to create expression for user 1: %v", err)
		}
	}

	// Создаем выражение для второго пользователя
	_, err = CreateExpression(userID2, "10*2")
	if err != nil {
		t.Fatalf("Failed to create expression for user 2: %v", err)
	}

	// Проверяем, что GetUserExpressions возвращает правильное количество выражений
	exprs1, err := GetUserExpressions(userID1)
	if err != nil {
		t.Errorf("GetUserExpressions() error = %v", err)
		return
	}

	if len(exprs1) != 3 {
		t.Errorf("GetUserExpressions() for user 1 returned %d expressions, want 3", len(exprs1))
	}

	exprs2, err := GetUserExpressions(userID2)
	if err != nil {
		t.Errorf("GetUserExpressions() error = %v", err)
		return
	}

	if len(exprs2) != 1 {
		t.Errorf("GetUserExpressions() for user 2 returned %d expressions, want 1", len(exprs2))
	}
}
