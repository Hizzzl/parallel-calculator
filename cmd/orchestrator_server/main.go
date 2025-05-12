package main

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"parallel-calculator/internal/auth"
	"parallel-calculator/internal/config"
	"parallel-calculator/internal/db"
	"parallel-calculator/internal/grpc"
	"parallel-calculator/internal/logger"
	"parallel-calculator/internal/orchestrator"
	"strconv"
	"syscall"

	"github.com/gorilla/mux"
)

func main() {
	config.InitConfig(".env")
	logger.InitClientLogger()
	defer logger.CloseLogger()

	// Инициализируем базу данных
	err := db.InitDB("internal/db/")
	if err != nil {
		logger.ERROR.Fatalf("Ошибка инициализации базы данных: %v", err)
	}

	// Настраиваем HTTP сервер
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

	// Запускаем HTTP сервер в отдельной горутине
	httpServer := &http.Server{
		Addr:    ":" + config.AppConfig.ServerPort,
		Handler: r,
	}

	go func() {
		logger.INFO.Println("HTTP сервер запущен на порту " + config.AppConfig.ServerPort)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.ERROR.Fatalf("Ошибка запуска HTTP сервера: %v", err)
		}
	}()

	// Запускаем gRPC сервер в отдельной горутине
	// Используем порт HTTP + 1 для gRPC
	httpPort, _ := strconv.Atoi(config.AppConfig.ServerPort)
	grpcPort := fmt.Sprintf("%d", httpPort+1)
	grpcAddress := ":" + grpcPort

	// Используем обновленную сигнатуру, но игнорируем возвращаемый сервер
	_, err = grpc.StartGRPCServer(grpcAddress)
	if err != nil {
		logger.ERROR.Fatalf("Ошибка запуска gRPC сервера: %v", err)
	}
	logger.INFO.Println("gRPC сервер запущен на порту " + grpcPort)

	// Ожидаем сигнал для остановки
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	<-sigCh

	logger.INFO.Println("Получен сигнал остановки, завершаем работу серверов...")
	logger.INFO.Println("Оркестратор остановлен")
}
