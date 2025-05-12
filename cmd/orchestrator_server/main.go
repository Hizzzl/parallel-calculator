package main

import (
	"log"
	"net/http"
	"parallel-calculator/internal/auth"
	"parallel-calculator/internal/config"
	"parallel-calculator/internal/db"
	"parallel-calculator/internal/logger"
	"parallel-calculator/internal/orchestrator"

	"github.com/gorilla/mux"
)

func main() {
	config.InitConfig(".env")
	logger.InitClientLogger()
	defer logger.CloseLogger()

	// Инициализируем базу данных
	err := db.InitDB()
	if err != nil {
		logger.ERROR.Fatalf("Ошибка инициализации базы данных: %v", err)
	}

	r := mux.NewRouter()

	// Публичные эндпоинты для аутентификации
	r.HandleFunc("/api/v1/register", auth.Register).Methods("POST")
	r.HandleFunc("/api/v1/login", auth.Login).Methods("POST")

	// Защищенные маршруты для пользовательского API
	protected := r.PathPrefix("/api/v1").Subrouter()
	protected.Use(auth.AuthMiddleware)
	protected.HandleFunc("/calculate", orchestrator.HandleCalculate).Methods("POST")
	protected.HandleFunc("/expressions", orchestrator.HandleGetExpressions).Methods("GET")
	protected.HandleFunc("/expressions/{id}", orchestrator.HandleGetExpressionByID).Methods("GET")

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
