package orchestrator

import (
	"errors"
	"fmt"
	"net/http"
	"parallel-calculator/internal/auth"
	"parallel-calculator/internal/db"
)

// GetUserIDFromRequest извлекает ID пользователя из запроса с JWT-токеном
func GetUserIDFromRequest(r *http.Request) (int64, error) {
	// Получаем claims из контекста (добавленные middleware auth)
	claims, ok := auth.GetUserFromContext(r.Context())
	if !ok {
		return 0, errors.New("не удалось получить информацию о пользователе")
	}

	return claims.UserID, nil
}

// CreateExpressionInDB создает новое выражение в базе данных
func CreateExpressionInDB(userID int64, expressionStr string) (*db.Expression, error) {
	// Создаем запись выражения в БД
	expr, err := db.CreateExpression(userID, expressionStr)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания выражения в БД: %w", err)
	}

	return expr, nil
}

// CreateOperationInDB создает операцию в базе данных
func CreateOperationInDB(
	expressionID int64,
	parentOpID *int64,
	operator string,
	leftValue, rightValue *float64,
	isRoot bool,
	childPosition *string) (*db.Operation, error) {

	// Создаем операцию в БД
	op, err := db.CreateOperation(
		expressionID,
		parentOpID,
		operator,
		leftValue, rightValue,
		isRoot,
		childPosition)

	if err != nil {
		return nil, fmt.Errorf("ошибка создания операции в БД: %w", err)
	}

	return op, nil
}

// UpdateOperationStatusInDB обновляет статус операции в БД
func UpdateOperationStatusInDB(operationID int64, status string) error {
	return db.UpdateOperationStatus(operationID, status)
}

// SetOperationResultInDB устанавливает результат операции в БД
func SetOperationResultInDB(operationID int64, result float64) error {
	return db.SetOperationResult(operationID, result)
}

// HandleOperationErrorInDB обрабатывает ошибки операций
func HandleOperationErrorInDB(operationID int64, errorMsg string) error {
	return db.HandleOperationError(operationID, errorMsg)
}

// GetExpressionsByUserID получает все выражения пользователя
func GetExpressionsByUserID(userID int64) ([]*db.Expression, error) {
	// Здесь нужно реализовать получение выражений из БД по ID пользователя
	// Пока это заглушка
	return nil, errors.New("не реализовано")
}

// GetReadyOperationFromDB получает одну операцию, готовую к выполнению
func GetReadyOperationFromDB() (*db.Operation, error) {
	return db.GetReadyOperation()
}
