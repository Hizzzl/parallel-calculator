package grpc

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"path/filepath"
	"testing"
	"time"

	"parallel-calculator/internal/config"
	"parallel-calculator/internal/db"
	"parallel-calculator/internal/logger"

	_ "github.com/mattn/go-sqlite3"
)

func InitTest(t *testing.T) {
	config.InitConfig("../../.env")
	db.DB, _ = sql.Open("sqlite3", "file:memdb1?mode=memory&cache=shared")

	db.ApplySchema(filepath.Join("../../internal/db", "schema.sql"))

	// Инициализация логгеров с заглушкой вместо вывода
	// Используем ioutil.Discard, который просто отбрасывает все записи
	logger.INFO = log.New(ioutil.Discard, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	logger.ERROR = log.New(ioutil.Discard, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
}

// setupTestEnvironment создает тестовое окружение и запускает сервер
func setupTestEnvironment(t *testing.T) (string, func(), int64) {
	// Инициализируем тестовое окружение
	InitTest(t)

	// Очищаем базу данных перед тестом
	_, err := db.DB.Exec("DELETE FROM operations")
	if err != nil {
		t.Fatalf("Failed to clear operations: %v", err)
	}
	_, err = db.DB.Exec("DELETE FROM expressions")
	if err != nil {
		t.Fatalf("Failed to clear expressions: %v", err)
	}
	_, err = db.DB.Exec("DELETE FROM users")
	if err != nil {
		t.Fatalf("Failed to clear users: %v", err)
	}

	// Сначала создаем пользователя с уникальным ID
	// Генерируем уникальный ID на основе текущего времени
	taskID := time.Now().UnixNano() % 1000000

	result, err := db.DB.Exec(`
		INSERT INTO users (login, password_hash)
		VALUES (?, ?);
	`, fmt.Sprintf("testuser_%d", taskID), "hashpwd")
	if err != nil {
		t.Fatalf("Failed to insert test user: %v", err)
	}

	userID, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("Failed to get last inserted user ID: %v", err)
	}

	// Затем создаем выражение
	result, err = db.DB.Exec(`
		INSERT INTO expressions (user_id, original_expression, status)
		VALUES (?, '2+3', 'pending');
	`, userID)
	if err != nil {
		t.Fatalf("Failed to insert test expression: %v", err)
	}

	expressionID, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("Failed to get last inserted expression ID: %v", err)
	}

	// Добавляем тестовую операцию (задачу)
	result, err = db.DB.Exec(`
		INSERT INTO operations (expression_id, left_value, right_value, operator, status)
		VALUES (?, 2.0, 3.0, '+', 'ready');
	`, expressionID)
	if err != nil {
		t.Fatalf("Failed to insert test operation: %v", err)
	}

	operationID, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("Failed to get last inserted operation ID: %v", err)
	}

	// Запускаем gRPC сервер на случайном порту
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("Failed to listen: %v", err)
	}

	port := listener.Addr().(*net.TCPAddr).Port
	serverAddr := fmt.Sprintf("localhost:%d", port)

	// Запускаем сервер в отдельной горутине
	server, err := StartGRPCServer(fmt.Sprintf(":%d", port))
	if err != nil {
		listener.Close()
		t.Fatalf("Failed to start gRPC server: %v", err)
	}

	// Функция очистки ресурсов
	cleanup := func() {
		server.Stop()
		listener.Close()
	}

	return serverAddr, cleanup, operationID
}

func TestGetTask(t *testing.T) {
	// Инициализируем тестовое окружение
	InitTest(t)

	// Очищаем базу данных перед тестом
	_, err := db.DB.Exec("DELETE FROM operations")
	if err != nil {
		t.Fatalf("Failed to clear operations: %v", err)
	}
	_, err = db.DB.Exec("DELETE FROM expressions")
	if err != nil {
		t.Fatalf("Failed to clear expressions: %v", err)
	}
	_, err = db.DB.Exec("DELETE FROM users")
	if err != nil {
		t.Fatalf("Failed to clear users: %v", err)
	}

	// Сначала создаем пользователя с уникальным ID
	// Генерируем уникальный ID на основе текущего времени
	taskID := time.Now().UnixNano() % 1000000

	result, err := db.DB.Exec(`
		INSERT INTO users (login, password_hash)
		VALUES (?, ?);
	`, fmt.Sprintf("testuser_%d", taskID), "hashpwd")
	if err != nil {
		t.Fatalf("Failed to insert test user: %v", err)
	}

	userID, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("Failed to get last inserted user ID: %v", err)
	}

	// Затем создаем выражение
	result, err = db.DB.Exec(`
		INSERT INTO expressions (user_id, original_expression, status)
		VALUES (?, '2+3', 'pending');
	`, userID)
	if err != nil {
		t.Fatalf("Failed to insert test expression: %v", err)
	}

	expressionID, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("Failed to get last inserted expression ID: %v", err)
	}

	// Добавляем тестовую операцию (задачу)
	result, err = db.DB.Exec(`
		INSERT INTO operations (expression_id, left_value, right_value, operator, status)
		VALUES (?, 2.0, 3.0, '+', 'ready');
	`, expressionID)
	if err != nil {
		t.Fatalf("Failed to insert test operation: %v", err)
	}

	operationID, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("Failed to get last inserted operation ID: %v", err)
	}

	// Запускаем gRPC сервер на случайном порту (с +2 к порту)
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("Failed to listen: %v", err)
	}

	port := listener.Addr().(*net.TCPAddr).Port + 2 // Добавляем 2 к порту для избежания конфликта
	serverAddr := fmt.Sprintf("localhost:%d", port)

	// Запускаем сервер в отдельной горутине
	server, err := StartGRPCServer(fmt.Sprintf(":%d", port))
	if err != nil {
		listener.Close()
		t.Fatalf("Failed to start gRPC server: %v", err)
	}

	// Функция очистки ресурсов
	cleanup := func() {
		server.Stop()
		listener.Close()
	}
	defer cleanup()

	testTaskID := operationID

	// Даем серверу время на запуск
	time.Sleep(100 * time.Millisecond)

	// Создаем клиента, который подключается к тестовому серверу
	client, err := NewGRPCTaskClient(serverAddr)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// Получаем задачу
	task, err := client.GetTask()

	// Проверяем результаты
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if task == nil {
		t.Fatal("Expected task, got nil")
	}

	if task.ID != uint32(testTaskID) {
		t.Errorf("Expected ID %d, got %d", testTaskID, task.ID)
	}

	// Проверяем остальные поля
	if task.LeftValue != 2.0 {
		t.Errorf("Expected LeftValue 2.0, got %f", task.LeftValue)
	}

	if task.RightValue != 3.0 {
		t.Errorf("Expected RightValue 3.0, got %f", task.RightValue)
	}

	if task.Operator != "+" {
		t.Errorf("Expected Operator +, got %s", task.Operator)
	}
}

func TestSendTaskResult(t *testing.T) {
	// Настраиваем тестовое окружение с другим портом
	// Инициализируем тестовое окружение
	InitTest(t)

	// Очищаем базу данных перед тестом
	_, err := db.DB.Exec("DELETE FROM operations")
	if err != nil {
		t.Fatalf("Failed to clear operations: %v", err)
	}
	_, err = db.DB.Exec("DELETE FROM expressions")
	if err != nil {
		t.Fatalf("Failed to clear expressions: %v", err)
	}
	_, err = db.DB.Exec("DELETE FROM users")
	if err != nil {
		t.Fatalf("Failed to clear users: %v", err)
	}

	// Сначала создаем пользователя с уникальным ID
	// Генерируем уникальный ID на основе текущего времени
	taskID := time.Now().UnixNano() % 1000000

	result, err := db.DB.Exec(`
		INSERT INTO users (login, password_hash)
		VALUES (?, ?);
	`, fmt.Sprintf("testuser_%d", taskID), "hashpwd")
	if err != nil {
		t.Fatalf("Failed to insert test user: %v", err)
	}

	userID, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("Failed to get last inserted user ID: %v", err)
	}

	// Затем создаем выражение
	result, err = db.DB.Exec(`
		INSERT INTO expressions (user_id, original_expression, status)
		VALUES (?, '2+3', 'pending');
	`, userID)
	if err != nil {
		t.Fatalf("Failed to insert test expression: %v", err)
	}

	expressionID, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("Failed to get last inserted expression ID: %v", err)
	}

	// Добавляем тестовую операцию (задачу)
	result, err = db.DB.Exec(`
		INSERT INTO operations (expression_id, left_value, right_value, operator, status)
		VALUES (?, 2.0, 3.0, '+', 'ready');
	`, expressionID)
	if err != nil {
		t.Fatalf("Failed to insert test operation: %v", err)
	}

	operationID, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("Failed to get last inserted operation ID: %v", err)
	}

	// Запускаем gRPC сервер на случайном порту (с +1 к порту)
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("Failed to listen: %v", err)
	}

	port := listener.Addr().(*net.TCPAddr).Port + 1 // Добавляем 1 к порту для избежания конфликта
	serverAddr := fmt.Sprintf("localhost:%d", port)

	// Запускаем сервер в отдельной горутине
	server, err := StartGRPCServer(fmt.Sprintf(":%d", port))
	if err != nil {
		listener.Close()
		t.Fatalf("Failed to start gRPC server: %v", err)
	}

	// Функция очистки ресурсов
	cleanup := func() {
		server.Stop()
		listener.Close()
	}
	defer cleanup()

	testTaskID := operationID

	// Даем серверу время на запуск
	time.Sleep(100 * time.Millisecond)

	// Создаем клиента, который подключается к тестовому серверу
	client, err := NewGRPCTaskClient(serverAddr)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// Создаем результат задачи
	taskResult := TaskResult{
		ID:     uint32(testTaskID),
		Result: 5.0,
	}

	// Отправляем результат
	err = client.SendTaskResult(taskResult)
	if err != nil {
		t.Fatalf("Failed to send task result: %v", err)
	}

	// Проверяем, что задача была обновлена в базе данных
	var status string
	var resultValue sql.NullFloat64
	row := db.DB.QueryRow("SELECT status, result FROM operations WHERE id = ?", testTaskID)
	err = row.Scan(&status, &resultValue)
	if err != nil {
		t.Fatalf("Failed to query operation status: %v", err)
	}

	if status != "completed" {
		t.Errorf("Expected status 'completed', got %s", status)
	}

	if !resultValue.Valid || resultValue.Float64 != 5.0 {
		t.Errorf("Expected result 5.0, got %v", resultValue)
	}
}
