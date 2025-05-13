package orchestrator_test

import (
	"database/sql"
	"go/ast"
	"parallel-calculator/internal/config"
	"parallel-calculator/internal/db"
	"parallel-calculator/internal/orchestrator"
	"path/filepath"
	"testing"
)

// initTestDB инициализирует тестовую базу данных
func initTestDB(t *testing.T) {
	config.InitConfig("../../.env")
	db.DB, _ = sql.Open("sqlite3", "file:memdb1?mode=memory&cache=shared")

	db.ApplySchema(filepath.Join("../../internal/db", "schema.sql"))
}

// cleanupTestDB очищает тестовую базу данных
func cleanupTestDB() {
	db.CleanupDB()
	db.CloseDB()
}

// TestCreateAST проверяет создание AST из выражения
func TestCreateAST(t *testing.T) {
	initTestDB(t)
	tests := []struct {
		name        string
		expression  string
		wantErr     bool
		checkResult bool // Указывает, нужно ли проверять возвращенный AST
	}{
		{
			name:        "Valid simple expression",
			expression:  "2+2",
			wantErr:     false,
			checkResult: true,
		},
		{
			name:        "Valid complex expression",
			expression:  "2*(3+4)",
			wantErr:     false,
			checkResult: true,
		},
		{
			name:        "Invalid expression",
			expression:  "2++2",
			wantErr:     true,
			checkResult: false,
		},
		{
			name:        "Empty expression",
			expression:  "",
			wantErr:     true,
			checkResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node, err := orchestrator.CreateAST(tt.expression)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateAST() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && tt.checkResult {
				if node == nil {
					t.Errorf("CreateAST() returned nil node for valid expression")
					return
				}

				// Проверяем, что возвращаемый узел действительно является AST-узлом
				_, ok := node.(ast.Expr)
				if !ok {
					t.Errorf("CreateAST() returned node is not ast.Expr")
				}
			}
		})
	}
}

// TestProcessExpression проверяет обработку выражения
func TestProcessExpression(t *testing.T) {
	// Инициализируем БД
	initTestDB(t)

	// Создаем тестового пользователя
	user, err := db.CreateUser("testuser_proc_expr", "testpass")
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	tests := []struct {
		name       string
		expression string
		userID     int64
		wantErr    bool
	}{
		{
			name:       "Valid simple expression",
			expression: "2+2",
			userID:     user.ID,
			wantErr:    false,
		},
		{
			name:       "Valid complex expression",
			expression: "2*(3+4)",
			userID:     user.ID,
			wantErr:    false,
		},
		{
			name:       "Invalid expression",
			expression: "2++2",
			userID:     user.ID,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exprID, err := orchestrator.ProcessExpression(tt.expression, tt.userID)
			if (err != nil) != tt.wantErr {
				t.Errorf("ProcessExpression() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if exprID == nil {
					t.Errorf("ProcessExpression() returned nil ID for valid expression")
					return
				}

				// Проверяем, что выражение было сохранено в БД
				expr, err := db.GetExpressionByID(*exprID)
				if err != nil {
					t.Errorf("Failed to get expression from DB: %v", err)
					return
				}

				if expr.Expression != tt.expression {
					t.Errorf("Saved expression = %v, want %v", expr.Expression, tt.expression)
				}

				if expr.UserID != tt.userID {
					t.Errorf("Expression user ID = %v, want %v", expr.UserID, tt.userID)
				}

				// Для успешного случая проверяем, что были созданы операции
				ops, err := db.GetOperationsByExpressionID(*exprID)
				if err != nil {
					t.Errorf("Failed to get operations: %v", err)
					return
				}

				if len(ops) == 0 {
					t.Errorf("No operations created for expression")
				}
			}
		})
	}
}

// TaskResult представляет результат выполнения задачи
type TaskResult struct {
	ID     int64   // ID операции
	Result float64 // Результат операции
	Error  string  // Ошибка, если есть
}

// TestProcessExpressionResult проверяет обработку результата выражения
func TestProcessExpressionResult(t *testing.T) {
	// Инициализируем БД
	initTestDB(t)

	// Создаем тестового пользователя
	user, err := db.CreateUser("testuser_proc_result", "testpass")
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	// Создаем тестовое выражение
	expr, err := db.CreateExpression(user.ID, "3+4")
	if err != nil {
		t.Fatalf("Failed to create test expression: %v", err)
	}

	// Создаем корневую операцию
	rootOp, err := db.CreateOperation(
		expr.ID,
		nil,
		"+",
		nil, nil,
		true,
		nil,
		db.StatusPending,
	)
	if err != nil {
		t.Fatalf("Failed to create root operation: %v", err)
	}

	// Создаем левый операнд
	leftPos := "left"
	leftVal := 3.0
	leftOp, err := db.CreateOperation(
		expr.ID,
		&rootOp.ID,
		"",
		&leftVal,
		nil,
		false,
		&leftPos,
		db.StatusCompleted,
	)
	if err != nil {
		t.Fatalf("Failed to create left operation: %v", err)
	}

	// Создаем правый операнд
	rightPos := "right"
	rightVal := 4.0
	rightOp, err := db.CreateOperation(
		expr.ID,
		&rootOp.ID,
		"",
		&rightVal,
		nil,
		false,
		&rightPos,
		db.StatusCompleted,
	)
	if err != nil {
		t.Fatalf("Failed to create right operation: %v", err)
	}

	// Обновляем аргументы корневой операции
	err = orchestrator.UpdateParentOperation(leftOp, leftVal)
	if err != nil {
		t.Fatalf("Failed to update parent with left operation: %v", err)
	}

	err = orchestrator.UpdateParentOperation(rightOp, rightVal)
	if err != nil {
		t.Fatalf("Failed to update parent with right operation: %v", err)
	}

	// Получаем обновленную корневую операцию
	rootOp, err = db.GetOperationByID(rootOp.ID)
	if err != nil {
		t.Fatalf("Failed to get updated root operation: %v", err)
	}

	// Проверяем, что статус корневой операции стал "ready"
	if rootOp.Status != db.StatusReady {
		t.Fatalf("Root operation status = %v, want %v", rootOp.Status, db.StatusReady)
	}

	// Создаем результат для корневой операции
	result := orchestrator.TaskResult{
		ID:     rootOp.ID,
		Result: 7.0, // 3 + 4 = 7
		Error:  "nil",
	}

	// Тестируем успешный случай
	t.Run("Successful result processing", func(t *testing.T) {
		err := orchestrator.ProcessExpressionResult(result)
		if err != nil {
			t.Errorf("ProcessExpressionResult() error = %v", err)
			return
		}

		// Проверяем, что статус корневой операции обновился на "completed"
		updatedRootOp, err := db.GetOperationByID(rootOp.ID)
		if err != nil {
			t.Errorf("Failed to get updated root operation: %v", err)
			return
		}

		if updatedRootOp.Status != db.StatusCompleted {
			t.Errorf("Root operation status = %v, want %v", updatedRootOp.Status, db.StatusCompleted)
		}

		if updatedRootOp.Result == nil {
			t.Errorf("Root operation result is nil")
			return
		}

		if *updatedRootOp.Result != 7.0 {
			t.Errorf("Root operation result = %v, want %v", *updatedRootOp.Result, 7.0)
		}

		// Проверяем, что результат выражения тоже обновился
		updatedExpr, err := db.GetExpressionByID(expr.ID)
		if err != nil {
			t.Errorf("Failed to get updated expression: %v", err)
			return
		}

		if updatedExpr.Status != db.StatusCompleted {
			t.Errorf("Expression status = %v, want %v", updatedExpr.Status, db.StatusCompleted)
		}

		if updatedExpr.Result == nil {
			t.Errorf("Expression result is nil")
			return
		}

		if *updatedExpr.Result != 7.0 {
			t.Errorf("Expression result = %v, want %v", *updatedExpr.Result, 7.0)
		}
	})

	// Создаем еще одно выражение для тестирования ошибки
	exprWithError, err := db.CreateExpression(user.ID, "5/0")
	if err != nil {
		t.Fatalf("Failed to create test expression with error: %v", err)
	}

	// Создаем корневую операцию для выражения с ошибкой
	rootOpWithError, err := db.CreateOperation(
		exprWithError.ID,
		nil,
		"/",
		nil, nil,
		true,
		nil,
		db.StatusPending,
	)
	if err != nil {
		t.Fatalf("Failed to create root operation with error: %v", err)
	}

	// Создаем результат с ошибкой
	errorResult := orchestrator.TaskResult{
		ID:     rootOpWithError.ID,
		Result: 0,
		Error:  "division by zero",
	}

	// Тестируем обработку ошибки
	t.Run("Error result processing", func(t *testing.T) {
		err := orchestrator.ProcessExpressionResult(errorResult)
		if err != nil {
			t.Errorf("ProcessExpressionResult() error = %v", err)
			return
		}

		// Проверяем, что статус операции обновился на "canceled", так как HandleOperationErrorWithCancellation
		// отменяет все операции, связанные с выражением
		updatedRootOp, err := db.GetOperationByID(rootOpWithError.ID)
		if err != nil {
			t.Errorf("Failed to get updated root operation: %v", err)
			return
		}

		if updatedRootOp.Status != db.StatusCanceled {
			t.Errorf("Root operation status = %v, want %v", updatedRootOp.Status, db.StatusCanceled)
		}

		// HandleOperationErrorWithCancellation устанавливает сообщение об ошибке для выражения,
		// но не обязательно для самой операции, поэтому это условие не проверяем

		// Проверяем, что статус выражения обновился на "error"
		updatedExpr, err := db.GetExpressionByID(exprWithError.ID)
		if err != nil {
			t.Errorf("Failed to get updated expression: %v", err)
			return
		}

		// Проверяем, что статус выражения изменился на "error"
		if updatedExpr.Status != db.StatusError {
			t.Errorf("Expression status = %v, want %v", updatedExpr.Status, db.StatusError)
		}

		// Проверяем, что сообщение об ошибке было установлено
		if updatedExpr.ErrorMessage == nil {
			t.Errorf("Expression error message is nil")
			return
		}

		// Убедимся, что сообщение об ошибке содержит нашу ошибку
		if *updatedExpr.ErrorMessage != "division by zero" {
			t.Errorf("Expression error message = %v, want %v", *updatedExpr.ErrorMessage, "division by zero")
		}
	})
}
