package main

import (
	"log"
	"net/http"
	"parallel-calculator/internal/config"
	"parallel-calculator/internal/logger"
	"parallel-calculator/internal/orchestrator"

	"github.com/gorilla/mux"
)

func main() {
	config.InitConfig("configs/.env")
	logger.InitClientLogger()
	defer logger.CloseLogger()

	r := mux.NewRouter()
	// Регистрируем эндпоинты для пользовательского API
	r.HandleFunc("/api/v1/calculate", orchestrator.HandleCalculate)
	r.HandleFunc("/api/v1/expressions", orchestrator.HandleGetExpressions)
	r.HandleFunc("/api/v1/expressions/{id}", orchestrator.HandleGetExpressionByID) // id передаём через query-параметр

	// Эндпоинты для внутренних запросов от агента
	r.HandleFunc("/internal/task", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			orchestrator.HandleGetTask(w, r)
		} else if r.Method == http.MethodPost {
			orchestrator.HandlePostTaskResult(w, r)
		} else {
			http.Error(w, "Метод не поддерживается", http.StatusMethodNotAllowed)
		}
	})

	logger.INFO.Println("Orchestrator server started on port " + config.AppConfig.ServerPort)
	defer logger.INFO.Println("Orchestrator server stopped")

	log.Fatal(http.ListenAndServe(":"+config.AppConfig.ServerPort, r))
}
