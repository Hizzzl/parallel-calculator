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
	if _, err := saveASTToDB(tx, expressionID, node, nil, true); err != nil {
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

		// Проверяем тип левого операнда - должен быть числом или бинарным выражением
		switch n.X.(type) {
		case *ast.BasicLit, *ast.BinaryExpr, *ast.ParenExpr:
			// Поддерживаемые типы операндов
		default:
			return fmt.Errorf("unsupported left operand type: %T", n.X)
		}

		// Проверяем тип правого операнда - должен быть числом или бинарным выражением
		switch n.Y.(type) {
		case *ast.BasicLit, *ast.BinaryExpr, *ast.ParenExpr:
			// Поддерживаемые типы операндов
		default:
			return fmt.Errorf("unsupported right operand type: %T", n.Y)
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

	case *ast.Ident:
		// Запрещаем использование идентификаторов (переменных)
		return fmt.Errorf("variables not allowed: '%s'", n.Name)

	case *ast.CallExpr:
		// Запрещаем вызовы функций
		return fmt.Errorf("function calls not allowed")

	default:
		logger.LogINFO(fmt.Sprintf("Неизвестный тип узла AST: %T", n))
		return fmt.Errorf("unsupported expression element: %T", n)
	}
}

// saveASTToDB сохраняет AST в базу данных, возвращает ID корневой операции
// Вспомогательная функция для второго прохода
func saveASTToDB(tx *sql.Tx, expressionID int64, node ast.Node, parentOpID *int64, isLeft bool) (int64, error) {
	// Если это числовой литерал и у него есть родитель, обновляем значение родительской операции
	if parentOpID != nil {
		// Проверяем, является ли узел числом (литералом)
		if value, isLiteral := IsLiteralOnly(node); isLiteral {
			// Получаем позицию узла относительно родителя
			if isLeft {
				// Обновляем левый аргумент родительской операции
				_, err := tx.Exec(
					`UPDATE operations SET left_value = ? WHERE id = ?`,
					value, *parentOpID,
				)
				if err != nil {
					return 0, err
				}
			} else {
				// Обновляем правый аргумент родительской операции
				_, err := tx.Exec(
					`UPDATE operations SET right_value = ? WHERE id = ?`,
					value, *parentOpID,
				)
				if err != nil {
					return 0, err
				}
			}

			// Проверяем, можно ли установить статус Ready для родительской операции
			var leftVal, rightVal sql.NullFloat64
			err := tx.QueryRow(
				`SELECT left_value, right_value FROM operations WHERE id = ?`,
				*parentOpID,
			).Scan(&leftVal, &rightVal)

			if err == nil && leftVal.Valid && rightVal.Valid {
				// У операции есть оба аргумента, переводим в статус Ready
				_, err = tx.Exec(
					`UPDATE operations SET status = ? WHERE id = ?`,
					db.StatusReady, *parentOpID,
				)
				if err != nil {
					return 0, err
				}
			}

			// Для литерала возвращаем 0, так как отдельная запись в БД не создавалась
			return 0, nil
		}
	}

	// Формируем позицию узла относительно родителя
	var childPosition *string
	if parentOpID != nil {
		pos := "right"
		if isLeft {
			pos = "left"
		}
		childPosition = &pos
	}

	switch n := node.(type) {
	case *ast.BinaryExpr:
		// Проверяем, являются ли оба операнда литералами
		leftVal, leftIsLiteral := IsLiteralOnly(n.X)
		rightVal, rightIsLiteral := IsLiteralOnly(n.Y)

		// Определяем начальные значения для левой и правой частей
		var leftArg, rightArg *float64 = nil, nil

		// Если операнды - литералы, сразу сохраняем их значения
		if leftIsLiteral {
			leftArg = &leftVal
		}
		if rightIsLiteral {
			rightArg = &rightVal
		}

		// Создаем операцию для бинарного выражения
		// Статус будет установлен автоматически в функции CreateOperation
		op, err := db.CreateOperation(
			expressionID,
			parentOpID,
			n.Op.String(),
			leftArg, rightArg, // Значения левой и правой частей (если это литералы)
			parentOpID == nil, // Корневая операция, если нет родителя
			childPosition,     // Позиция относительно родителя
		)
		if err != nil {
			return 0, err
		} else {
			// Только если операнд не литерал, рекурсивно обрабатываем его
			if !leftIsLiteral {
				// Рекурсивно обрабатываем левую часть
				_, err = saveASTToDB(tx, expressionID, n.X, &op.ID, true)
				if err != nil {
					return 0, err
				}
			}

			if !rightIsLiteral {
				// Рекурсивно обрабатываем правую часть
				_, err = saveASTToDB(tx, expressionID, n.Y, &op.ID, false)
				if err != nil {
					return 0, err
				}
			}
		}

		return op.ID, nil

	case *ast.ParenExpr:
		// Для выражения в скобках просто обрабатываем содержимое
		return saveASTToDB(tx, expressionID, n.X, parentOpID, isLeft)

	case *ast.BasicLit:
		// Если это корневой литерал без родителя (например, выражение из одного числа)
		value, _ := strconv.ParseFloat(n.Value, 64) // Ошибка уже проверена в validateAST

		// Создаем операцию для корневого числа (случай одиночного числа как выражения)
		op, err := db.CreateOperation(
			expressionID,
			nil,         // Без родителя
			"+",         // Просто использовать унарный плюс
			&value, nil, // Заполняем значение левой части
			true, // Корневая операция
			nil,  // Без позиции
		)

		// Сразу устанавливаем статус Ready для корневого литерала
		if err == nil {
			_, err = tx.Exec(
				`UPDATE operations SET status = ? WHERE id = ?`,
				db.StatusReady, op.ID,
			)
		}

		if err != nil {
			return 0, err
		}

		return op.ID, nil

	default:
		return 0, ErrInvalidExpression
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
