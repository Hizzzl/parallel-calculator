package orchestrator_test

import (
	"context"
	"net/http/httptest"
	"parallel-calculator/internal/auth"
	"parallel-calculator/internal/db"
	"parallel-calculator/internal/orchestrator"
	"testing"
)

// TestCreateOperationInDB проверяет создание операции в базе данных
func TestCreateOperationInDB(t *testing.T) {
	// Инициализируем БД
	initTestDB(t)
	defer cleanupTestDB()

	// Создаем тестового пользователя
	user, err := db.CreateUser("testuser_op", "testpass")
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	// Создаем тестовое выражение
	expressionStr := "3*4"
	expr, err := db.CreateExpression(user.ID, expressionStr)
	if err != nil {
		t.Fatalf("Failed to create test expression: %v", err)
	}

	// Тестируем создание корневой операции
	op, err := orchestrator.CreateOperationInDB(
		expr.ID,
		nil,
		"*",
		nil, nil,
		true,
		nil,
		db.StatusPending,
	)

	if err != nil {
		t.Errorf("CreateOperationInDB() error = %v", err)
		return
	}

	// Проверяем, что операция создана корректно
	if op.ExpressionID != expr.ID {
		t.Errorf("Operation expression ID = %v, want %v", op.ExpressionID, expr.ID)
	}

	if op.Operator != "*" {
		t.Errorf("Operation operator = %v, want %v", op.Operator, "*")
	}

	if !op.IsRootExpression {
		t.Errorf("Operation root status = %v, want true", op.IsRootExpression)
	}

	if op.Status != db.StatusPending {
		t.Errorf("Operation status = %v, want %v", op.Status, db.StatusPending)
	}

	// Тестируем создание дочерней операции
	childPosition := "left"
	leftValue := 3.0
	child, err := orchestrator.CreateOperationInDB(
		expr.ID,
		&op.ID,
		"",
		&leftValue, nil,
		false,
		&childPosition,
		db.StatusCompleted,
	)

	if err != nil {
		t.Errorf("CreateOperationInDB() error = %v for child operation", err)
		return
	}

	// Проверяем, что дочерняя операция создана корректно
	if child.ParentOpID == nil || *child.ParentOpID != op.ID {
		parentID := int64(0)
		if child.ParentOpID != nil {
			parentID = *child.ParentOpID
		}
		t.Errorf("Child operation parent ID = %v, want %v", parentID, op.ID)
	}

	if child.ChildPosition == nil || *child.ChildPosition != childPosition {
		pos := ""
		if child.ChildPosition != nil {
			pos = *child.ChildPosition
		}
		t.Errorf("Child operation position = %v, want %v", pos, childPosition)
	}

	if child.LeftValue == nil || *child.LeftValue != leftValue {
		val := 0.0
		if child.LeftValue != nil {
			val = *child.LeftValue
		}
		t.Errorf("Child operation left value = %v, want %v", val, leftValue)
	}

	if child.IsRootExpression {
		t.Errorf("Child operation root status = %v, want false", child.IsRootExpression)
	}

	if child.Status != db.StatusCompleted {
		t.Errorf("Child operation status = %v, want %v", child.Status, db.StatusCompleted)
	}
}

// TestSetOperationResultInDB проверяет установку результата операции
func TestSetOperationResultInDB(t *testing.T) {
	// Инициализируем БД
	initTestDB(t)
	defer cleanupTestDB()

	// Создаем тестового пользователя
	user, err := db.CreateUser("testuser_op_result", "testpass")
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	// Создаем тестовое выражение
	expressionStr := "5*6"
	expr, err := db.CreateExpression(user.ID, expressionStr)
	if err != nil {
		t.Fatalf("Failed to create test expression: %v", err)
	}

	// Создаем корневую операцию
	root, err := db.CreateOperation(
		expr.ID,
		nil,
		"*",
		nil, nil,
		true,
		nil,
		db.StatusPending,
	)
	if err != nil {
		t.Fatalf("Failed to create root operation: %v", err)
	}

	// Устанавливаем результат операции
	result := 30.0 // Результат 5*6
	err = orchestrator.SetOperationResultInDB(root.ID, result)
	if err != nil {
		t.Errorf("SetOperationResultInDB() error = %v", err)
		return
	}

	// Проверяем, что результат операции установлен корректно
	updatedOp, err := db.GetOperationByID(root.ID)
	if err != nil {
		t.Errorf("Failed to get updated operation: %v", err)
		return
	}

	if updatedOp.Result == nil {
		t.Errorf("Operation result is nil, expected %v", result)
		return
	}

	if *updatedOp.Result != result {
		t.Errorf("Operation result = %v, want %v", *updatedOp.Result, result)
	}

	// Поскольку это корневая операция, проверяем, что результат также установлен для выражения
	updatedExpr, err := db.GetExpressionByID(expr.ID)
	if err != nil {
		t.Errorf("Failed to get updated expression: %v", err)
		return
	}

	if updatedExpr.Result == nil {
		t.Errorf("Expression result is nil, expected %v", result)
		return
	}

	if *updatedExpr.Result != result {
		t.Errorf("Expression result = %v, want %v", *updatedExpr.Result, result)
	}

	// Дополнительный тест для некорневой операции
	// Создаем дочернюю операцию
	childPos := "left"
	child, err := db.CreateOperation(
		expr.ID,
		&root.ID,
		"",
		nil, nil,
		false,
		&childPos,
		db.StatusPending,
	)
	if err != nil {
		t.Fatalf("Failed to create child operation: %v", err)
	}

	// Устанавливаем результат для дочерней операции
	childResult := 5.0
	err = orchestrator.SetOperationResultInDB(child.ID, childResult)
	if err != nil {
		t.Errorf("SetOperationResultInDB() error = %v for child operation", err)
		return
	}

	// Проверяем, что результат дочерней операции установлен, но результат выражения не изменился
	updatedChild, err := db.GetOperationByID(child.ID)
	if err != nil {
		t.Errorf("Failed to get updated child operation: %v", err)
		return
	}

	if updatedChild.Result == nil {
		t.Errorf("Child operation result is nil, expected %v", childResult)
		return
	}

	if *updatedChild.Result != childResult {
		t.Errorf("Child operation result = %v, want %v", *updatedChild.Result, childResult)
	}

	// Проверяем, что результат выражения не изменился после установки результата дочерней операции
	updatedExpr, err = db.GetExpressionByID(expr.ID)
	if err != nil {
		t.Errorf("Failed to get expression after child operation update: %v", err)
		return
	}

	if *updatedExpr.Result != result {
		t.Errorf("Expression result changed after child operation update: got %v, want %v", *updatedExpr.Result, result)
	}
}

// TestHandleOperationErrorInDB проверяет обработку ошибки операции в базе данных
func TestHandleOperationErrorInDB(t *testing.T) {
	// Инициализируем БД
	initTestDB(t)
	defer cleanupTestDB()

	// Создаем тестового пользователя
	user, err := db.CreateUser("testuser_error", "testpass")
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	// Создаем тестовое выражение
	expressionStr := "2/0" // Деление на ноль, которое вызовет ошибку
	expr, err := db.CreateExpression(user.ID, expressionStr)
	if err != nil {
		t.Fatalf("Failed to create test expression: %v", err)
	}

	// Создаем операцию деления
	op, err := db.CreateOperation(
		expr.ID,
		nil,
		"/",
		nil, nil,
		true,
		nil,
		db.StatusPending,
	)
	if err != nil {
		t.Fatalf("Failed to create operation: %v", err)
	}

	// Обрабатываем ошибку операции
	errorMsg := "Деление на ноль"
	err = orchestrator.HandleOperationErrorInDB(op.ID, errorMsg)
	if err != nil {
		t.Errorf("HandleOperationErrorInDB() error = %v", err)
		return
	}

	// Проверяем, что ошибка установлена для операции
	updatedOp, err := db.GetOperationByID(op.ID)
	if err != nil {
		t.Errorf("Failed to get updated operation: %v", err)
		return
	}

	if updatedOp.ErrorMessage == nil {
		t.Errorf("Operation error message is nil, expected: %v", errorMsg)
		return
	}

	if *updatedOp.ErrorMessage != errorMsg {
		t.Errorf("Operation error message = %v, want %v", *updatedOp.ErrorMessage, errorMsg)
	}

	// Проверяем, что ошибка также установлена для выражения
	updatedExpr, err := db.GetExpressionByID(expr.ID)
	if err != nil {
		t.Errorf("Failed to get updated expression: %v", err)
		return
	}

	if updatedExpr.ErrorMessage == nil {
		t.Errorf("Expression error message is nil, expected: %v", errorMsg)
		return
	}

	if *updatedExpr.ErrorMessage != errorMsg {
		t.Errorf("Expression error message = %v, want %v", *updatedExpr.ErrorMessage, errorMsg)
	}
}

// TestHandleOperationErrorWithCancellation проверяет обработку ошибки операции с отменой всех связанных операций
func TestHandleOperationErrorWithCancellation(t *testing.T) {
	// Инициализируем БД
	initTestDB(t)
	defer cleanupTestDB()

	// Создаем тестового пользователя
	user, err := db.CreateUser("testuser_cancel", "testpass")
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	// Создаем тестовое выражение
	expressionStr := "(2+3)*(4/0)" // Выражение с ошибкой деления на ноль
	expr, err := db.CreateExpression(user.ID, expressionStr)
	if err != nil {
		t.Fatalf("Failed to create test expression: %v", err)
	}

	// Создаем несколько операций для этого выражения
	// Корневая операция - умножение
	root, err := db.CreateOperation(
		expr.ID,
		nil,
		"*",
		nil, nil,
		true,
		nil,
		db.StatusPending,
	)
	if err != nil {
		t.Fatalf("Failed to create root operation: %v", err)
	}

	// Левая операция - сложение
	leftPos := "left"
	left, err := db.CreateOperation(
		expr.ID,
		&root.ID,
		"+",
		nil, nil,
		false,
		&leftPos,
		db.StatusPending,
	)
	if err != nil {
		t.Fatalf("Failed to create left operation: %v", err)
	}

	// Правая операция - деление (с ошибкой)
	rightPos := "right"
	right, err := db.CreateOperation(
		expr.ID,
		&root.ID,
		"/",
		nil, nil,
		false,
		&rightPos,
		db.StatusPending,
	)
	if err != nil {
		t.Fatalf("Failed to create right operation: %v", err)
	}

	// Обрабатываем ошибку операции с отменой всех связанных операций
	errorMsg := "Деление на ноль в правом операнде"
	err = orchestrator.HandleOperationErrorWithCancellation(right.ID, errorMsg)
	if err != nil {
		t.Errorf("HandleOperationErrorWithCancellation() error = %v", err)
		return
	}

	// Проверяем, что ошибка установлена для выражения
	updatedExpr, err := db.GetExpressionByID(expr.ID)
	if err != nil {
		t.Errorf("Failed to get updated expression: %v", err)
		return
	}

	if updatedExpr.ErrorMessage == nil {
		t.Errorf("Expression error message is nil, expected: %v", errorMsg)
		return
	}

	if *updatedExpr.ErrorMessage != errorMsg {
		t.Errorf("Expression error message = %v, want %v", *updatedExpr.ErrorMessage, errorMsg)
	}

	// Проверяем, что все операции были отменены
	updatedRoot, err := db.GetOperationByID(root.ID)
	if err != nil {
		t.Errorf("Failed to get updated root operation: %v", err)
		return
	}

	updatedLeft, err := db.GetOperationByID(left.ID)
	if err != nil {
		t.Errorf("Failed to get updated left operation: %v", err)
		return
	}

	updatedRight, err := db.GetOperationByID(right.ID)
	if err != nil {
		t.Errorf("Failed to get updated right operation: %v", err)
		return
	}

	// Проверяем статусы всех операций
	if updatedRoot.Status != db.StatusCanceled {
		t.Errorf("Root operation status = %v, want %v", updatedRoot.Status, db.StatusCanceled)
	}

	if updatedLeft.Status != db.StatusCanceled {
		t.Errorf("Left operation status = %v, want %v", updatedLeft.Status, db.StatusCanceled)
	}

	if updatedRight.Status != db.StatusCanceled {
		t.Errorf("Right operation status = %v, want %v", updatedRight.Status, db.StatusCanceled)
	}
}

// TestFinalizeExpression проверяет установку результата выражения и обновление его статуса
func TestFinalizeExpression(t *testing.T) {
	// Инициализируем БД
	initTestDB(t)
	defer cleanupTestDB()

	// Создаем тестового пользователя
	user, err := db.CreateUser("testuser_finalize", "testpass")
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	// Создаем тестовое выражение
	expressionStr := "5+7"
	expr, err := db.CreateExpression(user.ID, expressionStr)
	if err != nil {
		t.Fatalf("Failed to create test expression: %v", err)
	}

	// Тестируем завершение выражения с результатом
	result := 12.0 // Результат 5+7
	err = orchestrator.FinalizeExpression(expr.ID, result)
	if err != nil {
		t.Errorf("FinalizeExpression() error = %v", err)
		return
	}

	// Получаем обновленное выражение
	updatedExpr, err := db.GetExpressionByID(expr.ID)
	if err != nil {
		t.Errorf("Failed to get updated expression: %v", err)
		return
	}

	// Проверяем, что результат и статус выражения обновлены корректно
	if updatedExpr.Result == nil {
		t.Errorf("Expression result is nil, expected %v", result)
		return
	}

	if *updatedExpr.Result != result {
		t.Errorf("Expression result = %v, want %v", *updatedExpr.Result, result)
	}

	if updatedExpr.Status != db.StatusCompleted {
		t.Errorf("Expression status = %v, want %v", updatedExpr.Status, db.StatusCompleted)
	}
}

// TestCreateExpressionInDB проверяет создание выражения в базе данных
func TestCreateExpressionInDB(t *testing.T) {
	// Инициализируем БД
	initTestDB(t)
	defer cleanupTestDB()

	// Создаем тестового пользователя
	user, err := db.CreateUser("testuser_integration", "testpass")
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	// Тестируем создание выражения
	expressionStr := "2+2"
	expr, err := orchestrator.CreateExpressionInDB(user.ID, expressionStr)
	if err != nil {
		t.Errorf("CreateExpressionInDB() error = %v", err)
		return
	}

	// Проверяем, что выражение создано с правильными параметрами
	if expr.UserID != user.ID {
		t.Errorf("Expression user ID = %v, want %v", expr.UserID, user.ID)
	}

	if expr.Expression != expressionStr {
		t.Errorf("Expression value = %v, want %v", expr.Expression, expressionStr)
	}

	if expr.Status != db.StatusPending {
		t.Errorf("Expression status = %v, want %v", expr.Status, db.StatusPending)
	}
}

// TestGetExpressionByID проверяет получение выражения по ID
func TestGetExpressionByID(t *testing.T) {
	// Инициализируем БД
	initTestDB(t)
	defer cleanupTestDB()

	// Создаем тестового пользователя
	user, err := db.CreateUser("testuser_get_expr", "testpass")
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	// Создаем тестовое выражение
	expressionStr := "3*4"
	createdExpr, err := db.CreateExpression(user.ID, expressionStr)
	if err != nil {
		t.Fatalf("Failed to create test expression: %v", err)
	}

	// Тестируем получение выражения по ID
	expr, err := orchestrator.GetExpressionByID(createdExpr.ID)
	if err != nil {
		t.Errorf("GetExpressionByID() error = %v", err)
		return
	}

	// Проверяем, что получено правильное выражение
	if expr.ID != createdExpr.ID {
		t.Errorf("Expression ID = %v, want %v", expr.ID, createdExpr.ID)
	}

	if expr.UserID != user.ID {
		t.Errorf("Expression user ID = %v, want %v", expr.UserID, user.ID)
	}

	if expr.Expression != expressionStr {
		t.Errorf("Expression value = %v, want %v", expr.Expression, expressionStr)
	}

	// Тестируем получение несуществующего выражения
	_, err = orchestrator.GetExpressionByID(999999)
	if err != orchestrator.ErrExpressionNotFound {
		t.Errorf("GetExpressionByID() for non-existent expression error = %v, want %v", err, orchestrator.ErrExpressionNotFound)
	}
}

// TestUpdateParentOperation проверяет обновление аргументов родительской операции
func TestUpdateParentOperation(t *testing.T) {
	// Инициализируем БД
	initTestDB(t)
	defer cleanupTestDB()

	// Создаем тестового пользователя
	user, err := db.CreateUser("testuser_parent_op", "testpass")
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	// Создаем тестовое выражение
	expressionStr := "2+3"
	expr, err := db.CreateExpression(user.ID, expressionStr)
	if err != nil {
		t.Fatalf("Failed to create test expression: %v", err)
	}

	// Создаем родительскую операцию (сложение)
	parentOpID := int64(0)
	parentOp, err := db.CreateOperation(
		expr.ID,
		nil,
		"+",
		nil, nil,
		true,
		nil,
		db.StatusPending,
	)
	if err != nil {
		t.Fatalf("Failed to create parent operation: %v", err)
	}
	parentOpID = parentOp.ID

	// Создаем левую дочернюю операцию (число 2)
	leftChildPos := "left"
	leftChild, err := db.CreateOperation(
		expr.ID,
		&parentOpID,
		"",
		floatPtr(2.0),
		nil,
		false,
		&leftChildPos,
		db.StatusCompleted,
	)
	if err != nil {
		t.Fatalf("Failed to create left child operation: %v", err)
	}

	// Тестируем обновление родительской операции после "вычисления" левого дочернего узла
	err = orchestrator.UpdateParentOperation(leftChild, 2.0)
	if err != nil {
		t.Errorf("UpdateParentOperation() for left child error = %v", err)
		return
	}

	// Получаем обновленную родительскую операцию
	updatedParent, err := db.GetOperationByID(parentOpID)
	if err != nil {
		t.Errorf("GetOperationByID() for parent error = %v", err)
		return
	}

	// Проверяем, что левый аргумент установлен, но правый еще нет
	if updatedParent.LeftValue == nil {
		t.Errorf("Parent operation left value is nil, expected 2.0")
		return
	}

	if *updatedParent.LeftValue != 2.0 {
		t.Errorf("Parent operation left value = %v, want %v", *updatedParent.LeftValue, 2.0)
	}

	if updatedParent.RightValue != nil {
		t.Errorf("Parent operation right value should be nil at this point")
	}

	// Проверяем, что статус родительской операции все еще pending, так как правый аргумент не установлен
	if updatedParent.Status != db.StatusPending {
		t.Errorf("Parent operation status = %v, want %v", updatedParent.Status, db.StatusPending)
	}

	// Создаем правую дочернюю операцию (число 3)
	rightChildPos := "right"
	rightChild, err := db.CreateOperation(
		expr.ID,
		&parentOpID,
		"",
		floatPtr(3.0),
		nil,
		false,
		&rightChildPos,
		db.StatusCompleted,
	)
	if err != nil {
		t.Fatalf("Failed to create right child operation: %v", err)
	}

	// Тестируем обновление родительской операции после "вычисления" правого дочернего узла
	err = orchestrator.UpdateParentOperation(rightChild, 3.0)
	if err != nil {
		t.Errorf("UpdateParentOperation() for right child error = %v", err)
		return
	}

	// Получаем обновленную родительскую операцию
	updatedParent, err = db.GetOperationByID(parentOpID)
	if err != nil {
		t.Errorf("GetOperationByID() for parent error = %v", err)
		return
	}

	// Проверяем, что оба аргумента установлены
	if updatedParent.LeftValue == nil || updatedParent.RightValue == nil {
		t.Errorf("Parent operation arguments are not both set")
		return
	}

	if *updatedParent.LeftValue != 2.0 {
		t.Errorf("Parent operation left value = %v, want %v", *updatedParent.LeftValue, 2.0)
	}

	if *updatedParent.RightValue != 3.0 {
		t.Errorf("Parent operation right value = %v, want %v", *updatedParent.RightValue, 3.0)
	}

	// Проверяем, что статус родительской операции изменился на ready, так как оба аргумента установлены
	if updatedParent.Status != db.StatusReady {
		t.Errorf("Parent operation status = %v, want %v", updatedParent.Status, db.StatusReady)
	}
}

// Helper функция для создания указателя на float64
func floatPtr(v float64) *float64 {
	return &v
}

// TestGetUserIDFromRequest проверяет извлечение ID пользователя из запроса
func TestGetUserIDFromRequest(t *testing.T) {
	// Инициализируем БД
	initTestDB(t)
	defer cleanupTestDB()

	// Создаем тестовые данные
	userID := int64(123)

	tests := []struct {
		name        string
		userInCtx   bool
		userID      int64
		expectedErr bool
	}{
		{
			name:        "Valid user in context",
			userInCtx:   true,
			userID:      userID,
			expectedErr: false,
		},
		{
			name:        "No user in context",
			userInCtx:   false,
			userID:      0,
			expectedErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Создаем mock HTTP запроса
			req := httptest.NewRequest("GET", "/", nil)

			// Если нужно, добавляем пользователя в контекст
			if tt.userInCtx {
				// Создаем claims с указанным userID
				claims := &auth.Claims{UserID: tt.userID}
				// Добавляем claims в контекст
				ctx := context.WithValue(req.Context(), auth.UserContextKey, claims)
				req = req.WithContext(ctx)
			}

			// Вызываем тестируемую функцию
			gotUserID, err := orchestrator.GetUserIDFromRequest(req)

			// Проверяем ошибку
			if (err != nil) != tt.expectedErr {
				t.Errorf("GetUserIDFromRequest() error = %v, expectedErr %v", err, tt.expectedErr)
				return
			}

			// Если не ожидается ошибка, проверяем корректность ID пользователя
			if !tt.expectedErr && gotUserID != tt.userID {
				t.Errorf("GetUserIDFromRequest() = %v, want %v", gotUserID, tt.userID)
			}
		})
	}
}
