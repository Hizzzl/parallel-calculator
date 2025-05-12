package orchestrator_test

// import (
// 	"go/ast"
// 	"parallel-calculator/internal/orchestrator"
// 	"testing"
// )

// // TestCreateAST проверяет корректность создания AST из выражения
// func TestCreateAST(t *testing.T) {
// 	tests := []struct {
// 		name       string
// 		expression string
// 		wantErr    bool
// 	}{
// 		{
// 			name:       "Simple addition",
// 			expression: "2+2",
// 			wantErr:    false,
// 		},
// 		{
// 			name:       "Subtraction",
// 			expression: "5-3",
// 			wantErr:    false,
// 		},
// 		{
// 			name:       "Multiplication",
// 			expression: "4*2",
// 			wantErr:    false,
// 		},
// 		{
// 			name:       "Division",
// 			expression: "8/4",
// 			wantErr:    false,
// 		},
// 		{
// 			name:       "Complex expression",
// 			expression: "2*(3+4)",
// 			wantErr:    false,
// 		},
// 		{
// 			name:       "Expression with parentheses",
// 			expression: "(1+2)*(3+4)",
// 			wantErr:    false,
// 		},
// 		{
// 			name:       "Expression with negative number",
// 			expression: "-5+3",
// 			wantErr:    false,
// 		},
// 		{
// 			name:       "Invalid expression - incomplete",
// 			expression: "2+",
// 			wantErr:    true,
// 		},
// 		{
// 			name:       "Invalid expression - mismatched parentheses",
// 			expression: "2*(3+4",
// 			wantErr:    true,
// 		},
// 		{
// 			name:       "Invalid expression - empty",
// 			expression: "",
// 			wantErr:    true,
// 		},
// 		{
// 			name:       "Invalid expression - invalid characters",
// 			expression: "2+a",
// 			wantErr:    true,
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			got, err := orchestrator.CreateAST(tt.expression)

// 			// Проверяем наличие ошибки
// 			if (err != nil) != tt.wantErr {
// 				t.Errorf("CreateAST() error = %v, wantErr %v", err, tt.wantErr)
// 				return
// 			}

// 			// Если не ожидаем ошибки, проверяем результат
// 			if !tt.wantErr {
// 				if got == nil {
// 					t.Errorf("CreateAST() returned nil AST for valid expression")
// 				}

// 				// Проверяем, что AST является валидной структурой
// 				switch n := got.(type) {
// 				case *ast.BinaryExpr, *ast.ParenExpr, *ast.BasicLit, *ast.UnaryExpr:
// 					// Это валидные типы узлов AST для математических выражений
// 				default:
// 					t.Errorf("CreateAST() returned unexpected AST node type: %T", n)
// 				}
// 			}
// 		})
// 	}
// }

// // TestIsLiteral проверяет функцию определения литерала
// // func TestIsLiteral(t *testing.T) {
// // 	// Здесь мы создадим некоторые AST узлы вручную и проверим функцию isLiteral
// // 	tests := []struct {
// // 		name        string
// // 		expression  string      // Выражение для создания AST
// // 		isLiteral   bool        // Ожидаемый результат - является ли выражение литералом
// // 		literalValue *float64   // Ожидаемое значение литерала (если применимо)
// // 	}{
// // 		{
// // 			name:        "Simple number",
// // 			expression:  "2",
// // 			isLiteral:   true,
// // 			literalValue: floatPtr(2),
// // 		},
// // 		{
// // 			name:        "Expression is not a literal",
// // 			expression:  "2+2",
// // 			isLiteral:   false,
// // 			literalValue: nil,
// // 		},
// // 		{
// // 			name:        "Parenthesized number",
// // 			expression:  "(5)",
// // 			isLiteral:   true,
// // 			literalValue: floatPtr(5),
// // 		},
// // 		{
// // 			name:        "Nested parenthesized number",
// // 			expression:  "((7))",
// // 			isLiteral:   true,
// // 			literalValue: floatPtr(7),
// // 		},
// // 		{
// // 			name:        "Negative number",
// // 			expression:  "-3",
// // 			isLiteral:   false, // Унарные операции не считаются литералами в текущей реализации
// // 			literalValue: nil,
// // 		},
// // 	}

// // 	for _, tt := range tests {
// // 		t.Run(tt.name, func(t *testing.T) {
// // 			// Создаем AST из выражения
// // 			astNode, err := orchestrator.CreateAST(tt.expression)
// // 			if err != nil {
// // 				// Пропускаем тест, если не можем создать AST
// // 				t.Skipf("Failed to create AST for %s: %v", tt.expression, err)
// // 				return
// // 			}

// // 			// Пытаемся получить доступ к приватной функции isLiteral через рефлексию
// // 			// Примечание: для этого функция isLiteral должна быть экспортирована или
// // 			// нужно создать тестируемый wrapper в пакете orchestrator
// // 			// Предположим, что есть публичная функция IsLiteralForTesting
// // 			value, isLiteral := orchestrator.IsLiteralForTesting(astNode)

// // 			// Проверяем результат
// // 			if isLiteral != tt.isLiteral {
// // 				t.Errorf("isLiteral() = %v, want %v", isLiteral, tt.isLiteral)
// // 			}

// // 			// Если ожидаем, что это литерал, проверяем значение
// // 			if tt.isLiteral && isLiteral {
// // 				if (value == nil) != (tt.literalValue == nil) {
// // 					t.Errorf("isLiteral() value is %v, want %v", value, tt.literalValue)
// // 				} else if value != nil && tt.literalValue != nil && *value != *tt.literalValue {
// // 					t.Errorf("isLiteral() value = %f, want %f", *value, *tt.literalValue)
// // 				}
// // 			}
// // 		})
// // 	}
// // }

// // Helper функция для создания указателя на float64
// func floatPtr(v float64) *float64 {
// 	return &v
// }

// // Примечание: для корректной работы этого теста,
// // необходимо экспортировать функцию isLiteral в пакете orchestrator,
// // создав публичную версию для тестирования:

// /*
// // IsLiteralForTesting публичная версия приватной функции isLiteral для тестирования
// func IsLiteralForTesting(node ast.Node) (*float64, bool) {
//     return isLiteral(node)
// }
// */
