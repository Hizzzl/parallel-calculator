package agent_test

import (
	"parallel-calculator/internal/agent"
	"parallel-calculator/internal/config"
	"parallel-calculator/internal/logger"
	"testing"
	"time"
)

// mockTaskClient реализует интерфейс TaskClient для тестирования
type mockTaskClient struct {
	getTaskFunc          func() (*agent.Task, error)
	sendTaskResultFunc   func(result agent.TaskResult) error
	closeFunc            func() error
	getTaskCalled        int
	sendTaskResultCalled int
	closeCalled          int
	lastTaskResult       agent.TaskResult
}

func (m *mockTaskClient) GetTask() (*agent.Task, error) {
	m.getTaskCalled++
	return m.getTaskFunc()
}

func (m *mockTaskClient) SendTaskResult(result agent.TaskResult) error {
	m.sendTaskResultCalled++
	m.lastTaskResult = result
	return m.sendTaskResultFunc(result)
}

func (m *mockTaskClient) Close() error {
	m.closeCalled++
	return m.closeFunc()
}

// Инициализация конфигурации для тестов
func setupAgentTest() {
	// Инициализируем конфигурацию
	config.InitConfig("../../.env")

	// Инициализируем логгер
	logger.InitAgentLogger()
	logger.InitClientLogger()
}

func TestNewGRPCClient(t *testing.T) {
	// Инициализируем конфигурацию
	setupAgentTest()

	// Проверяем, что клиент создается успешно
	client, err := agent.NewGRPCClient("localhost:50051")
	if err != nil {
		t.Errorf("NewGRPCClient() error = %v", err)
		return
	}

	if client == nil {
		t.Errorf("NewGRPCClient() returned nil client")
	}
}

func TestSetGlobalClient(t *testing.T) {
	// Инициализируем конфигурацию
	setupAgentTest()
	// Создаем мок клиента
	mockClient := &mockTaskClient{
		getTaskFunc: func() (*agent.Task, error) {
			return nil, nil
		},
		sendTaskResultFunc: func(result agent.TaskResult) error {
			return nil
		},
		closeFunc: func() error {
			return nil
		},
	}

	// Устанавливаем глобального клиента
	agent.SetGlobalClient(mockClient)

	// Проверяем, что глобальный клиент установлен (косвенно, через последующие тесты)
	// Запуск воркера с задачей должен привести к вызову SendTaskResult на глобальном клиенте
	tasksChan := make(chan agent.Task, 1)

	// Запускаем воркер в горутине
	go agent.Worker(tasksChan, 1)

	// Создаем задачу для отправки воркеру
	task := agent.Task{
		ID:            1,
		LeftValue:     2.0,
		RightValue:    3.0,
		Operator:      "+",
		OperationTime: 10 * time.Millisecond,
	}

	// Отправляем задачу
	tasksChan <- task

	// Даем время воркеру обработать задачу
	time.Sleep(50 * time.Millisecond)

	// Проверяем, что метод SendTaskResult был вызван
	if mockClient.sendTaskResultCalled != 1 {
		t.Errorf("Worker did not call SendTaskResult on global client")
	}

	// Проверяем, что результат правильный
	expectedResult := 5.0 // 2.0 + 3.0 = 5.0
	if mockClient.lastTaskResult.Result != expectedResult {
		t.Errorf("Task result = %v, want %v", mockClient.lastTaskResult.Result, expectedResult)
	}
}

func TestWorkerWithDifferentOperators(t *testing.T) {
	// Инициализируем конфигурацию
	setupAgentTest()
	// Создаем мок клиента
	mockClient := &mockTaskClient{
		getTaskFunc: func() (*agent.Task, error) {
			return nil, nil
		},
		sendTaskResultFunc: func(result agent.TaskResult) error {
			return nil
		},
		closeFunc: func() error {
			return nil
		},
	}

	// Устанавливаем глобального клиента
	agent.SetGlobalClient(mockClient)

	tests := []struct {
		name           string
		task           agent.Task
		expectedResult float64
		expectedError  string
	}{
		{
			name: "Addition",
			task: agent.Task{
				ID:            1,
				LeftValue:     2.0,
				RightValue:    3.0,
				Operator:      "+",
				OperationTime: 10 * time.Millisecond,
			},
			expectedResult: 5.0,
			expectedError:  "nil",
		},
		{
			name: "Subtraction",
			task: agent.Task{
				ID:            2,
				LeftValue:     5.0,
				RightValue:    3.0,
				Operator:      "-",
				OperationTime: 10 * time.Millisecond,
			},
			expectedResult: 2.0,
			expectedError:  "nil",
		},
		{
			name: "Multiplication",
			task: agent.Task{
				ID:            3,
				LeftValue:     4.0,
				RightValue:    3.0,
				Operator:      "*",
				OperationTime: 10 * time.Millisecond,
			},
			expectedResult: 12.0,
			expectedError:  "nil",
		},
		{
			name: "Division",
			task: agent.Task{
				ID:            4,
				LeftValue:     6.0,
				RightValue:    2.0,
				Operator:      "/",
				OperationTime: 10 * time.Millisecond,
			},
			expectedResult: 3.0,
			expectedError:  "nil",
		},
		{
			name: "Division by zero",
			task: agent.Task{
				ID:            5,
				LeftValue:     6.0,
				RightValue:    0.0,
				Operator:      "/",
				OperationTime: 10 * time.Millisecond,
			},
			expectedResult: 0.0,
			expectedError:  "division by zero",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Сбрасываем счетчики вызовов
			mockClient.sendTaskResultCalled = 0

			// Создаем канал задач
			tasksChan := make(chan agent.Task, 1)

			// Запускаем воркер в горутине
			go agent.Worker(tasksChan, 1)

			// Отправляем задачу
			tasksChan <- tt.task

			// Даем время воркеру обработать задачу
			time.Sleep(50 * time.Millisecond)

			// Проверяем, что метод SendTaskResult был вызван
			if mockClient.sendTaskResultCalled != 1 {
				t.Errorf("Worker did not call SendTaskResult")
				return
			}

			// Проверяем, что результат правильный
			if mockClient.lastTaskResult.Result != tt.expectedResult {
				t.Errorf("Task result = %v, want %v", mockClient.lastTaskResult.Result, tt.expectedResult)
			}

			// Проверяем, что ошибка правильная
			if mockClient.lastTaskResult.Error != tt.expectedError {
				t.Errorf("Task error = %v, want %v", mockClient.lastTaskResult.Error, tt.expectedError)
			}
		})
	}
}

func TestGRPCClientAdapter(t *testing.T) {
	// Инициализируем конфигурацию
	setupAgentTest()
	// Прямое тестирование адаптера через его методы не очень эффективно,
	// так как он в основном просто делегирует вызовы реальному gRPC клиенту.
	// Однако мы можем проверить основные сценарии с моками.

	t.Run("GetTask error handling", func(t *testing.T) {
		// Этот тест проверяет обработку ошибок в GetTask
		// Однако мы не можем напрямую создать grpcClientAdapter из-за его приватности.
		// Тестирование через интерфейс TaskClient будет более подходящим.
	})
}
