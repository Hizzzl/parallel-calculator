package agent

import (
	"parallel-calculator/internal/config"
	"parallel-calculator/internal/logger"
	"time"
)

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

	logger.INFO.Println("Starting Agent")
	for {
		time.Sleep(config.AppConfig.AgentRequestTimeout)
		task, err := GetTask()
		if err != nil {
			logger.ERROR.Println(err)
			// Не завершаем агента, просто продолжаем работу
			continue
		}
		if task == nil {
			continue
		}
		tasks_chan <- *task
	}
}
