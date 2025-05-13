package orchestrator

import (
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"parallel-calculator/internal/db"
	"parallel-calculator/internal/logger"
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

	// Парсим AST и создаем операции в базе данных
	err = ParseAST(expression.ID, astNode)
	if err != nil {
		// Обновляем статус ошибки в БД
		db.SetExpressionError(expression.ID, "Error creating operations: "+err.Error())
		return &expression.ID, nil
	}

	return &expression.ID, nil
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
		// Обрабатываем ошибку операции и отменяем все связанные операции
		err := HandleOperationErrorWithCancellation(result.ID, result.Error)
		if err != nil {
			return fmt.Errorf("ошибка при обработке ошибки операции: %w", err)
		}

		return nil
	}

	// 1. Устанавливаем результат операции
	err = SetOperationResultInDB(result.ID, result.Result)
	if err != nil {
		return fmt.Errorf("ошибка при установке результата операции: %w", err)
	}

	// 2. Обновляем статус операции на "completed"
	err = UpdateOperationStatusInDB(result.ID, db.StatusCompleted)
	if err != nil {
		return fmt.Errorf("ошибка при обновлении статуса операции: %w", err)
	}

	// 3. Если это корневая операция выражения, обновляем результат выражения
	if op.IsRootExpression {
		// Обновляем результат выражения и устанавливаем статус "completed"
		err = FinalizeExpression(op.ExpressionID, result.Result)
		if err != nil {
			return fmt.Errorf("ошибка при финализации выражения: %w", err)
		}
	} else if op.ParentOpID != nil {
		// 4. Если это не корневая операция, но у неё есть родитель,
		// обновляем аргументы родительской операции
		err = UpdateParentOperation(op, result.Result)
		if err != nil {
			return fmt.Errorf("ошибка при обновлении родительской операции: %w", err)
		}
	}

	// 5. Проверяем зависимости всех операций после обновления результата
	return nil
}
