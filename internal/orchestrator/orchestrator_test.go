package orchestrator

import (
	"go/ast"
	"go/token"
	"testing"

	"go/parser"
)

func TestGenerateID(t *testing.T) {
	t.Run("simple test", func(t *testing.T) {
		for i := 0; i < 1000; i++ {
			id := generateID()
			if id == 0 {
				t.Error("expected id to be greater than 0")
			}
		}
	})
}

func TestIsLiteral(t *testing.T) {
	cases := []struct {
		name     string
		node     ast.Node
		expected bool
	}{
		{
			name:     "simple test true",
			node:     &ast.BasicLit{},
			expected: true,
		},
		{
			name:     "simple test false",
			node:     &ast.Ident{},
			expected: false,
		},
		{
			name:     "simple test false",
			node:     nil,
			expected: false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := isLiteral(tc.node)
			if result != tc.expected {
				t.Errorf("expected %v, got %v", tc.expected, result)
			}
		})
	}
}

func TestCreateAST(t *testing.T) {
	cases := []struct {
		name         string
		expression   string
		expected_err bool
	}{
		{
			name:         "simple test true",
			expression:   "1 + 1",
			expected_err: false,
		},
		{
			name:         "simple test false",
			expression:   "1 +",
			expected_err: true,
		},
		{
			name:         "empty expression",
			expression:   "",
			expected_err: true,
		},
		{
			name:         "simple invalid expression",
			expression:   "(1",
			expected_err: true,
		},
		{
			name:         "simple literal with brackets",
			expression:   "(5)",
			expected_err: false,
		},
		{
			name:         "simple literal",
			expression:   "5",
			expected_err: true,
		},
		{
			name:         "long correct expression",
			expression:   "(1 + 1) * 2 + 5 / 10 *(1 + 2+(2-4))",
			expected_err: false,
		},
		{
			name:         "long incorrect expression with brackets",
			expression:   "((1 + 1) * 2 + + 5 / 10 *(1 + 2+(2-4)))",
			expected_err: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := CreateAST(tc.expression)
			if err != nil {
				if !tc.expected_err {
					t.Errorf("expected no error, got error %v", err)
				}
			}
		})
	}
}

func TestCalculateExecutionPlanCases(t *testing.T) {
	t.Run("ValidExpression", func(t *testing.T) {
		// Строим AST для выражения: ((1+2) * ((3-4)/5))

		// Левое подвыражение: 1 + 2
		leftExpr := &ast.BinaryExpr{
			X:  &ast.BasicLit{Kind: token.INT, Value: "1"},
			Op: token.ADD,
			Y:  &ast.BasicLit{Kind: token.INT, Value: "2"},
		}

		// Правое подвыражение: (3 - 4)
		subExpr := &ast.BinaryExpr{
			X:  &ast.BasicLit{Kind: token.INT, Value: "3"},
			Op: token.SUB,
			Y:  &ast.BasicLit{Kind: token.INT, Value: "4"},
		}

		// Выражение: (subExpr) / 5
		rightExpr := &ast.BinaryExpr{
			X:  subExpr,
			Op: token.QUO,
			Y:  &ast.BasicLit{Kind: token.INT, Value: "5"},
		}

		// Корневое выражение: (leftExpr) * (rightExpr)
		rootExpr := &ast.BinaryExpr{
			X:  leftExpr,
			Op: token.MUL,
			Y:  rightExpr,
		}

		plan := ExecutionPlan{}

		count, err := CalculateExecutionPlan(rootExpr, &plan, 0, "nil")
		if err != nil {
			t.Errorf("Ошибка при обработке корректного выражения: %v", err)
		}

		if count != 4 {
			t.Errorf("Ожидается 4 оператора, но получено %d", count)
		}

		if len(plan.OrderIds) != 4 {
			t.Errorf("Ожидается 4 задачи в плане, получено %d", len(plan.OrderIds))
		}

		if len(plan.Expressions) != 4 {
			t.Errorf("Ожидается 4 выражения в плане, получено %d", len(plan.Expressions))
		}

		// Проверка корневого выражения
		var rootExp Expression
		found := false
		for _, exp := range plan.Expressions {
			if exp.isRoot { // предполагается, что у Expression есть метод IsRoot()
				rootExp = exp
				found = true
				break
			}
		}
		if !found {
			t.Error("Корневое выражение не найдено")
		}
		if rootExp.id != plan.RootId {
			t.Error("RootId плана не совпадает с id корневого выражения")
		}
	})

	t.Run("NilNode", func(t *testing.T) {
		plan := ExecutionPlan{}
		count, err := CalculateExecutionPlan(nil, &plan, 0, "nil")
		if err != nil {
			t.Errorf("При передаче nil не ожидается ошибка, получена: %v", err)
		}
		if count != 0 {
			t.Errorf("При nil узле ожидается 0, получено %d", count)
		}
		if len(plan.OrderIds) != 0 || len(plan.Expressions) != 0 {
			t.Error("План должен оставаться пустым для nil узла")
		}
	})

	t.Run("InvalidExpression", func(t *testing.T) {
		// Передаем узел, не являющийся операторным выражением (Ident)
		invalidNode := &ast.Ident{}
		plan := ExecutionPlan{}
		count, err := CalculateExecutionPlan(invalidNode, &plan, 0, "nil")
		if err == nil {
			t.Error("Ожидается ошибка для неподдерживаемого узла")
		}
		if count != 0 {
			t.Errorf("При ошибке ожидается счетчик 0, получено %d", count)
		}
		if len(plan.OrderIds) != 0 || len(plan.Expressions) != 0 {
			t.Error("План должен оставаться пустым для неподдерживаемого узла")
		}
	})

	t.Run("ParenExpression", func(t *testing.T) {
		// Строим выражение: (1+2) с помощью ParenExpr
		binExpr := &ast.BinaryExpr{
			X:  &ast.BasicLit{Kind: token.INT, Value: "1"},
			Op: token.ADD,
			Y:  &ast.BasicLit{Kind: token.INT, Value: "2"},
		}
		paren := &ast.ParenExpr{X: binExpr}
		plan := ExecutionPlan{}

		count, err := CalculateExecutionPlan(paren, &plan, 0, "nil")
		if err != nil {
			t.Errorf("Не ожидается ошибка для паренизированного выражения, получена: %v", err)
		}
		if count != 1 {
			t.Errorf("Ожидается 1 оператор, получено %d", count)
		}
		if len(plan.OrderIds) != 1 || len(plan.Expressions) != 1 {
			t.Error("План должен содержать 1 задачу и 1 выражение для паренизированного выражения")
		}
	})

	t.Run("LiteralExpression", func(t *testing.T) {
		// Создаем узел литерал: 42
		lit := &ast.BasicLit{Kind: token.INT, Value: "42"}
		plan := ExecutionPlan{}
		count, err := CalculateExecutionPlan(lit, &plan, 0, "nil")
		if err != ErrOnlyOneLiteral {
			t.Errorf("Ожидается ошибка ErrOnlyOneLiteral, получена: %v", err)
		}
		if count != 0 {
			t.Errorf("Ожидается, что счетчик равен 0, получено: %d", count)
		}
		if len(plan.OrderIds) != 0 || len(plan.Expressions) != 0 {
			t.Error("План задач должен оставаться пустым для литерала")
		}
	})

	t.Run("StringExpression", func(t *testing.T) {
		// Создаем узел литерал: 42
		lit := &ast.BasicLit{Kind: token.STRING, Value: "abc"}
		plan := ExecutionPlan{}
		count, err := CalculateExecutionPlan(lit, &plan, 0, "nil")
		if err != ErrOnlyOneLiteral {
			t.Errorf("Ожидается ошибка ErrOnlyOneLiteral, получена: %v", err)
		}
		if count != 0 {
			t.Errorf("Ожидается, что счетчик равен 0, получено: %d", count)
		}
		if len(plan.OrderIds) != 0 || len(plan.Expressions) != 0 {
			t.Error("План задач должен оставаться пустым для литерала")
		}
	})

	t.Run("LiteralInParen", func(t *testing.T) {
		// Создаем узел литерал обёрнутый в круглые скобки: (42)
		lit := &ast.BasicLit{Kind: token.INT, Value: "42"}
		paren := &ast.ParenExpr{X: lit}
		plan := ExecutionPlan{}
		count, err := CalculateExecutionPlan(paren, &plan, 0, "nil")
		if err != ErrOnlyOneLiteral {
			t.Errorf("Ожидается ошибка ErrOnlyOneLiteral для литерала в скобках, получена: %v", err)
		}
		if count != 0 {
			t.Errorf("Ожидается, что счетчик равен 0 для литерала в скобках, получено: %d", count)
		}
		if len(plan.OrderIds) != 0 || len(plan.Expressions) != 0 {
			t.Error("План задач должен оставаться пустым для литерала в скобках")
		}
	})
	t.Run("NotIntLiteralLeft", func(t *testing.T) {
		ast, _ := parser.ParseExpr("\"abc\"+1")

		plan := ExecutionPlan{}
		count, err := CalculateExecutionPlan(ast, &plan, 0, "nil")
		if err == nil {
			t.Errorf("Ожидается ошибка ErrOnlyOneLiteral для литерала в скобках, получена: %v", err)
		}
		if count != 0 {
			t.Errorf("Ожидается, что счетчик равен 0 для литерала в скобках, получено: %d", count)
		}
		if len(plan.OrderIds) != 0 || len(plan.Expressions) != 0 {
			t.Error("План задач должен оставаться пустым для литерала в скобках")
		}
	})
	t.Run("NotIntLiteralRight", func(t *testing.T) {
		ast, _ := parser.ParseExpr("1+\"abc\"")

		plan := ExecutionPlan{}
		count, err := CalculateExecutionPlan(ast, &plan, 0, "nil")
		if err == nil {
			t.Errorf("Ожидается ошибка ErrOnlyOneLiteral для литерала в скобках, получена: %v", err)
		}
		if count != 0 {
			t.Errorf("Ожидается, что счетчик равен 0 для литерала в скобках, получено: %d", count)
		}
		if len(plan.OrderIds) != 0 || len(plan.Expressions) != 0 {
			t.Error("План задач должен оставаться пустым для литерала в скобках")
		}
	})
	t.Run("NotIntLiteralLeftParen", func(t *testing.T) {
		ast, _ := parser.ParseExpr("(\"abc\")+1")

		plan := ExecutionPlan{}
		count, err := CalculateExecutionPlan(ast, &plan, 0, "nil")
		if err == nil {
			t.Errorf("Ожидается ошибка ErrOnlyOneLiteral для литерала в скобках, получена: %v", err)
		}
		if count != 0 {
			t.Errorf("Ожидается, что счетчик равен 0 для литерала в скобках, получено: %d", count)
		}
		if len(plan.OrderIds) != 0 || len(plan.Expressions) != 0 {
			t.Error("План задач должен оставаться пустым для литерала в скобках")
		}
	})
	t.Run("NotIntLiteralRightParen", func(t *testing.T) {
		ast, _ := parser.ParseExpr("1+(\"abc\")")

		plan := ExecutionPlan{}
		count, err := CalculateExecutionPlan(ast, &plan, 0, "nil")
		if err == nil {
			t.Errorf("Ожидается ошибка ErrOnlyOneLiteral для литерала в скобках, получена: %v", err)
		}
		if count != 0 {
			t.Errorf("Ожидается, что счетчик равен 0 для литерала в скобках, получено: %d", count)
		}
		if len(plan.OrderIds) != 0 || len(plan.Expressions) != 0 {
			t.Error("План задач должен оставаться пустым для литерала в скобках")
		}
	})
}

func TestProcessExpression(t *testing.T) {
	t.Run("CompositeExpression", func(t *testing.T) {
		// Сбрасываем dummy менеджер
		ManagerInstance = NewManager()

		id, err := ProcessExpression("1+2")
		if err != nil {
			t.Errorf("Ошибка для составного выражения: %v", err)
		}
		if id == 0 {
			t.Error("Ожидается ненулевой id для составного выражения")
		}
		if len(ManagerInstance.TasksQueue.storage) != 1 {
			t.Errorf("Ожидается, что AddTask вызвана 1 раз, получено: %d", len(ManagerInstance.TasksQueue.storage))
		}
		map_len := 0
		ManagerInstance.AllExpressions.Range(func(key any, value any) bool {
			map_len++
			return true
		})
		if map_len != 1 {
			t.Errorf("Ожидается, что StoreExpression вызвана 1 раз, получено: %d", map_len)
		}
	})

	t.Run("ComplexExpression", func(t *testing.T) {
		// Сбрасываем dummy менеджер
		ManagerInstance = NewManager()

		id, err := ProcessExpression("2 + (1 * 2 - 3 + (10 * 4 - 14))")
		if err != nil {
			t.Errorf("Ошибка для составного выражения: %v", err)
		}
		if id == 0 {
			t.Error("Ожидается ненулевой id для составного выражения")
		}
		if len(ManagerInstance.TasksQueue.storage) != 6 {
			t.Errorf("Ожидается, что AddTask вызвана 6 раз, получено: %d", len(ManagerInstance.TasksQueue.storage))
		}
		map_len := 0
		ManagerInstance.AllExpressions.Range(func(key any, value any) bool {
			map_len++
			return true
		})
		if map_len != 6 {
			t.Errorf("Ожидается, что StoreExpression вызвана 6 раз, получено: %d", map_len)
		}
	})

	t.Run("SingleLiteral", func(t *testing.T) {
		ManagerInstance = NewManager()

		id, err := ProcessExpression("42")
		if err != nil {
			t.Errorf("Ошибка для литерального выражения: %v", err)
		}
		if id == 0 {
			t.Error("Ожидается ненулевой id для литерального выражения")
		}
		// Для литерального выражения задачи не должны добавляться
		if len(ManagerInstance.TasksQueue.storage) != 0 {
			t.Errorf("Ожидается, что AddTask не вызывается для литерала, получено: %d", len(ManagerInstance.TasksQueue.storage))
		}
		map_len := 0
		ManagerInstance.AllExpressions.Range(func(key any, value any) bool {
			map_len++
			return true
		})
		if map_len != 1 {
			t.Errorf("Ожидается, что StoreExpression вызвана 1 раз для литерала, получено: %d", map_len)
		}
		// Проверяем, что выражение сохранено с результатом равным 42
		ManagerInstance.AllExpressions.Range(func(key any, value any) bool {
			exp := value.(Expression)
			if exp.result != 42 {
				t.Errorf("Ожидается, что результат выражения равен 42, получено: %v", exp.result)
			}
			return true
		})
	})

	t.Run("InvalidExpression", func(t *testing.T) {
		ManagerInstance = NewManager()

		// Передаем некорректное выражение (например, лишний оператор)
		_, err := ProcessExpression("1+")
		if err == nil {
			t.Error("Ожидается ошибка для некорректного выражения")
		}
	})

	t.Run("InvalidExpressionString", func(t *testing.T) {
		ManagerInstance = NewManager()

		// Передаем некорректное выражение (например, лишний оператор)
		_, err := ProcessExpression("abcd")
		if err == nil {
			t.Error("Ожидается ошибка для некорректного выражения")
		}
	})

	t.Run("ParenComposite", func(t *testing.T) {
		ManagerInstance = NewManager()

		id, err := ProcessExpression("(1+2)")
		if err != nil {
			t.Errorf("Ошибка для паренизированного составного выражения: %v", err)
		}
		if id == 0 {
			t.Error("Ожидается ненулевой id для паренизированного составного выражения")
		}
		if len(ManagerInstance.TasksQueue.storage) != 1 {
			t.Errorf("Ожидается, что AddTask вызвана 1 раз для паренизированного выражения, получено: %d", len(ManagerInstance.TasksQueue.storage))
		}
		map_len := 0
		ManagerInstance.AllExpressions.Range(func(key any, value any) bool {
			map_len++
			return true
		})
		if map_len != 1 {
			t.Errorf("Ожидается, что StoreExpression вызвана 1 раз для паренизированного выражения, получено: %d", map_len)
		}
	})
}

func TestGetFirstLiteralValue(t *testing.T) {
	cases := []struct {
		name     string
		node     ast.Node
		expected float64
		hasError bool
	}{
		{
			name:     "simple literal",
			node:     &ast.BasicLit{Kind: token.INT, Value: "42"},
			expected: 42,
			hasError: false,
		},
		{
			name:     "paren expression with literal",
			node:     &ast.ParenExpr{X: &ast.BasicLit{Kind: token.INT, Value: "10"}},
			expected: 10,
			hasError: false,
		},
		{
			name:     "double paren expression with literal",
			node:     &ast.ParenExpr{X: &ast.ParenExpr{X: &ast.BasicLit{Kind: token.INT, Value: "5"}}},
			expected: 5,
			hasError: false,
		},
		{
			name:     "invalid literal type",
			node:     &ast.BasicLit{Kind: token.STRING, Value: "abc"},
			expected: 0,
			hasError: true,
		},
		{
			name:     "non-literal node",
			node:     &ast.Ident{Name: "x"},
			expected: 0,
			hasError: true,
		},
		{
			name:     "nil node",
			node:     nil,
			expected: 0,
			hasError: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := getFirstLiteralValue(tc.node)
			if tc.hasError {
				if err == nil {
					t.Errorf("ожидалась ошибка, но её не было")
				}
			} else {
				if err != nil {
					t.Errorf("не ожидалась ошибка, но получили: %v", err)
				}
				if result != tc.expected {
					t.Errorf("ожидалось значение %v, получено %v", tc.expected, result)
				}
			}
		})
	}
}

func TestProcessExpressionResult(t *testing.T) {
	cases := []struct {
		name       string
		setup      func() TaskResult
		checkAfter func(t *testing.T, result error)
	}{
		{
			name: "простая задача с успешным результатом",
			setup: func() TaskResult {
				ManagerInstance = NewManager()

				// Создаем выражение
				expr := Expression{
					id:        42,
					parentId:  0,
					isRoot:    true,
					status:    "working",
					childSide: "nil",
				}

				ManagerInstance.StoreExpression(42, expr)

				return TaskResult{
					ID:     42,
					Result: 10,
					Error:  "nil",
				}
			},
			checkAfter: func(t *testing.T, err error) {
				if err != nil {
					t.Errorf("ожидался nil, получена ошибка: %v", err)
				}

				expr, err := ManagerInstance.GetExpressionById(42)
				if err != nil {
					t.Errorf("не удалось получить выражение: %v", err)
				}

				if expr.result != 10 {
					t.Errorf("ожидался результат 10, получен %v", expr.result)
				}

				if expr.status != "done" {
					t.Errorf("ожидался статус 'done', получен %v", expr.status)
				}
			},
		},
		{
			name: "обновление значения в родительском выражении",
			setup: func() TaskResult {
				ManagerInstance = NewManager()

				// Создаем родительское выражение
				parentExpr := Expression{
					id:         30,
					parentId:   0,
					isRoot:     true,
					status:     "waiting",
					childSide:  "nil",
					leftValue:  make(chan float64, 1),
					rightValue: make(chan float64, 1),
				}

				// Создаем дочернее выражение
				childExpr := Expression{
					id:        40,
					parentId:  30,
					isRoot:    false,
					status:    "working",
					childSide: "left",
				}

				ManagerInstance.StoreExpression(30, parentExpr)
				ManagerInstance.StoreExpression(40, childExpr)

				return TaskResult{
					ID:     40,
					Result: 15,
					Error:  "nil",
				}
			},
			checkAfter: func(t *testing.T, err error) {
				if err != nil {
					t.Errorf("ожидался nil, получена ошибка: %v", err)
				}

				// Проверяем, что значение было передано в родительское выражение
				parentExpr, _ := ManagerInstance.GetExpressionById(30)

				// Проверяем, что значение было помещено в левый канал родителя
				select {
				case val := <-parentExpr.leftValue:
					if val != 15 {
						t.Errorf("ожидалось значение 15 в левом канале родителя, получено %v", val)
					}
				default:
					t.Error("значение не было отправлено в левый канал родителя")
				}
			},
		},
		{
			name: "обработка несуществующего выражения",
			setup: func() TaskResult {
				ManagerInstance = NewManager()

				return TaskResult{
					ID:     9999,
					Result: 42,
					Error:  "nil",
				}
			},
			checkAfter: func(t *testing.T, err error) {
				if err == nil {
					t.Error("ожидалась ошибка для несуществующего выражения, получен nil")
				}
				if err != ErrExpressionNotFound {
					t.Errorf("ожидалась ошибка ErrExpressionNotFound, получена %v", err)
				}
			},
		},
		{
			name: "невалидный идентификатор родителя",
			setup: func() TaskResult {
				ManagerInstance = NewManager()

				// Создаем выражение с невалидным идентификатором родителя
				expr := Expression{
					id:        50,
					parentId:  0,
					isRoot:    false, // не корневой, но parentId == 0
					status:    "working",
					childSide: "left",
				}

				ManagerInstance.StoreExpression(50, expr)

				return TaskResult{
					ID:     50,
					Result: 100,
					Error:  "nil",
				}
			},
			checkAfter: func(t *testing.T, err error) {
				if err == nil {
					t.Error("ожидалась ошибка для невалидного parentId, получен nil")
				}
				if err != ErrInvalidParentId {
					t.Errorf("ожидалась ошибка ErrInvalidParentId, получена %v", err)
				}
			},
		},
		{
			name: "обновление значения в правом канале родителя",
			setup: func() TaskResult {
				ManagerInstance = NewManager()

				// Создаем родительское выражение
				parentExpr := Expression{
					id:         60,
					parentId:   0,
					isRoot:     true,
					status:     "waiting",
					childSide:  "nil",
					leftValue:  make(chan float64, 1),
					rightValue: make(chan float64, 1),
				}

				// Создаем дочернее выражение
				childExpr := Expression{
					id:        70,
					parentId:  60,
					isRoot:    false,
					status:    "working",
					childSide: "right",
				}

				ManagerInstance.StoreExpression(60, parentExpr)
				ManagerInstance.StoreExpression(70, childExpr)

				return TaskResult{
					ID:     70,
					Result: 25,
					Error:  "nil",
				}
			},
			checkAfter: func(t *testing.T, err error) {
				if err != nil {
					t.Errorf("ожидался nil, получена ошибка: %v", err)
				}

				// Проверяем, что значение было передано в родительское выражение
				parentExpr, _ := ManagerInstance.GetExpressionById(60)

				// Проверяем, что значение было помещено в правый канал родителя
				select {
				case val := <-parentExpr.rightValue:
					if val != 25 {
						t.Errorf("ожидалось значение 25 в правом канале родителя, получено %v", val)
					}
				default:
					t.Error("значение не было отправлено в правый канал родителя")
				}
			},
		},
		{
			name: "Error in result with root",
			setup: func() TaskResult {
				taskResult := TaskResult{
					ID:     1,
					Result: 0,
					Error:  "division by zero",
				}
				ManagerInstance = NewManager()
				ManagerInstance.StoreExpression(1, Expression{
					id:        1,
					parentId:  0,
					rootId:    1,
					isRoot:    true,
					status:    "working",
					childSide: "nil",
				})
				return taskResult
			},
			checkAfter: func(t *testing.T, err error) {
				if err != nil {
					t.Errorf("ожидался nil, получена ошибка: %v", err)
				}
				rootExpr, _ := ManagerInstance.GetExpressionById(1)
				if rootExpr.status != "division by zero" {
					t.Errorf("ожидался статус division by zero, получен %v", rootExpr.status)
				}
			},
		},
		{
			name: "Error in result without root",
			setup: func() TaskResult {
				taskResult := TaskResult{
					ID:     1,
					Result: 0,
					Error:  "division by zero",
				}
				ManagerInstance = NewManager()
				ManagerInstance.StoreExpression(1, Expression{
					id:        1,
					parentId:  2,
					rootId:    3,
					isRoot:    false,
					status:    "working",
					childSide: "nil",
				})
				return taskResult
			},
			checkAfter: func(t *testing.T, err error) {
				if err == nil {
					t.Error("ожидалась ошибка, получен nil")
				}
			},
		},
		{
			name: "Error in result without node",
			setup: func() TaskResult {
				taskResult := TaskResult{
					ID:     1,
					Result: 0,
					Error:  "division by zero",
				}
				ManagerInstance = NewManager()
				return taskResult
			},
			checkAfter: func(t *testing.T, err error) {
				if err == nil {
					t.Error("ожидалась ошибка, получен nil")
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.setup()
			err := ProcessExpressionResult(result)
			tc.checkAfter(t, err)
		})
	}
}
