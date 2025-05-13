package orchestrator_test

import (
	"go/ast"
	"go/parser"
	"parallel-calculator/internal/db"
	"parallel-calculator/internal/orchestrator"
	"testing"
)

// prepareASTNode подготавливает AST узел для тестирования
func prepareASTNode(t *testing.T, expr string) ast.Node {
	// Создаем файловый набор для парсинга
	node, err := parser.ParseExpr(expr)
	if err != nil {
		t.Fatalf("Failed to parse expression '%s': %v", expr, err)
	}
	return node
}

// TestParseAST_SimpleNumber проверяет обработку простого числа
func TestParseAST_SimpleNumber(t *testing.T) {
	// Инициализируем БД
	initTestDB(t)

	// Создаем тестового пользователя
	user, err := db.CreateUser("testuser_simple", "testpass")
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	// Создаем тестовое выражение
	expr, err := db.CreateExpression(user.ID, "42")
	if err != nil {
		t.Fatalf("Failed to create test expression: %v", err)
	}

	// Создаем AST узел для простого числа
	node := prepareASTNode(t, "42")

	// Парсим AST и создаем операции в базе данных
	err = orchestrator.ParseAST(expr.ID, node)
	if err != nil {
		t.Errorf("ParseAST() error = %v", err)
		return
	}

	// Получаем выражение из базы данных и проверяем его статус и результат
	updatedExpr, err := db.GetExpressionByID(expr.ID)
	if err != nil {
		t.Errorf("GetExpressionByID() error = %v", err)
		return
	}

	// Проверяем, что статус и результат установлены правильно
	if updatedExpr.Status != orchestrator.StatusCompleted {
		t.Errorf("Expression status = %v, want %v", updatedExpr.Status, orchestrator.StatusCompleted)
	}

	if updatedExpr.Result == nil {
		t.Errorf("Expression result is nil, expected 42.0")
		return
	}

	if *updatedExpr.Result != 42.0 {
		t.Errorf("Expression result = %v, want %v", *updatedExpr.Result, 42.0)
	}
}

// TestParseAST_NumberInParentheses проверяет обработку числа в скобках
func TestParseAST_NumberInParentheses(t *testing.T) {
	// Инициализируем БД
	initTestDB(t)
	// defer cleanupTestDB()

	// Создаем тестового пользователя
	user, err := db.CreateUser("testuser_parens", "testpass")
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	// Создаем тестовое выражение
	expr, err := db.CreateExpression(user.ID, "(42)")
	if err != nil {
		t.Fatalf("Failed to create test expression: %v", err)
	}

	// Создаем AST узел для числа в скобках
	node := prepareASTNode(t, "(42)")

	// Парсим AST и создаем операции в базе данных
	err = orchestrator.ParseAST(expr.ID, node)
	if err != nil {
		t.Errorf("ParseAST() error = %v", err)
		return
	}

	// Получаем выражение из базы данных и проверяем его статус и результат
	updatedExpr, err := db.GetExpressionByID(expr.ID)
	if err != nil {
		t.Errorf("GetExpressionByID() error = %v", err)
		return
	}

	// Проверяем, что статус и результат установлены правильно
	if updatedExpr.Status != orchestrator.StatusCompleted {
		t.Errorf("Expression status = %v, want %v", updatedExpr.Status, orchestrator.StatusCompleted)
	}

	if updatedExpr.Result == nil {
		t.Errorf("Expression result is nil, expected 42.0")
		return
	}

	if *updatedExpr.Result != 42.0 {
		t.Errorf("Expression result = %v, want %v", *updatedExpr.Result, 42.0)
	}
}

// TestParseAST_ComplexExpression проверяет обработку сложного выражения со скобками
func TestParseAST_ComplexExpression(t *testing.T) {
	// Инициализируем БД
	initTestDB(t)
	// defer cleanupTestDB()

	// Создаем тестового пользователя
	user, err := db.CreateUser("testuser_complex", "testpass")
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	// Создаем тестовое выражение
	expr, err := db.CreateExpression(user.ID, "(2+3)*(4+5)")
	if err != nil {
		t.Fatalf("Failed to create test expression: %v", err)
	}

	// Создаем AST узел для сложного выражения со скобками
	node := prepareASTNode(t, "(2+3)*(4+5)")

	// Парсим AST и создаем операции в базе данных
	err = orchestrator.ParseAST(expr.ID, node)
	if err != nil {
		t.Errorf("ParseAST() error = %v", err)
		return
	}

	// Получаем выражение из базы данных
	updatedExpr, err := db.GetExpressionByID(expr.ID)
	if err != nil {
		t.Errorf("GetExpressionByID() error = %v", err)
		return
	}

	// Проверяем, что статус задан как `pending` (т.к. выражение еще не вычислено)
	if updatedExpr.Status != orchestrator.StatusPending {
		t.Errorf("Expression status = %v, want %v", updatedExpr.Status, orchestrator.StatusPending)
	}

	// Получаем операции, связанные с этим выражением
	ops, err := db.GetOperationsByExpressionID(expr.ID)
	if err != nil {
		t.Errorf("GetOperationsByExpressionID() error = %v", err)
		return
	}

	// Проверяем, что создано как минимум 3 операции (1 корневая и 2 дочерних)
	if len(ops) < 3 {
		t.Errorf("Expected at least 3 operations, got %d", len(ops))
		return
	}

	// Проверяем, что есть ровно одна корневая операция
	rootOps := 0
	for _, op := range ops {
		if op.IsRootExpression {
			rootOps++
		}
	}

	if rootOps != 1 {
		t.Errorf("Expected exactly 1 root operation, got %d", rootOps)
	}

	// Проверяем, что корневая операция - это умножение
	for _, op := range ops {
		if op.IsRootExpression {
			if op.Operator != "*" {
				t.Errorf("Root operation should be multiplication, got %s", op.Operator)
			}
		}
	}
}
