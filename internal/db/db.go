package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"parallel-calculator/internal/config"

	_ "github.com/mattn/go-sqlite3"
)

var (
	DB *sql.DB

	// Мьютекс для защиты параллельных операций с базой данных
	DbMutex sync.Mutex
)

// InitDB инициализирует соединение с базой данных SQLite
func InitDB(packagePath string) error {
	// Получаем путь к базе данных из конфигурации
	dbPath := config.AppConfig.DBPath

	// Проверка, что директория для базы данных существует
	dbDir := filepath.Dir(dbPath)
	if _, err := os.Stat(dbDir); os.IsNotExist(err) {
		if err := os.MkdirAll(dbDir, 0755); err != nil {
			return fmt.Errorf("failed to create database directory: %w", err)
		}
	}

	var err error

	// Открываем соединение с базой данных
	DB, err = sql.Open("sqlite3", dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	// Проверяем соединение
	if err = DB.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	if err = ApplySchema(filepath.Join(packagePath, "schema.sql")); err != nil {
		return fmt.Errorf("failed to apply schema: %w", err)
	}

	return nil
}

func CloseDB() error {
	if DB != nil {
		return DB.Close()
	}
	return nil
}

// CleanupDB очищает все таблицы в базе данных (используется для тестирования)
func CleanupDB() error {
	DbMutex.Lock()
	defer DbMutex.Unlock()

	tables := []string{"operations", "expressions", "users"}

	for _, table := range tables {
		_, err := DB.Exec("DELETE FROM " + table)
		if err != nil {
			return fmt.Errorf("failed to clean up table %s: %w", table, err)
		}
	}

	// Сбрасываем автоинкрементные счетчики
	for _, table := range tables {
		_, err := DB.Exec("DELETE FROM sqlite_sequence WHERE name=?", table)
		if err != nil {
			return fmt.Errorf("failed to reset auto-increment for table %s: %w", table, err)
		}
	}

	return nil
}

// applySchema выполняет SQL из файла schema.sql
func ApplySchema(schemaPath string) error {

	schemaBytes, err := os.ReadFile(schemaPath)
	if err != nil {
		return fmt.Errorf("failed to read schema file: %w", err)
	}

	_, err = DB.Exec(string(schemaBytes))
	if err != nil {
		return fmt.Errorf("failed to execute schema: %w", err)
	}

	return nil
}
