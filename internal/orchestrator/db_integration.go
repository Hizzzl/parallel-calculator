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
	childPosition *string,
	status string) (*db.Operation, error) {

	// Создаем операцию в БД
	op, err := db.CreateOperation(
		expressionID,
		parentOpID,
		operator,
		leftValue, rightValue,
		isRoot,
		childPosition,
		status)

	if err != nil {
		return nil, fmt.Errorf("ошибка создания операции в БД: %w", err)
	}

	return op, nil
}

// UpdateOperationStatusInDB обновляет статус операции в БД
func UpdateOperationStatusInDB(operationID int64, status string) error {
	return db.UpdateOperationStatus(operationID, status)
}

// SetOperationResultInDB устанавливает результат операции
func SetOperationResultInDB(operationID int64, result float64) error {
	// Вызываем базовую функцию для установки результата операции
	err := db.SetOperationResult(operationID, result)
	if err != nil {
		return err
	}

	// Проверяем, является ли это корневой операцией
	operationInfo, err := db.GetOperationInfo(operationID)
	if err != nil {
		return err
	}

	// Если это корневая операция, обновляем результат выражения
	if operationInfo.IsRootExpression && operationInfo.ExpressionID > 0 {
		return db.SetExpressionResult(operationInfo.ExpressionID, result)
	}

	return nil
}

// HandleOperationErrorInDB обрабатывает ошибку операции
func HandleOperationErrorInDB(operationID int64, errorMsg string) error {
	// Получаем операцию, чтобы узнать выражение
	op, err := db.GetOperationByID(operationID)
	if err != nil {
		return fmt.Errorf("ошибка при получении операции: %w", err)
	}

	// Устанавливаем ошибку для операции
	err = db.SetOperationError(operationID, errorMsg)
	if err != nil {
		return fmt.Errorf("ошибка при установке ошибки операции: %w", err)
	}

	// Устанавливаем ошибку для выражения
	err = db.SetExpressionError(op.ExpressionID, errorMsg)
	if err != nil {
		return fmt.Errorf("ошибка при установке ошибки выражения: %w", err)
	}

	return nil
}

// HandleOperationErrorWithCancellation обрабатывает ошибку операции и отменяет все связанные операции
func HandleOperationErrorWithCancellation(operationID int64, errorMsg string) error {
	// Сначала получаем операцию, чтобы узнать выражение
	op, err := db.GetOperationByID(operationID)
	if err != nil {
		return fmt.Errorf("ошибка при получении операции: %w", err)
	}

	// Устанавливаем ошибку для выражения
	err = db.SetExpressionError(op.ExpressionID, errorMsg)
	if err != nil {
		return fmt.Errorf("ошибка при установке ошибки выражения: %w", err)
	}

	// Отменяем все остальные операции выражения
	err = db.CancelOperationsByExpressionID(op.ExpressionID)
	if err != nil {
		return fmt.Errorf("ошибка при отмене операций: %w", err)
	}

	return nil
}

// GetExpressionsByUserID получает все выражения пользователя
func GetExpressionsByUserID(userID int64) ([]*db.Expression, error) {
	return db.GetUserExpressions(userID)
}

// GetExpressionByID получает выражение по ID
func GetExpressionByID(id int64) (*db.Expression, error) {
	return db.GetExpressionByID(id)
}

// UpdateParentOperation обновляет аргументы родительской операции
func UpdateParentOperation(op *db.Operation, result float64) error {
	if op.ParentOpID == nil || op.ChildPosition == nil {
		return fmt.Errorf("operation has no parent or position")
	}

	// Обновляем соответствующий аргумент в родительской операции
	switch *op.ChildPosition {
	case "left":
		// Обновляем левый аргумент родителя
		err := db.UpdateOperationLeftValue(*op.ParentOpID, result)
		if err != nil {
			return fmt.Errorf("ошибка при обновлении левого аргумента родителя: %w", err)
		}
	case "right":
		// Обновляем правый аргумент родителя
		err := db.UpdateOperationRightValue(*op.ParentOpID, result)
		if err != nil {
			return fmt.Errorf("ошибка при обновлении правого аргумента родителя: %w", err)
		}
	default:
		return fmt.Errorf("неизвестная позиция операнда: %s", *op.ChildPosition)
	}

	// Проверяем, есть ли у родителя оба значения
	parentOp, err := db.GetOperationByID(*op.ParentOpID)
	if err != nil {
		return fmt.Errorf("ошибка при получении родительской операции: %w", err)
	}

	if parentOp.LeftValue != nil && parentOp.RightValue != nil {
		// Если оба аргумента заполнены, устанавливаем статус "ready"
		err = db.UpdateOperationStatus(*op.ParentOpID, db.StatusReady)
		if err != nil {
			return fmt.Errorf("ошибка при обновлении статуса родительской операции: %w", err)
		}
	}

	return nil
}

// GetReadyOperationFromDB получает одну операцию, готовую к выполнению
func GetReadyOperationFromDB() (*db.Operation, error) {
	return db.GetReadyOperation()
}

// FinalizeExpression устанавливает результат выражения и обновляет его статус на "completed"
func FinalizeExpression(expressionID int64, result float64) error {
	// 1. Устанавливаем результат выражения
	err := db.SetExpressionResult(expressionID, result)
	if err != nil {
		return fmt.Errorf("ошибка при установке результата выражения: %w", err)
	}

	// 2. Обновляем статус выражения на "completed"
	err = db.UpdateExpressionStatus(expressionID, db.StatusCompleted)
	if err != nil {
		return fmt.Errorf("ошибка при обновлении статуса выражения: %w", err)
	}

	return nil
}
