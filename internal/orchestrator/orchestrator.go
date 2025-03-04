package orchestrator

import (
	"fmt"
	"go/ast"
	"go/parser"
	"parallel-calculator/internal/logger"
	"sort"
	"strconv"

	"github.com/google/uuid"
)

// Данная функция генерирует уникальный id, кроме 0. 0 будет использоваться, чтобы
// отметить, что задача не является подзадачей
func generateID() uint32 {
	for {
		id := uuid.New().ID()
		if id != 0 {
			return id
		}
	}
}

func isLiteral(node ast.Node) bool {
	_, ok := node.(*ast.BasicLit)
	return ok
}

func CreateAST(expression string) (ast.Node, error) {
	ast, err := parser.ParseExpr(expression)
	if err != nil {
		logger.LogINFO(fmt.Sprintf("Error after ParseExpr: %v", err))
		return nil, ErrInvalidExpression
	}
	return ast, nil
}

// Обход в глубину AST и создание списка задач.
// node_position - "left" или "right" относительно родительского узла
// Возвращает количество узлов "операций"
func CalculateExecutionPlan(node ast.Node, plan *ExecutionPlan, parent_id uint32, node_position string) (int, error) {

	if node == nil {
		return 0, nil
	}

	switch n := node.(type) {
	case *ast.BinaryExpr:
		expression := Expression{
			id:         generateID(),
			parentId:   parent_id,
			rootId:     plan.RootId,
			childSide:  node_position,
			isRoot:     parent_id == 0,
			leftValue:  make(chan float64, 1),
			rightValue: make(chan float64, 1),
			operator:   n.Op.String(),
			status:     "waiting",
			result:     0,
		}

		if parent_id == 0 {
			plan.RootId = expression.id
		}

		// создаем элемент очереди задач
		order := Order{
			id:          expression.id,
			orderNumber: 0,
		}

		if isLiteral(n.X) {
			leftValue, err := strconv.ParseFloat(n.X.(*ast.BasicLit).Value, 64)
			if err != nil {
				return 0, err
			}
			expression.leftValue <- leftValue
		} else {
			result, err := CalculateExecutionPlan(n.X, plan, expression.id, "left")

			if err != nil {
				return 0, err
			}
			order.orderNumber += result
		}

		if isLiteral(n.Y) {
			rightValue, err := strconv.ParseFloat(n.Y.(*ast.BasicLit).Value, 64)
			if err != nil {
				return 0, err
			}
			expression.rightValue <- rightValue
		} else {
			result, err := CalculateExecutionPlan(n.Y, plan, expression.id, "right")
			if err != nil {
				return 0, err
			}
			order.orderNumber += result
		}

		plan.OrderIds = append(plan.OrderIds, order)
		plan.Expressions = append(plan.Expressions, expression)

		return order.orderNumber + 1, nil
	case *ast.ParenExpr:
		return CalculateExecutionPlan(n.X, plan, parent_id, node_position)
	case *ast.BasicLit:
		return 0, ErrOnlyOneLiteral
	default:
		return 0, ErrInvalidAST
	}
}

func getFirstLiteralValue(node ast.Node) (float64, error) {
	switch n := node.(type) {
	case *ast.BasicLit:
		value, err := strconv.Atoi(n.Value)
		if err != nil {
			return 0, err
		}
		return float64(value), nil
	case *ast.ParenExpr:
		return getFirstLiteralValue(n.X)
	default:
		return 0, ErrLiteralNotFound
	}
}

// Обработчик выражения, возвращает id выражения
func ProcessExpression(expression string) (uint32, error) {
	ast, err := CreateAST(expression)
	if err != nil {
		return 0, err
	}

	plan := ExecutionPlan{}
	_, err = CalculateExecutionPlan(ast, &plan, 0, "nil")
	if err == ErrOnlyOneLiteral {
		val, _ := getFirstLiteralValue(ast)
		plan.Expressions = append(plan.Expressions, Expression{
			id:         generateID(),
			parentId:   0,
			childSide:  "nil",
			isRoot:     true,
			leftValue:  nil,
			rightValue: nil,
			operator:   "",
			status:     "done",
			result:     val,
		})
		plan.RootId = generateID()
	} else if err != nil {
		return 0, err
	}

	sort.Slice(plan.OrderIds, func(i, j int) bool {
		return plan.OrderIds[i].orderNumber < plan.OrderIds[j].orderNumber
	})

	for _, task := range plan.OrderIds {
		ManagerInstance.AddTask(task.id)
	}

	for _, expression := range plan.Expressions {
		ManagerInstance.StoreExpression(expression.id, expression)
	}

	return plan.RootId, nil
}

func ProcessExpressionResult(result TaskResult) error {
	if result.Error != "nil" {
		rootId, err := ManagerInstance.GetRootId(result.ID)
		if err != nil {
			return err
		}
		err = ManagerInstance.UpdateExpressionStatus(rootId, result.Error)
		if err != nil {
			return err
		}
	}
	err := ManagerInstance.UpdateExpressionResult(result.ID, result.Result)
	if err != nil {
		return err
	}

	parent_id, err := ManagerInstance.GetParentID(result.ID)
	if err != nil {
		return err
	}
	if parent_id == 0 {
		is_root, err := ManagerInstance.IsRoot(result.ID)
		if err != nil {
			return err
		}
		if !is_root {
			return ErrInvalidParentId
		}
		return nil
	}

	node_position, err := ManagerInstance.GetNodePosition(result.ID)
	if err != nil {
		return err
	}

	err = ManagerInstance.UpdateExpressionValue(parent_id, node_position, result.Result)
	if err != nil {
		return err
	}
	return nil
}
