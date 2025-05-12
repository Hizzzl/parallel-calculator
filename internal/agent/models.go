package agent

import "time"

// Task представляет собой задачу для вычислений
type Task struct {
	ID            uint32        `json:"id"`
	LeftValue     float64       `json:"arg1"`
	RightValue    float64       `json:"arg2"`
	Operator      string        `json:"operation"`
	OperationTime time.Duration `json:"operation_time"`
}

// TaskResult представляет собой результат выполнения задачи
type TaskResult struct {
	ID     uint32  `json:"id"`
	Result float64 `json:"result"`
	Error  string  `json:"error"`
}
