package agent

import (
	"parallel-calculator/internal/logger"
	"time"
)

// Глобальный клиент gRPC для работы с оркестратором
var globalClient TaskClient

// Устанавливает глобального клиента для использования рабочими потоками
func SetGlobalClient(client TaskClient) {
	globalClient = client
}

// Worker обрабатывает поступающие задачи из канала
func Worker(tasks_chan chan Task, worker_id int) {
	for task := range tasks_chan {
		logger.INFO.Println("Worker", worker_id, "received task: ", task)
		result := 0.0
		Error := "nil"

		switch task.Operator {
		case "+":
			result = task.LeftValue + task.RightValue
		case "-":
			result = task.LeftValue - task.RightValue
		case "*":
			result = task.LeftValue * task.RightValue
		case "/":
			if task.RightValue == 0 {
				result = 0
				Error = "division by zero"
			} else {
				result = task.LeftValue / task.RightValue
			}
		}
		time.Sleep(task.OperationTime)
		taskResult := TaskResult{
			ID:     task.ID,
			Result: result,
			Error:  Error,
		}
		
				if globalClient == nil {
			logger.ERROR.Printf("Worker %d: глобальный клиент не установлен", worker_id)
			return
		}
		
		// gRPC клиенты потокобезопасны по умолчанию
		err := globalClient.SendTaskResult(taskResult)
		
		if err != nil {
			logger.ERROR.Printf("Worker %d: ошибка отправки результата: %v", worker_id, err)
			return
		}
	}
}
