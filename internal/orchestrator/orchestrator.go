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

// func CreateAST(expression string) (ast.Node, error) {
// 	ast, err := parser.ParseExpr(expression)
// 	if err != nil {
// 		logger.LogINFO(fmt.Sprintf("Error after ParseExpr: %v", err))
// 		return nil, ErrInvalidExpression
// 	}
// 	return ast, nil
// }

// // Обход в глубину AST и создание списка задач.
// // Обратите внимание: этот метод устарел. Вместо него используйте ParseAST с новым подходом двух проходов
// func CalculateExecutionPlan(node ast.Node, plan *ExecutionPlan, parent_id int64, node_position string) (int, error) {

// 	if node == nil {
// 		return 0, nil
// 	}

// 	switch n := node.(type) {
// 	case *ast.BinaryExpr:
// 		// Создаем временный ID, в продакшене должен использоваться ID из БД
// 		tempID := time.Now().UnixNano()
// 		expression := Expression{
// 			id:         tempID,
// 			parentId:   parent_id,
// 			rootId:     plan.RootId,
// 			childSide:  node_position,
// 			isRoot:     parent_id == 0,
// 			leftValue:  make(chan float64, 1),
// 			rightValue: make(chan float64, 1),
// 			operator:   n.Op.String(),
// 			status:     "waiting",
// 			result:     0,
// 		}

// 		if parent_id == 0 {
// 			plan.RootId = expression.id
// 		}

// 		// создаем элемент очереди задач
// 		order := Order{
// 			id:          expression.id,
// 			orderNumber: 0,
// 		}

// 		if isLiteral(n.X) {
// 			leftValue, err := strconv.ParseFloat(n.X.(*ast.BasicLit).Value, 64)
// 			if err != nil {
// 				return 0, err
// 			}
// 			expression.leftValue <- leftValue
// 		} else {
// 			result, err := CalculateExecutionPlan(n.X, plan, expression.id, "left")

// 			if err != nil {
// 				return 0, err
// 			}
// 			order.orderNumber += result
// 		}

// 		if isLiteral(n.Y) {
// 			rightValue, err := strconv.ParseFloat(n.Y.(*ast.BasicLit).Value, 64)
// 			if err != nil {
// 				return 0, err
// 			}
// 			expression.rightValue <- rightValue
// 		} else {
// 			result, err := CalculateExecutionPlan(n.Y, plan, expression.id, "right")
// 			if err != nil {
// 				return 0, err
// 			}
// 			order.orderNumber += result
// 		}

// 		plan.OrderIds = append(plan.OrderIds, order)
// 		plan.Expressions = append(plan.Expressions, expression)

// 		return order.orderNumber + 1, nil
// 	case *ast.ParenExpr:
// 		return CalculateExecutionPlan(n.X, plan, parent_id, node_position)
// 	case *ast.BasicLit:
// 		return 0, ErrOnlyOneLiteral
// 	default:
// 		return 0, ErrInvalidAST
// 	}
// }

// func getFirstLiteralValue(node ast.Node) (float64, error) {
// 	switch n := node.(type) {
// 	case *ast.BasicLit:
// 		value, err := strconv.Atoi(n.Value)
// 		if err != nil {
// 			return 0, err
// 		}
// 		return float64(value), nil
// 	case *ast.ParenExpr:
// 		return getFirstLiteralValue(n.X)
// 	default:
// 		return 0, ErrLiteralNotFound
// 	}
// }

// // isLiteralOnly проверяет, является ли узел AST литералом и возвращает его значение
// func isLiteralOnly(node ast.Node) (float64, bool) {
// 	switch n := node.(type) {
// 	case *ast.BasicLit:
// 		value, err := strconv.ParseFloat(n.Value, 64)
// 		if err != nil {
// 			return 0, false
// 		}
// 		return value, true
// 	case *ast.ParenExpr:
// 		return isLiteralOnly(n.X)
// 	default:
// 		return 0, false
// 	}
// }

// // ProcessExpression обрабатывает математическое выражение и создает операции в базе данных
// func ProcessExpression(expressionID int64, expressionStr string, userID int64) error {
// 	// Парсим AST из выражения
// 	astNode, err := CreateAST(expressionStr)
// 	if err != nil {
// 		// Если произошла ошибка при парсинге, устанавливаем статус ошибки в БД
// 		db.SetExpressionError(expressionID, "Invalid expression syntax: "+err.Error())
// 		return err
// 	}

// 	// Проверяем, если выражение - просто литерал
// 	if value, isLiteral := isLiteralOnly(astNode); isLiteral {
// 		// Создаем константную операцию
// 		op, err := db.CreateOperation(
// 			expressionID,
// 			nil,
// 			"const",
// 			&value, nil,
// 			nil, nil,
// 			true, // корневая операция
// 		)
// 		if err != nil {
// 			return err
// 		}

// 		// Устанавливаем результат операции
// 		err = db.SetOperationResult(op.ID, value)
// 		if err != nil {
// 			return err
// 		}

// 		// Устанавливаем результат выражения
// 		err = db.SetExpressionResult(expressionID, value)
// 		if err != nil {
// 			return err
// 		}

// 		// Выражение уже вычислено, выходим
// 		return nil
// 	}

// 	// Используем парсер AST для создания операций в БД
// 	_, err = ParseAST(expressionID, astNode)
// 	if err != nil {
// 		// Обрабатываем ошибку парсинга
// 		db.SetExpressionError(expressionID, "Error parsing expression: "+err.Error())
// 		return err
// 	}

// 	// Проверяем зависимости всех операций
// 	err = db.CheckAllDependencies()
// 	if err != nil {
// 		return err
// 	}

// 	// Добавляем готовые операции в очередь задач
// 	readyOps, err := db.GetReadyOperations()
// 	if err != nil {
// 		return err
// 	}

// 	// Добавляем операции в очередь задач
// 	for _, op := range readyOps {
// 		// Добавляем задачу в очередь для агентов напрямую с ID из БД
// 		ManagerInstance.AddTask(op.ID)

// 		// Создаем временную структуру для очереди
// 		expr := createCompatExpression(op)
// 		ManagerInstance.StoreExpression(op.ID, expr)
// 	}

// 	return nil
// }

// // createCompatExpression создает совместимую структуру Expression из операции БД
// func createCompatExpression(op *db.Operation) Expression {
// 	// Создаем каналы для значений
// 	leftChan := make(chan float64, 1)
// 	rightChan := make(chan float64, 1)

// 	// Заполняем каналы, если есть значения
// 	if op.LeftValue != nil {
// 		leftChan <- *op.LeftValue
// 	}

// 	if op.RightValue != nil {
// 		rightChan <- *op.RightValue
// 	}

// 	// Определяем позицию узла
// 	nodePosition := "nil"
// 	if op.ParentOpID != nil {
// 		// Попробуем определить, левый это или правый дочерний узел
// 		parentOp, err := db.GetOperationByID(*op.ParentOpID)
// 		if err == nil {
// 			if parentOp.LeftOpID != nil && *parentOp.LeftOpID == op.ID {
// 				nodePosition = "left"
// 			} else if parentOp.RightOpID != nil && *parentOp.RightOpID == op.ID {
// 				nodePosition = "right"
// 			}
// 		}
// 	}

// 	// Используем ID напрямую из БД
// 	var parentID int64 = 0
// 	if op.ParentOpID != nil {
// 		parentID = *op.ParentOpID
// 	}

// 	// Извлекаем результат, если есть
// 	var result float64 = 0
// 	if op.Result != nil {
// 		result = *op.Result
// 	}

// 	// Создаем совместимую структуру
// 	return Expression{
// 		id:         op.ID,
// 		parentId:   parentID,
// 		rootId:     op.ExpressionID, // В качестве rootId используем ID выражения
// 		childSide:  nodePosition,
// 		isRoot:     op.IsRootExpression,
// 		leftValue:  leftChan,
// 		rightValue: rightChan,
// 		operator:   op.Operator,
// 		status:     op.Status,
// 		result:     result,
// 	}
// }
