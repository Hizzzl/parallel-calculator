package main

import (
	"parallel-calculator/internal/agent"
	"parallel-calculator/internal/config"
	"parallel-calculator/internal/logger"
)

func main() {
	config.InitConfig("configs/.env")
	logger.InitAgentLogger()
	defer logger.CloseLogger()

	logger.INFO.Println("Agent server started")
	defer logger.INFO.Println("Agent server stopped")

	agent.StartAgent()
}
