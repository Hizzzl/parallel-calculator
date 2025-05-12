package agent

import (
	"parallel-calculator/internal/logger"
	"time"
)

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
		err := SendTaskResult(taskResult)
		if err != nil {
			logger.ERROR.Println(err)
			return
		}
	}
}
