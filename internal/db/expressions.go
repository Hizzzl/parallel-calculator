package db

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
)

var (
	ErrExpressionNotFound = errors.New("expression not found")
)

// CreateExpression создает новое выражение в базе данных
func CreateExpression(userID int64, expression string) (*Expression, error) {
	res, err := DB.Exec(
		`INSERT INTO expressions (user_id, original_expression, status) VALUES (?, ?, ?)`,
		userID, expression, StatusPending,
	)
	if err != nil {
		return nil, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	return &Expression{
		ID:         id,
		UserID:     userID,
		Expression: expression,
		Status:     StatusPending,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}, nil
}

// GetExpressionByID получает выражение по ID
func GetExpressionByID(id int64) (*Expression, error) {
	var expr Expression
	var createdAtStr, updatedAtStr string
	var result sql.NullFloat64
	var errorMessage sql.NullString

	err := DB.QueryRow(
		`SELECT id, user_id, original_expression, status, result, 
         error_message, created_at, updated_at 
         FROM expressions WHERE id = ?`,
		id,
	).Scan(
		&expr.ID, &expr.UserID, &expr.Expression, &expr.Status,
		&result, &errorMessage, &createdAtStr, &updatedAtStr,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrExpressionNotFound
		}
		return nil, err
	}

	if result.Valid {
		val := result.Float64
		expr.Result = &val
	}

	if errorMessage.Valid {
		val := errorMessage.String
		expr.ErrorMessage = &val
	}

	// Сначала пробуем парсить время в формате RFC3339
	expr.CreatedAt, err = time.Parse(time.RFC3339, createdAtStr)
	if err != nil {
		// Если не получилось, пробуем в формате SQL
		expr.CreatedAt, err = time.Parse("2006-01-02 15:04:05", createdAtStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse created_at time '%s': %v", createdAtStr, err)
		}
	}

	expr.UpdatedAt, err = time.Parse(time.RFC3339, updatedAtStr)
	if err != nil {
		// Если не получилось, пробуем в формате SQL
		expr.UpdatedAt, err = time.Parse("2006-01-02 15:04:05", updatedAtStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse updated_at time '%s': %v", updatedAtStr, err)
		}
	}

	return &expr, nil
}

// UpdateExpressionStatus обновляет статус выражения
func UpdateExpressionStatus(id int64, status string) error {
	_, err := DB.Exec(
		"UPDATE expressions SET status = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?",
		status, id,
	)
	return err
}

// SetExpressionResult устанавливает результат выражения
func SetExpressionResult(id int64, result float64) error {
	_, err := DB.Exec(
		`UPDATE expressions 
         SET result = ?, status = ?, updated_at = CURRENT_TIMESTAMP 
         WHERE id = ?`,
		result, StatusCompleted, id,
	)
	return err
}

// SetExpressionError устанавливает ошибку для выражения
func SetExpressionError(id int64, errorMessage string) error {
	_, err := DB.Exec(
		`UPDATE expressions 
         SET error_message = ?, status = ?, updated_at = CURRENT_TIMESTAMP 
         WHERE id = ?`,
		errorMessage, StatusError, id,
	)
	return err
}

// GetUserExpressions получает все выражения пользователя
func GetUserExpressions(userID int64) ([]*Expression, error) {
	rows, err := DB.Query(
		`SELECT id, user_id, original_expression, status, result, 
         error_message, created_at, updated_at 
         FROM expressions 
         WHERE user_id = ? 
         ORDER BY created_at DESC`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var expressions []*Expression

	for rows.Next() {
		var expr Expression
		var createdAtStr, updatedAtStr string
		var result sql.NullFloat64
		var errorMessage sql.NullString

		err := rows.Scan(
			&expr.ID, &expr.UserID, &expr.Expression, &expr.Status,
			&result, &errorMessage, &createdAtStr, &updatedAtStr,
		)
		if err != nil {
			return nil, err
		}

		if result.Valid {
			val := result.Float64
			expr.Result = &val
		}

		if errorMessage.Valid {
			val := errorMessage.String
			expr.ErrorMessage = &val
		}

		// Сначала пробуем парсить время в формате RFC3339
		expr.CreatedAt, err = time.Parse(time.RFC3339, createdAtStr)
		if err != nil {
			// Если не получилось, пробуем в формате SQL
			expr.CreatedAt, err = time.Parse("2006-01-02 15:04:05", createdAtStr)
			if err != nil {
				return nil, fmt.Errorf("failed to parse created_at time '%s': %v", createdAtStr, err)
			}
		}

		// Аналогично для UpdatedAt
		expr.UpdatedAt, err = time.Parse(time.RFC3339, updatedAtStr)
		if err != nil {
			// Если не получилось, пробуем в формате SQL
			expr.UpdatedAt, err = time.Parse("2006-01-02 15:04:05", updatedAtStr)
			if err != nil {
				return nil, fmt.Errorf("failed to parse updated_at time '%s': %v", updatedAtStr, err)
			}
		}

		expressions = append(expressions, &expr)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return expressions, nil
}
