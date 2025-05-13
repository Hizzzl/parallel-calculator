package orchestrator

import (
	"database/sql"
	"fmt"
	"go/ast"
	"go/token"
	"parallel-calculator/internal/db"
	"parallel-calculator/internal/logger"
	"strconv"
)

// Статусы для выражений и операций
const (
	StatusPending    = "pending"
	StatusReady      = "ready"
	StatusProcessing = "processing"
	StatusCompleted  = "completed"
	StatusError      = "error"
	StatusCanceled   = "canceled"
)

// ErrInvalidExpression ошибка для случая неверного выражения
var ErrInvalidExpression = fmt.Errorf("invalid expression")

// ParseAST парсит AST и создает операции в базе данных
func ParseAST(expressionID int64, node ast.Node) error {
	if err := validateAST(node); err != nil {
		return err
	}

	tx, err := db.DB.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	// Второй проход: сохранение в БД с транзакцией
	if _, err := saveASTToDB(tx, expressionID, node, nil, "nil"); err != nil {
		return err
	}

	return tx.Commit()
}

// validateAST проверяет корректность AST без сохранения в БД
func validateAST(node ast.Node) error {
	if node == nil {
		return ErrInvalidExpression
	}

	switch n := node.(type) {
	case *ast.BinaryExpr:
		switch n.Op {
		case token.ADD, token.SUB, token.MUL, token.QUO:
		default:
			return fmt.Errorf("unsupported operator: %s", n.Op)
		}

		if err := validateAST(n.X); err != nil {
			return err
		}

		if err := validateAST(n.Y); err != nil {
			return err
		}

		return nil

	case *ast.ParenExpr:
		// Проверяем содержимое скобок
		return validateAST(n.X)

	case *ast.BasicLit:
		if n.Kind != token.INT && n.Kind != token.FLOAT {
			return fmt.Errorf("unsupported literal type: %s", n.Kind)
		}

		_, err := strconv.ParseFloat(n.Value, 64)
		if err != nil {
			return fmt.Errorf("invalid numeric value: %s - %v", n.Value, err)
		}
		return nil

	default:
		logger.LogINFO(fmt.Sprintf("Неизвестный тип узла AST: %T", n))
		return fmt.Errorf("unsupported expression element: %T", n)
	}
}

// saveASTToDB сохраняет AST в базу данных, возвращает ID корневой операции
func saveASTToDB(tx *sql.Tx, expressionID int64, node ast.Node, parentOpID *int64, childPosition string) (int64, error) {
	switch n := node.(type) {
	case *ast.BasicLit:
		value, _ := strconv.ParseFloat(n.Value, 64)

		err := db.UpdateExpressionStatus(
			expressionID, StatusCompleted,
		)
		if err != nil {
			return 0, err
		}
		err = db.SetExpressionResult(expressionID, value)
		return 0, err

	case *ast.ParenExpr:
		return saveASTToDB(tx, expressionID, n.X, parentOpID, childPosition)

	case *ast.BinaryExpr:
		var (
			leftVal, rightVal *float64
			childPos          *string
		)

		if parentOpID != nil {
			childPos = &childPosition
		}

		if value, ok := IsLiteralOnly(n.X); ok {
			leftVal = &value
		}
		if value, ok := IsLiteralOnly(n.Y); ok {
			rightVal = &value
		}

		status := StatusPending
		if leftVal != nil && rightVal != nil {
			status = StatusReady
		}

		op, err := db.CreateOperation(
			expressionID, parentOpID, n.Op.String(),
			leftVal, rightVal, parentOpID == nil, childPos, status,
		)
		if err != nil {
			return 0, err
		}

		if leftVal == nil {
			if _, err := saveASTToDB(tx, expressionID, n.X, &op.ID, "left"); err != nil {
				return 0, err
			}
		}

		if rightVal == nil {
			if _, err := saveASTToDB(tx, expressionID, n.Y, &op.ID, "right"); err != nil {
				return 0, err
			}
		}

		return op.ID, nil

	default:
		return 0, fmt.Errorf("unsupported node type: %T", node)
	}
}

// IsLiteralOnly проверяет, является ли узел AST литералом и возвращает его значение
func IsLiteralOnly(node ast.Node) (float64, bool) {
	switch n := node.(type) {
	case *ast.BasicLit:
		value, err := strconv.ParseFloat(n.Value, 64)
		if err != nil {
			return 0, false
		}
		return value, true
	case *ast.ParenExpr:
		return IsLiteralOnly(n.X)
	default:
		return 0, false
	}
}
