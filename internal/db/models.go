package db

import (
	"time"
)

// User представляет пользователя в системе
type User struct {
	ID           int64     `json:"id"`
	Login        string    `json:"login"`
	PasswordHash string    `json:"-"` // Не включается в JSON сериализацию
	CreatedAt    time.Time `json:"created_at"`
}

// Expression представляет математическое выражение
type Expression struct {
	ID           int64     `json:"id"`
	UserID       int64     `json:"user_id"`
	Expression   string    `json:"expression"`
	Status       string    `json:"status"`
	Result       *float64  `json:"result"`
	ErrorMessage *string   `json:"error_message"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Operation представляет отдельную операцию в выражении
type Operation struct {
	ID               int64     `json:"id"`
	ExpressionID     int64     `json:"expression_id"`
	ParentOpID       *int64    `json:"parent_operation_id"`
	ChildPosition    *string   `json:"child_position"` // "left" или "right" или nil для корневой операции
	LeftValue        *float64  `json:"left_value"`
	RightValue       *float64  `json:"right_value"`
	Operator         string    `json:"operator"`
	Status           string    `json:"status"`
	Result           *float64  `json:"result"`
	ErrorMessage     *string   `json:"error_message"`
	IsRootExpression bool      `json:"is_root_expression"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// Статусы для выражений и операций
const (
	StatusPending    = "pending"
	StatusReady      = "ready"
	StatusProcessing = "processing"
	StatusCompleted  = "completed"
	StatusError      = "error"
	StatusCanceled   = "canceled"
)
