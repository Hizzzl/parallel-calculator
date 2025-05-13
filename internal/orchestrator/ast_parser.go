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
// Единый метод для валидации и создания операций в БД
func ParseAST(expressionID int64, node ast.Node) error {
	// Первый проход: проверка корректности без генерации ID
	if err := validateAST(node); err != nil {
		return err
	}

	// Открываем транзакцию для второго прохода
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

	// Фиксируем транзакцию
	return tx.Commit()
}

// validateAST проверяет корректность AST без сохранения в БД
// Вспомогательная функция для первого прохода
func validateAST(node ast.Node) error {
	if node == nil {
		return ErrInvalidExpression
	}

	switch n := node.(type) {
	case *ast.BinaryExpr:
		// Проверяем оператор
		switch n.Op {
		case token.ADD, token.SUB, token.MUL, token.QUO:
			// Поддерживаемые операторы
		default:
			return fmt.Errorf("unsupported operator: %s", n.Op)
		}

		// Рекурсивно проверяем левую и правую части
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
		// Проверяем, что литерал - число
		if n.Kind != token.INT && n.Kind != token.FLOAT {
			return fmt.Errorf("unsupported literal type: %s", n.Kind)
		}

		// Проверяем, что число можно распарсить
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

		// Это одиночное число как корневое выражение
		op, err := db.CreateOperation(
			expressionID, nil, "+", &value, nil, true, nil, StatusCompleted,
		)
		return op.ID, err

	case *ast.ParenExpr:
		return saveASTToDB(tx, expressionID, n.X, parentOpID, childPosition)

	case *ast.BinaryExpr:
		// Создаем операцию для выражения
		var (
			leftVal, rightVal *float64
			childPos          *string
		)

		if parentOpID != nil {
			childPos = &childPosition
		}

		// Проверяем, являются ли операнды литералами
		if value, ok := IsLiteralOnly(n.X); ok {
			leftVal = &value
		}
		if value, ok := IsLiteralOnly(n.Y); ok {
			rightVal = &value
		}

		// Определяем начальный статус
		status := StatusPending
		if leftVal != nil && rightVal != nil {
			status = StatusReady
		}

		// Создаем запись операции
		op, err := db.CreateOperation(
			expressionID, parentOpID, n.Op.String(),
			leftVal, rightVal, parentOpID == nil, childPos, status,
		)
		if err != nil {
			return 0, err
		}

		// Обрабатываем нелитеральные операнды
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
