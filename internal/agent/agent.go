package agent

import (
	"fmt"
	"parallel-calculator/internal/config"
	"parallel-calculator/internal/grpc"
	"parallel-calculator/internal/logger"
	"strconv"
	"time"
)

// TaskClient представляет интерфейс для клиента, который взаимодействует с оркестратором
type TaskClient interface {
	// GetTask получает задачу от оркестратора
	GetTask() (*Task, error)
	// SendTaskResult отправляет результат задачи оркестратору
	SendTaskResult(TaskResult) error
	// Close закрывает соединение с оркестратором
	Close() error
}

// grpcClientAdapter адаптирует GRPCTaskClient к интерфейсу TaskClient
type grpcClientAdapter struct {
	address string
	client  *grpc.GRPCTaskClient
}

// NewGRPCClient создает новый gRPC клиент для оркестратора
func NewGRPCClient(address string) (TaskClient, error) {
	return &grpcClientAdapter{address: address}, nil
}

// Инициализация клиента при первом использовании
func (g *grpcClientAdapter) ensureClient() error {
	if g.client == nil {
		var err error
		g.client, err = grpc.NewGRPCTaskClient(g.address)
		if err != nil {
			return err
		}
	}
	return nil
}

// GetTask получает задачу через gRPC
func (g *grpcClientAdapter) GetTask() (*Task, error) {
	if err := g.ensureClient(); err != nil {
		return nil, err
	}

	// Получаем задачу из gRPC
	grpcTask, err := g.client.GetTask()
	if err != nil {
		return nil, err
	}

	// Если нет задачи, возвращаем nil
	if grpcTask == nil {
		return nil, nil
	}

	// Преобразуем в локальный тип Task
	return &Task{
		ID:            grpcTask.ID,
		LeftValue:     grpcTask.LeftValue,
		RightValue:    grpcTask.RightValue,
		Operator:      grpcTask.Operator,
		OperationTime: grpcTask.OperationTime,
	}, nil
}

// SendTaskResult отправляет результат задачи через gRPC
func (g *grpcClientAdapter) SendTaskResult(result TaskResult) error {
	if err := g.ensureClient(); err != nil {
		return err
	}

	// Преобразуем в тип для gRPC
	grpcResult := grpc.TaskResult{
		ID:     result.ID,
		Result: result.Result,
		Error:  result.Error,
	}

	return g.client.SendTaskResult(grpcResult)
}

// Close закрывает соединение
func (g *grpcClientAdapter) Close() error {
	if g.client != nil {
		return g.client.Close()
	}
	return nil
}

// Оставляем только gRPC реализацию

// StartAgent инициализирует и запускает агента с заданным количеством воркеров,
// которые получают задачи от оркестратора и выполняют их параллельно
func StartAgent() {
	cp := config.AppConfig.ComputingPower
	// Исправлено дублирование условия
	if cp == 0 {
		logger.ERROR.Println("COMPUTING_POWER is 0. AUTO SET TO 1")
		cp = 1
	}

	logger.INFO.Println("COMPUTING_POWER set to", cp)
	logger.INFO.Println("Starting workers...")

	tasks_chan := make(chan Task, cp)
	for i := 0; i < cp; i++ {
		go Worker(tasks_chan, i+1)
	}

	logger.INFO.Println("Starting Agent with gRPC communication")

	var client TaskClient
	var err error

	// Создаем адрес для gRPC сервера (порт HTTP + 1)
	httpPort, _ := strconv.Atoi(config.AppConfig.OrchestratorPort)
	grpcPort := httpPort + 1
	grpcAddress := fmt.Sprintf("%s:%d", config.AppConfig.OrchestratorHost, grpcPort)

	logger.INFO.Printf("Connecting to gRPC server at %s", grpcAddress)
	client, err = NewGRPCClient(grpcAddress)
	if err != nil {
		logger.ERROR.Fatalf("Failed to create gRPC client: %v - terminating agent", err)
	}
	defer client.Close()

	SetGlobalClient(client)

	for {
		time.Sleep(config.AppConfig.AgentRequestTimeout)

		task, err := client.GetTask()

		if err != nil {
			logger.ERROR.Println(err)
			continue
		}
		if task == nil {
			continue
		}
		tasks_chan <- *task
	}
}
