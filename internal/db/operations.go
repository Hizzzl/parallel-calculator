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
	leftValue, rightValue *float64, isRoot bool, childPosition *string) (*Operation, error) {

	// Если оба значения аргументов предоставлены, операция готова к обработке
	initialStatus := StatusPending
	if leftValue != nil && rightValue != nil {
		initialStatus = StatusReady
	}

	query := `
		INSERT INTO operations 
		(expression_id, parent_operation_id, child_position, left_value, right_value,
		operator, status, is_root_expression) 
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	// Выполняем запрос с обработкой нулевых параметров
	res, err := DB.Exec(
		query,
		expressionID, parentOpID, childPosition, leftValue, rightValue,
		operator, initialStatus, isRoot,
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
		Status:           initialStatus,
		IsRootExpression: isRoot,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}, nil
}

// GetOperationByID получает операцию по ID
func GetOperationByID(id int64) (*Operation, error) {
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

	// Значение уже имеет тип bool, просто присваиваем
	op.IsRootExpression = isRoot

	// Парсим временные метки используя формат RFC3339
	op.CreatedAt, err = time.Parse(time.RFC3339, createdAtStr)
	if err != nil {
		// Если стандартный формат не подошел, пробуем формат SQL
		op.CreatedAt, err = time.Parse("2006-01-02 15:04:05", createdAtStr)
		if err != nil {
			return nil, err
		}
	}

	op.UpdatedAt, err = time.Parse(time.RFC3339, updatedAtStr)
	if err != nil {
		// Если стандартный формат не подошел, пробуем формат SQL
		op.UpdatedAt, err = time.Parse("2006-01-02 15:04:05", updatedAtStr)
		if err != nil {
			return nil, err
		}
	}

	return &op, nil
}

// GetOperationsByExpressionID получает все операции для выражения
func GetOperationsByExpressionID(expressionID int64) ([]*Operation, error) {
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

		// Parse timestamps
		// Сначала пробуем формат RFC3339 (ISO 8601)
		op.CreatedAt, err = time.Parse(time.RFC3339, createdAtStr)
		if err != nil {
			// Если не получилось, пробуем другой формат
			op.CreatedAt, err = time.Parse("2006-01-02 15:04:05", createdAtStr)
			if err != nil {
				return nil, err
			}
		}

		// Аналогично для UpdatedAt
		op.UpdatedAt, err = time.Parse(time.RFC3339, updatedAtStr)
		if err != nil {
			op.UpdatedAt, err = time.Parse("2006-01-02 15:04:05", updatedAtStr)
			if err != nil {
				return nil, err
			}
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
	_, err := DB.Exec(
		"UPDATE operations SET status = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?",
		status, id,
	)

	// Если это корневая операция и её статус 'завершена',
	// обновляем соответствующее выражение
	if status == StatusCompleted {
		var isRoot bool
		var expressionID int64
		var result float64

		err := DB.QueryRow(
			"SELECT is_root_expression, expression_id, result FROM operations WHERE id = ?",
			id,
		).Scan(&isRoot, &expressionID, &result)

		if err != nil {
			return err
		}

		if isRoot {
			err = SetExpressionResult(expressionID, result)
			if err != nil {
				return err
			}
		}
	}

	return err
}

// SetOperationResult устанавливает результат операции
func SetOperationResult(id int64, result float64) error {
	_, err := DB.Exec(
		`UPDATE operations 
		SET result = ?, status = ?, updated_at = CURRENT_TIMESTAMP 
		WHERE id = ?`,
		result, StatusCompleted, id,
	)

	if err != nil {
		return err
	}

	// Проверяем, является ли это корневой операцией
	var isRoot bool
	var expressionID int64

	err = DB.QueryRow(
		"SELECT is_root_expression, expression_id FROM operations WHERE id = ?",
		id,
	).Scan(&isRoot, &expressionID)

	if err != nil {
		return err
	}

	// Если это корневая операция, обновляем результат выражения
	if isRoot {
		err = SetExpressionResult(expressionID, result)
		if err != nil {
			return err
		}
	}

	return nil
}

// GetReadyOperation получает одну операцию, которая готова к обработке
func GetReadyOperation() (*Operation, error) {
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

	// Значение уже имеет тип bool, просто присваиваем
	op.IsRootExpression = isRoot

	// Parse timestamps
	// Сначала пробуем формат RFC3339 (ISO 8601)
	op.CreatedAt, err = time.Parse(time.RFC3339, createdAtStr)
	if err != nil {
		// Если первый формат не подошел, пробуем другой
		op.CreatedAt, err = time.Parse("2006-01-02 15:04:05", createdAtStr)
		if err != nil {
			return nil, err
		}
	}

	// Аналогично для UpdatedAt
	op.UpdatedAt, err = time.Parse(time.RFC3339, updatedAtStr)
	if err != nil {
		op.UpdatedAt, err = time.Parse("2006-01-02 15:04:05", updatedAtStr)
		if err != nil {
			return nil, err
		}
	}

	return &op, nil
}

// CheckDependencies проверяет, завершены ли все зависимости операции
// и обновляет её статус на 'готова', если они завершены
func CheckDependencies(id int64) error {
	// Получаем операцию
	op, err := GetOperationByID(id)
	if err != nil {
		return err
	}

	// Если уже готова или обрабатывается, проверять не нужно
	if op.Status == StatusReady || op.Status == StatusProcessing {
		return nil
	}

	// Находим все дочерние операции для данной операции
	rows, err := DB.Query(
		`SELECT id, status FROM operations 
		 WHERE parent_operation_id = ?`,
		op.ID,
	)
	if err != nil {
		return err
	}
	defer rows.Close()

	isReady := true
	hasChildren := false

	// Проверяем статус всех дочерних операций
	for rows.Next() {
		hasChildren = true
		var childID int64
		var status string

		if err := rows.Scan(&childID, &status); err != nil {
			return err
		}

		if status != StatusCompleted {
			isReady = false
			break
		}
	}

	if err = rows.Err(); err != nil {
		return err
	}

	// Если все дочерние операции завершены или их нет, обновляем статус на 'готова'
	if isReady && hasChildren {
		return UpdateOperationStatus(id, StatusReady)
	}

	return nil
}

// HandleOperationError обрабатывает ошибку операции и отменяет все связанные операции
func HandleOperationError(operationID int64, errorMsg string) error {
	// Получаем операцию для определения ID выражения
	op, err := GetOperationByID(operationID)
	if err != nil {
		return err
	}

	// Начинаем транзакцию
	tx, err := DB.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		} else {
			tx.Commit()
		}
	}()

	// 1. Устанавливаем ошибку для выражения
	_, err = tx.Exec(
		`UPDATE expressions 
		 SET error_message = ?, status = ?, updated_at = CURRENT_TIMESTAMP 
		 WHERE id = ?`,
		errorMsg, StatusError, op.ExpressionID,
	)
	if err != nil {
		return err
	}

	// 2. Отмечаем все операции выражения как отмененные или ошибочные
	// Операция, вызвавшая ошибку, получает статус "error"
	_, err = tx.Exec(
		`UPDATE operations
		 SET status = ?, error_message = ?, updated_at = CURRENT_TIMESTAMP
		 WHERE id = ?`,
		StatusError, errorMsg, operationID,
	)
	if err != nil {
		return err
	}

	// Все остальные операции получают статус "canceled"
	_, err = tx.Exec(
		`UPDATE operations
		 SET status = ?, updated_at = CURRENT_TIMESTAMP
		 WHERE expression_id = ? AND id != ?`,
		StatusCanceled, op.ExpressionID, operationID,
	)
	if err != nil {
		return err
	}

	return nil
}

// UpdateOperation обновляет операцию в базе данных
func UpdateOperation(op *Operation) error {
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

// UpdateOperationLeftValue обновляет левый аргумент операции
func UpdateOperationLeftValue(operationID int64, value float64) error {
	_, err := DB.Exec(
		`UPDATE operations 
		 SET left_value = ?, updated_at = CURRENT_TIMESTAMP 
		 WHERE id = ?`,
		value, operationID,
	)
	return err
}

// UpdateOperationRightValue обновляет правый аргумент операции
func UpdateOperationRightValue(operationID int64, value float64) error {
	_, err := DB.Exec(
		`UPDATE operations 
		 SET right_value = ?, updated_at = CURRENT_TIMESTAMP 
		 WHERE id = ?`,
		value, operationID,
	)
	return err
}
