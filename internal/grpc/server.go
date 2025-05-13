package grpc

import (
	"context"
	"net"
	"parallel-calculator/internal/config"
	"parallel-calculator/internal/db"
	"parallel-calculator/internal/logger"
	"parallel-calculator/internal/orchestrator"
	"parallel-calculator/proto"
	"time"

	"google.golang.org/grpc"
)

// OrchestratorService имплементирует TaskServiceServer из сгенерированного кода
type OrchestratorService struct {
	proto.UnimplementedTaskServiceServer
}

// GetTask возвращает задачу для обработки агентом
func (s *OrchestratorService) GetTask(ctx context.Context, req *proto.GetTaskRequest) (*proto.GetTaskResponse, error) {
	logger.INFO.Println("gRPC: Запрос на получение задачи от агента")
	// Получаем одну готовую к обработке операцию из БД
	readyOp, err := db.GetReadyOperation()
	if err != nil {
		logger.LogERROR("Ошибка получения операции: " + err.Error())
		return &proto.GetTaskResponse{
			HasTask: false,
		}, nil
	}

	// Проверяем, есть ли операция
	if readyOp == nil {
		return &proto.GetTaskResponse{
			HasTask: false,
		}, nil
	}

	// Обновляем статус операции на "обрабатывается"
	err = db.UpdateOperationStatus(readyOp.ID, db.StatusProcessing)
	if err != nil {
		logger.LogERROR("Ошибка обновления статуса операции: " + err.Error())
		return &proto.GetTaskResponse{
			HasTask: false,
		}, nil
	}

	var leftVal, rightVal float64
	if readyOp.LeftValue != nil {
		leftVal = *readyOp.LeftValue
	}
	if readyOp.RightValue != nil {
		rightVal = *readyOp.RightValue
	}

	var opTime time.Duration
	switch readyOp.Operator {
	case "+":
		opTime = config.AppConfig.TimeAddition
	case "-":
		opTime = config.AppConfig.TimeSubtraction
	case "*":
		opTime = config.AppConfig.TimeMultiplication
	case "/":
		opTime = config.AppConfig.TimeDivision
	default:
		opTime = config.AppConfig.TimeAddition
	}

	return &proto.GetTaskResponse{
		HasTask:         true,
		Id:              uint32(readyOp.ID),
		LeftValue:       leftVal,
		RightValue:      rightVal,
		Operator:        readyOp.Operator,
		OperationTimeNs: int64(opTime),
	}, nil
}

// SendTaskResult обрабатывает результат выполнения задачи
func (s *OrchestratorService) SendTaskResult(ctx context.Context, req *proto.TaskResultRequest) (*proto.TaskResultResponse, error) {
	logger.INFO.Printf("gRPC: Получен результат задачи ID=%d", req.Id)
	// Преобразуем запрос в структуру TaskResult для оркестратора
	taskResult := orchestrator.TaskResult{
		ID:     int64(req.Id),
		Result: req.Result,
		Error:  req.Error,
	}

	// Обрабатываем результат через оркестратор
	err := orchestrator.ProcessExpressionResult(taskResult)
	if err != nil {
		logger.LogERROR("Ошибка обработки результата: " + err.Error())
		return &proto.TaskResultResponse{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	return &proto.TaskResultResponse{
		Success: true,
		Error:   "",
	}, nil
}

// StartGRPCServer запускает gRPC сервер на указанном адресе и возвращает экземпляр сервера
func StartGRPCServer(address string) (*grpc.Server, error) {
	// Создаем TCP слушатель
	lis, err := net.Listen("tcp", address)
	if err != nil {
		return nil, err
	}

	s := grpc.NewServer()

	proto.RegisterTaskServiceServer(s, &OrchestratorService{})

	// Запускаем сервер
	logger.INFO.Printf("gRPC оркестратор запущен на %s", address)

	go func() {
		if err := s.Serve(lis); err != nil && err != grpc.ErrServerStopped {
			logger.ERROR.Fatalf("Ошибка запуска gRPC оркестратора: %v", err)
		}
	}()

	return s, nil
}
