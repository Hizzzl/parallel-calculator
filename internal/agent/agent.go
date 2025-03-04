package agent

import (
	"encoding/json"
	"errors"
	"net/http"
	"parallel-calculator/internal/config"
	"parallel-calculator/internal/logger"
	"strconv"
	"strings"
	"time"
)

type Task struct {
	ID            uint32        `json:"id"`
	LeftValue     float64       `json:"arg1"`
	RightValue    float64       `json:"arg2"`
	Operator      string        `json:"operation"`
	OperationTime time.Duration `json:"operation_time"`
}

type TaskResult struct {
	ID     uint32  `json:"id"`
	Result float64 `json:"result"`
	Error  string  `json:"error"`
}

func GetTask() (*Task, error) {
	logger.INFO.Println("Sending get task request")

	resp, err := http.Get("http://localhost:8080/internal/task")
	if err != nil {
		logger.ERROR.Println("Failed to get task: ", err)
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		logger.INFO.Println("Received 404 response")
		// Если задач нет, сервер может вернуть 404 – обрабатываем как отсутствие задачи.
		return nil, nil
	}

	if resp.StatusCode != http.StatusOK {
		logger.ERROR.Println("Received non-200 response: ", resp.StatusCode)
		return nil, errors.New("received non-200 response: " + strconv.Itoa(resp.StatusCode))
	}

	var data struct {
		Task Task `json:"task"`
	}
	logger.INFO.Println("Received task: ", resp.Body)
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		logger.ERROR.Println("Failed to decode response: ", err)
		return nil, err
	}
	logger.INFO.Println("Received task: ", data.Task)
	return &data.Task, nil
}

func SendTaskResult(taskResult TaskResult) error {
	logger.INFO.Println("Sending task result: ", taskResult)
	resp, err := http.Post("http://localhost:8080/internal/task", "application/json",
		strings.NewReader(`{"id":`+strconv.Itoa(int(taskResult.ID))+
			`,"result":`+strconv.Itoa(int(taskResult.Result))+
			`,"error":"`+taskResult.Error+`"}`))
	if err != nil {
		logger.ERROR.Println("Failed to send task result: ", err)
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		logger.ERROR.Println("Failed to send task result: status code is not 200")
		return errors.New("status code is not 200")
	}
	logger.INFO.Println("Task result sent successfully")
	return nil
}

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

func StartAgent() {
	cp := config.AppConfig.ComputingPower
	if cp == 0 {
		logger.ERROR.Println("COMPUTING_POWER not set. AUTO SET TO 1")
		cp = 1
	}
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

	logger.INFO.Println("Starting Agent")
	for {
		time.Sleep(config.AppConfig.AgentRequestTimeout)
		task, err := GetTask()
		if err != nil {
			logger.ERROR.Println(err)
			return
		}
		if task == nil {
			continue
		}
		tasks_chan <- *task
	}
}
