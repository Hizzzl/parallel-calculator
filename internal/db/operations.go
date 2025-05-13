package db

import (
	"database/sql"
	"errors"
	"time"
)

var (
	ErrOperationNotFound = errors.New("operation not found")
)

// CreateOperation создает новую операцию в базе данных
func CreateOperation(expressionID int64, parentOpID *int64, operator string,
	leftValue, rightValue *float64, isRoot bool, childPosition *string, status string) (*Operation, error) {
	DbMutex.Lock()
	defer DbMutex.Unlock()

	query := `
		INSERT INTO operations 
		(expression_id, parent_operation_id, child_position, left_value, right_value,
		operator, status, is_root_expression) 
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	res, err := DB.Exec(
		query,
		expressionID, parentOpID, childPosition, leftValue, rightValue,
		operator, status, isRoot,
	)
	if err != nil {
		return nil, err
	}

	// Возвращаем созданную операцию
	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}

	return &Operation{
		ID:               id,
		ExpressionID:     expressionID,
		ParentOpID:       parentOpID,
		ChildPosition:    childPosition,
		LeftValue:        leftValue,
		RightValue:       rightValue,
		Operator:         operator,
		Status:           status,
		IsRootExpression: isRoot,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}, nil
}

// GetOperationByID получает операцию по ID
func GetOperationByID(id int64) (*Operation, error) {
	DbMutex.Lock()
	defer DbMutex.Unlock()
	var op Operation
	var createdAtStr, updatedAtStr string
	var parentOpID sql.NullInt64
	var leftValue, rightValue, result sql.NullFloat64
	var childPosition sql.NullString
	var isRoot bool // Изменили тип на bool, так как SQLite хранит это поле как boolean

	err := DB.QueryRow(
		`SELECT id, expression_id, parent_operation_id, child_position, left_value, right_value,
		operator, status, result, is_root_expression, created_at, updated_at
		FROM operations WHERE id = ?`,
		id,
	).Scan(
		&op.ID, &op.ExpressionID, &parentOpID, &childPosition, &leftValue, &rightValue,
		&op.Operator, &op.Status, &result, &isRoot, &createdAtStr, &updatedAtStr,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrOperationNotFound
		}
		return nil, err
	}

	// Обрабатываем нулевые поля
	if parentOpID.Valid {
		val := parentOpID.Int64
		op.ParentOpID = &val
	}

	if leftValue.Valid {
		val := leftValue.Float64
		op.LeftValue = &val
	}

	if rightValue.Valid {
		val := rightValue.Float64
		op.RightValue = &val
	}

	// Обрабатываем позицию относительно родителя
	if childPosition.Valid {
		val := childPosition.String
		op.ChildPosition = &val
	}

	if result.Valid {
		val := result.Float64
		op.Result = &val
	}

	op.IsRootExpression = isRoot

	// Парсим временные метки в формате RFC3339
	op.CreatedAt, err = time.Parse(time.RFC3339, createdAtStr)
	if err != nil {
		return nil, err
	}

	op.UpdatedAt, err = time.Parse(time.RFC3339, updatedAtStr)
	if err != nil {
		return nil, err
	}

	return &op, nil
}

// GetOperationsByExpressionID получает все операции для выражения
func GetOperationsByExpressionID(expressionID int64) ([]*Operation, error) {
	DbMutex.Lock()
	defer DbMutex.Unlock()
	rows, err := DB.Query(
		`SELECT id, expression_id, parent_operation_id, child_position, left_value, right_value,
		operator, status, result, is_root_expression, created_at, updated_at
		FROM operations WHERE expression_id = ?`,
		expressionID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var operations []*Operation

	for rows.Next() {
		var op Operation
		var createdAtStr, updatedAtStr string
		var parentOpID sql.NullInt64
		var childPosition sql.NullString
		var leftValue, rightValue, result sql.NullFloat64
		var isRoot bool

		err := rows.Scan(
			&op.ID, &op.ExpressionID, &parentOpID, &childPosition, &leftValue, &rightValue,
			&op.Operator, &op.Status, &result, &isRoot, &createdAtStr, &updatedAtStr,
		)
		if err != nil {
			return nil, err
		}

		// Handle nullable fields
		if parentOpID.Valid {
			val := parentOpID.Int64
			op.ParentOpID = &val
		}

		if leftValue.Valid {
			val := leftValue.Float64
			op.LeftValue = &val
		}

		if rightValue.Valid {
			val := rightValue.Float64
			op.RightValue = &val
		}

		if childPosition.Valid {
			val := childPosition.String
			op.ChildPosition = &val
		}

		if result.Valid {
			val := result.Float64
			op.Result = &val
		}

		// Значение уже имеет тип bool, просто присваиваем
		op.IsRootExpression = isRoot

		// Парсим временные метки в формате RFC3339
		op.CreatedAt, err = time.Parse(time.RFC3339, createdAtStr)
		if err != nil {
			return nil, err
		}

		op.UpdatedAt, err = time.Parse(time.RFC3339, updatedAtStr)
		if err != nil {
			return nil, err
		}

		operations = append(operations, &op)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return operations, nil
}

// UpdateOperationStatus обновляет статус операции
func UpdateOperationStatus(id int64, status string) error {
	DbMutex.Lock()
	defer DbMutex.Unlock()
	_, err := DB.Exec(
		"UPDATE operations SET status = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?",
		status, id,
	)
	return err
}

// SetOperationResult устанавливает только результат операции, не изменяя статус
func SetOperationResult(id int64, result float64) error {
	DbMutex.Lock()
	defer DbMutex.Unlock()

	// Обновляем только значение результата
	_, err := DB.Exec(
		"UPDATE operations SET result = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?",
		result, id,
	)
	return err
}

// OperationInfo содержит основную информацию об операции
type OperationInfo struct {
	ID               int64
	ExpressionID     int64
	IsRootExpression bool
	Status           string
}

// SetOperationError устанавливает ошибку для операции и обновляет её статус
func SetOperationError(operationID int64, errorMsg string) error {
	DbMutex.Lock()
	defer DbMutex.Unlock()

	_, err := DB.Exec(
		"UPDATE operations SET status = ?, error_message = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?",
		StatusError, errorMsg, operationID,
	)
	return err
}

// GetOperationInfo получает основную информацию об операции
func GetOperationInfo(id int64) (*OperationInfo, error) {
	DbMutex.Lock()
	defer DbMutex.Unlock()

	var info OperationInfo
	err := DB.QueryRow(
		"SELECT id, expression_id, is_root_expression, status FROM operations WHERE id = ?",
		id,
	).Scan(&info.ID, &info.ExpressionID, &info.IsRootExpression, &info.Status)

	if err != nil {
		return nil, err
	}

	return &info, nil
}

// CancelOperationsByExpressionID отменяет все операции по expressionID
func CancelOperationsByExpressionID(expressionID int64) error {
	DbMutex.Lock()
	defer DbMutex.Unlock()

	_, err := DB.Exec(
		"UPDATE operations SET status = ?, updated_at = CURRENT_TIMESTAMP WHERE expression_id = ?",
		StatusCanceled, expressionID,
	)
	return err
}

// GetReadyOperation получает одну операцию, которая готова к обработке
func GetReadyOperation() (*Operation, error) {
	// Используем write lock, т.к. сразу после получения операции мы обновим её статус
	DbMutex.Lock()
	defer DbMutex.Unlock()

	var op Operation
	var createdAtStr, updatedAtStr string
	var parentOpID sql.NullInt64
	var childPosition sql.NullString
	var leftValue, rightValue, result sql.NullFloat64
	var isRoot bool

	err := DB.QueryRow(
		`SELECT id, expression_id, parent_operation_id, child_position, left_value, right_value,
		operator, status, result, is_root_expression, created_at, updated_at
		FROM operations 
		WHERE status = ? LIMIT 1`,
		StatusReady,
	).Scan(
		&op.ID, &op.ExpressionID, &parentOpID, &childPosition, &leftValue, &rightValue,
		&op.Operator, &op.Status, &result, &isRoot, &createdAtStr, &updatedAtStr,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Нет готовых операций
		}
		return nil, err
	}

	// Handle nullable fields
	if parentOpID.Valid {
		val := parentOpID.Int64
		op.ParentOpID = &val
	}

	if leftValue.Valid {
		val := leftValue.Float64
		op.LeftValue = &val
	}

	if rightValue.Valid {
		val := rightValue.Float64
		op.RightValue = &val
	}

	if childPosition.Valid {
		val := childPosition.String
		op.ChildPosition = &val
	}

	if result.Valid {
		val := result.Float64
		op.Result = &val
	}

	op.IsRootExpression = isRoot

	// Parse timestamps
	op.CreatedAt, err = time.Parse(time.RFC3339, createdAtStr)
	if err != nil {
		return nil, err
	}

	// Аналогично для UpdatedAt
	op.UpdatedAt, err = time.Parse(time.RFC3339, updatedAtStr)
	if err != nil {
		return nil, err
	}

	return &op, nil
}

// UpdateOperation обновляет операцию в базе данных
func UpdateOperation(op *Operation) error {
	DbMutex.Lock()
	defer DbMutex.Unlock()
	_, err := DB.Exec(
		`UPDATE operations 
		 SET parent_operation_id = ?, child_position = ?, left_value = ?, right_value = ?, 
		 operator = ?, status = ?, result = ?, 
		 error_message = ?, is_root_expression = ?
		 WHERE id = ?`,
		op.ParentOpID, op.ChildPosition, op.LeftValue, op.RightValue,
		op.Operator, op.Status, op.Result, op.ErrorMessage,
		op.IsRootExpression, op.ID,
	)

	return err
}

// UpdateOperationLeftValue устанавливает левый операнд операции
func UpdateOperationLeftValue(operationID int64, value float64) error {
	DbMutex.Lock()
	defer DbMutex.Unlock()

	_, err := DB.Exec(
		`UPDATE operations 
		 SET left_value = ?, updated_at = CURRENT_TIMESTAMP 
		 WHERE id = ?`,
		value, operationID,
	)
	return err
}

// UpdateOperationRightValue устанавливает правый операнд операции
func UpdateOperationRightValue(operationID int64, value float64) error {
	DbMutex.Lock()
	defer DbMutex.Unlock()

	_, err := DB.Exec(
		`UPDATE operations 
		 SET right_value = ?, updated_at = CURRENT_TIMESTAMP 
		 WHERE id = ?`,
		value, operationID,
	)
	return err
}
