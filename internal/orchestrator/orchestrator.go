package orchestrator

import (
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"parallel-calculator/internal/db"
	"parallel-calculator/internal/logger"
	"strconv"
)

// Errors
var (
	ErrQueueIsEmpty            = errors.New("queue is empty")
	ErrExpressionNotFound      = errors.New("expression not found")
	ErrInvalidAST              = errors.New("invalid AST")
	ErrLiteralNotFound         = errors.New("literal not found")
	ErrParentNotFound          = errors.New("parent expression not found")
	ErrInvalidNodePosition     = errors.New("invalid node position")
	ErrInvalidChannelCondition = errors.New("invalid channel condition")
	ErrOnlyOneLiteral          = errors.New("only one literal allowed")
	ErrInvalidParentId         = errors.New("invalid parent id")
)

func CreateAST(expression string) (ast.Node, error) {
	ast, err := parser.ParseExpr(expression)
	if err != nil {
		logger.LogINFO(fmt.Sprintf("Error after ParseExpr: %v", err))
		return nil, ErrInvalidExpression
	}
	return ast, nil
}

// Обрабатывает выражение. Возвращает id и ошибку
func ProcessExpression(expr string, userID int64) (*int64, error) {
	// Добавляем выражение в базу данных
	expression, err := CreateExpressionInDB(userID, expr)

	if err != nil {
		return nil, err
	}
	// Парсим AST из выражения
	astNode, err := CreateAST(expression.Expression)

	if err != nil {
		// Если произошла ошибка при парсинге, устанавливаем статус ошибки в БД
		db.SetExpressionError(expression.ID, "Invalid expression syntax: "+err.Error())
		// Возвращаем ошибку недействительного выражения
		return nil, ErrInvalidExpression
	}

	// Проверяем, если выражение - просто литерал
	if value, isLiteral := isLiteral(astNode); isLiteral {
		// Создаем константную операцию
		op, err := db.CreateOperation(
			expression.ID,
			nil,
			"const",
			value, nil,
			true, // корневая операция
			nil,  // без позиции (родительская)
		)
		if err != nil {
			return &expression.ID, err
		}

		// Устанавливаем результат операции
		err = db.SetOperationResult(op.ID, *value)
		if err != nil {
			return &expression.ID, err
		}

		// Выражение уже вычислено, выходим
		return &expression.ID, nil
	}

	// Проходим по дереву и смотрим, корректно ли выражение
	err = validateAST(astNode)
	if err != nil {
		// Обновляем статус ошибки в БД
		db.SetExpressionError(expression.ID, "Invalid expression structure: "+err.Error())
		// Возвращаем ошибку недействительного выражения
		return nil, ErrInvalidExpression
	}

	// Начинаем транзакцию
	tx, err := db.DB.Begin()
	if err != nil {
		return &expression.ID, err
	}
	defer func() {
		if err != nil {
			tx.Rollback()
			return
		}
		tx.Commit()
	}()

	// Парсим AST и создаем операции в базе данных
	err = ParseAST(expression.ID, astNode)
	if err != nil {
		// Обновляем статус ошибки в БД
		db.SetExpressionError(expression.ID, "Error creating operations: "+err.Error())
		return &expression.ID, nil
	}

	return &expression.ID, nil
}

// isLiteral проверяет, является ли узел литералом
func isLiteral(node ast.Node) (*float64, bool) {
	switch n := node.(type) {
	case *ast.BasicLit:
		if len(n.Value) != 1 {
			return nil, false
		}
		value, err := strconv.ParseFloat(n.Value, 64)
		if err != nil {
			return nil, false
		}
		return &value, true
	case *ast.ParenExpr:
		v, ok := isLiteral(n.X)
		return v, ok
	default:
		return nil, false
	}
}

// ProcessExpressionResult обрабатывает результат выполнения задачи
func ProcessExpressionResult(result TaskResult) error {
	// Получаем операцию, чтобы иметь доступ к ID выражения и другим данным
	op, err := db.GetOperationByID(result.ID)
	if err != nil {
		return fmt.Errorf("не удалось получить операцию: %w", err)
	}

	// Теперь ID из TaskResult напрямую является ID в базе данных
	// Проверяем наличие ошибки в результате
	if result.Error != "nil" && result.Error != "" {
		// 1. Ставим задаче статус ошибку
		db.UpdateOperationStatus(result.ID, db.StatusError)

		// 2. Обрабатываем ошибку операции и отменяем все связанные операции
		err := db.HandleOperationError(result.ID, result.Error)
		if err != nil {
			return fmt.Errorf("ошибка при обработке ошибки операции: %w", err)
		}

		// 3. Обновляем статус выражения
		err = db.SetExpressionError(op.ExpressionID, result.Error)
		if err != nil {
			return fmt.Errorf("ошибка при установке ошибки выражения: %w", err)
		}

		return nil
	}

	// Если нет ошибки:

	// 1. Устанавливаем результат операции
	err = db.SetOperationResult(result.ID, result.Result)
	if err != nil {
		return fmt.Errorf("ошибка при установке результата операции: %w", err)
	}

	// 2. Обновляем статус операции на "completed"
	err = db.UpdateOperationStatus(result.ID, db.StatusCompleted)
	if err != nil {
		return fmt.Errorf("ошибка при обновлении статуса операции: %w", err)
	}

	// 3. Если это корневая операция выражения, обновляем результат выражения
	if op.IsRootExpression {
		// Обновляем результат выражения
		err = db.SetExpressionResult(op.ExpressionID, result.Result)
		if err != nil {
			return fmt.Errorf("ошибка при установке результата выражения: %w", err)
		}

		// Обновляем статус выражения на "completed"
		err = db.UpdateExpressionStatus(op.ExpressionID, db.StatusCompleted)
		if err != nil {
			return fmt.Errorf("ошибка при обновлении статуса выражения: %w", err)
		}
	} else if op.ParentOpID != nil {
		// 4. Если это не корневая операция, но у неё есть родитель,
		// обновляем аргументы родительской операции

		// Обновляем соответствующий аргумент в родительской операции
		if op.ChildPosition != nil {
			switch *op.ChildPosition {
			case "left":
				err = db.UpdateOperationLeftValue(*op.ParentOpID, result.Result)
				if err != nil {
					return fmt.Errorf("ошибка при обновлении левого аргумента родителя: %w", err)
				}
			case "right":
				err = db.UpdateOperationRightValue(*op.ParentOpID, result.Result)
				if err != nil {
					return fmt.Errorf("ошибка при обновлении правого аргумента родителя: %w", err)
				}
			}

			// После обновления операции проверяем, есть ли у родителя оба значения
			// Если оба значения присутствуют, устанавливаем статус Ready
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
		}
	}

	// 5. Проверяем зависимости всех операций после обновления результата
	return nil
}
