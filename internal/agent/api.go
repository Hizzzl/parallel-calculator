package agent

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"parallel-calculator/internal/config"
	"parallel-calculator/internal/logger"
	"strconv"
)

// GetTask выполняет HTTP-запрос к оркестратору для получения задачи на выполнение
func GetTask() (*Task, error) {
	logger.INFO.Println("Sending get task request")

	url := config.AppConfig.OrchestratorBaseURL + "/internal/task"
	resp, err := http.Get(url)
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

// SendTaskResult отправляет результат выполнения задачи обратно оркестратору
func SendTaskResult(taskResult TaskResult) error {
	logger.INFO.Println("Sending task result: ", taskResult)
	
	url := config.AppConfig.OrchestratorBaseURL + "/internal/task"
	
	// Используем json.Marshal вместо ручного формирования JSON
	data, err := json.Marshal(taskResult)
	if err != nil {
		logger.ERROR.Println("Failed to marshal task result: ", err)
		return err
	}
	
	// Создаем payload из маршалинга
	resp, err := http.Post(url, "application/json", bytes.NewReader(data))
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
