package grpc

import (
	"context"
	"errors"
	"parallel-calculator/internal/logger"
	"parallel-calculator/proto"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// GRPCTaskClient представляет gRPC клиент для взаимодействия с оркестратором
type GRPCTaskClient struct {
	conn   *grpc.ClientConn
	client proto.TaskServiceClient
}

// NewGRPCTaskClient создает новый gRPC клиент для взаимодействия с оркестратором
func NewGRPCTaskClient(address string) (*GRPCTaskClient, error) {
	// Устанавливаем соединение с сервером, используя незащищенный канал (для простоты)
	// В продакшене рекомендуется использовать TLS
	conn, err := grpc.Dial(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}

	// Создаем клиент
	client := proto.NewTaskServiceClient(conn)

	return &GRPCTaskClient{
		conn:   conn,
		client: client,
	}, nil
}

// Close закрывает соединение с сервером
func (c *GRPCTaskClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// GetTask запрашивает задачу от оркестратора
// Возвращает структуру Task для агента
func (c *GRPCTaskClient) GetTask() (*Task, error) {
	logger.INFO.Println("Отправка запроса на получение задачи через gRPC")

	// Создаем контекст с таймаутом
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Отправляем запрос
	resp, err := c.client.GetTask(ctx, &proto.GetTaskRequest{})
	if err != nil {
		logger.ERROR.Println("Ошибка при получении задачи: ", err)
		return nil, err
	}

	// Если задач нет, возвращаем nil
	if !resp.HasTask {
		logger.INFO.Println("Нет доступных задач")
		return nil, nil
	}

	// Преобразуем ответ в структуру Task
	task := &Task{
		ID:            resp.Id,
		LeftValue:     resp.LeftValue,
		RightValue:    resp.RightValue,
		Operator:      resp.Operator,
		OperationTime: time.Duration(resp.OperationTimeNs),
	}

	logger.INFO.Println("Получена задача: ", task)
	return task, nil
}

// SendTaskResult отправляет результат выполнения задачи оркестратору
func (c *GRPCTaskClient) SendTaskResult(taskResult TaskResult) error {
	logger.INFO.Println("Отправка результата задачи через gRPC: ", taskResult)

	// Создаем контекст с таймаутом
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Отправляем запрос
	resp, err := c.client.SendTaskResult(ctx, &proto.TaskResultRequest{
		Id:     taskResult.ID,
		Result: taskResult.Result,
		Error:  taskResult.Error,
	})
	if err != nil {
		logger.ERROR.Println("Ошибка при отправке результата задачи: ", err)
		return err
	}

	// Проверяем успешность операции
	if !resp.Success {
		logger.ERROR.Println("Сервер сообщил об ошибке: ", resp.Error)
		return errors.New(resp.Error)
	}

	logger.INFO.Println("Результат задачи успешно отправлен")
	return nil
}
