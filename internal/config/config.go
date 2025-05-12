package config

import (
	"log"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	AgentLogFilePath    string
	ClientLogFilePath   string
	ComputingPower      int
	TimeAddition        time.Duration
	TimeSubtraction     time.Duration
	TimeMultiplication  time.Duration
	TimeDivision        time.Duration
	AgentRequestTimeout time.Duration
	ServerPort          string
	OrchestratorBaseURL string
	// База данных и аутентификация
	DBPath              string
	JWTSecret           string
	JWTExpirationMinutes int
}

var (
	AppConfig *Config
	Once      sync.Once
)

func InitConfig(configPath string) {
	AppConfig = &Config{}

	if _, err := os.Stat(configPath); err == nil {
		err := godotenv.Load(configPath)
		if err != nil {
			log.Fatal("Error loading .env file")
		}
	} else {
		log.Println("/.env not found")
		// current path
		log.Println(os.Getwd())
	}

	if os.Getenv("AGENT_LOG_FILE_PATH") != "" {
		AppConfig.AgentLogFilePath = os.Getenv("AGENT_LOG_FILE_PATH")
	} else {
		AppConfig.AgentLogFilePath = os.Getenv("AGENT_LOG_FILE_PATH")
	}

	if os.Getenv("CLIENT_LOG_FILE_PATH") != "" {
		AppConfig.ClientLogFilePath = os.Getenv("CLIENT_LOG_FILE_PATH")
	} else {
		AppConfig.ClientLogFilePath = os.Getenv("CLIENT_LOG_FILE_PATH")
	}

	if os.Getenv("COMPUTING_POWER") != "" {
		var err error
		AppConfig.ComputingPower, err = strconv.Atoi(os.Getenv("COMPUTING_POWER"))
		if err != nil {
			log.Fatal("COMPUTING_POWER not a number")
		}
	} else {
		log.Fatal("COMPUTING_POWER not set")
	}

	if os.Getenv("TIME_ADDITION_MS") != "" {
		value, err := strconv.Atoi(os.Getenv("TIME_ADDITION_MS"))
		if err != nil {
			log.Fatal("TIME_ADDITION_MS not a number")
		}
		AppConfig.TimeAddition = time.Duration(value) * time.Millisecond
	} else {
		log.Fatal("TIME_ADDITION_MS not set")
	}

	if os.Getenv("TIME_SUBTRACTION_MS") != "" {
		value, err := strconv.Atoi(os.Getenv("TIME_SUBTRACTION_MS"))
		if err != nil {
			log.Fatal("TIME_SUBTRACTION_MS not a number")
		}
		AppConfig.TimeSubtraction = time.Duration(value) * time.Millisecond
	} else {
		log.Fatal("TIME_SUBTRACTION_MS not set")
	}

	if os.Getenv("TIME_MULTIPLICATION_MS") != "" {
		value, err := strconv.Atoi(os.Getenv("TIME_MULTIPLICATION_MS"))
		if err != nil {
			log.Fatal("TIME_MULTIPLICATION_MS not a number")
		}
		AppConfig.TimeMultiplication = time.Duration(value) * time.Millisecond
	} else {
		log.Fatal("TIME_MULTIPLICATION_MS not set")
	}

	if os.Getenv("TIME_DIVISION_MS") != "" {
		value, err := strconv.Atoi(os.Getenv("TIME_DIVISION_MS"))
		if err != nil {
			log.Fatal("TIME_DIVISION_MS not a number")
		}
		AppConfig.TimeDivision = time.Duration(value) * time.Millisecond
	} else {
		log.Fatal("TIME_DIVISION_MS not set")
	}

	if os.Getenv("AGENT_REQUEST_TIMEOUT_MS") != "" {
		value, err := strconv.Atoi(os.Getenv("AGENT_REQUEST_TIMEOUT_MS"))
		if err != nil {
			log.Fatal("AGENT_REQUEST_TIMEOUT_MS not a number")
		}
		AppConfig.AgentRequestTimeout = time.Duration(value) * time.Millisecond
	} else {
		log.Fatal("AGENT_REQUEST_TIMEOUT_MS not set")
	}

	if os.Getenv("SERVER_PORT") != "" {
		AppConfig.ServerPort = os.Getenv("SERVER_PORT")
	} else {
		AppConfig.ServerPort = "8080"
	}

	if os.Getenv("ORCHESTRATOR_BASE_URL") != "" {
		AppConfig.OrchestratorBaseURL = os.Getenv("ORCHESTRATOR_BASE_URL")
	} else {
		AppConfig.OrchestratorBaseURL = "http://localhost:8080"
	}

	// Инициализация параметров базы данных
	if os.Getenv("DB_PATH") != "" {
		AppConfig.DBPath = os.Getenv("DB_PATH")
	} else {
		AppConfig.DBPath = "./data/calculator.db"
		log.Println("Using default DB_PATH: ./data/calculator.db")
	}

	if os.Getenv("JWT_SECRET") != "" {
		AppConfig.JWTSecret = os.Getenv("JWT_SECRET")
	} else {
		log.Println("WARNING: Using default JWT_SECRET. This is insecure for production!")
		AppConfig.JWTSecret = "your-secret-key-for-jwt-signing"
	}

	// Срок действия JWT токена в часах
	if os.Getenv("JWT_EXPIRATION_MINUTES") != "" {
		expiration, err := strconv.Atoi(os.Getenv("JWT_EXPIRATION_MINUTES"))
		if err != nil {
			log.Println("WARNING: JWT_EXPIRATION_MINUTES not a valid number, using default value of 1440 minutes (24 hours)")
			AppConfig.JWTExpirationMinutes = 1440
		} else {
			AppConfig.JWTExpirationMinutes = expiration
		}
	} else {
		log.Println("JWT_EXPIRATION_MINUTES not set, using default value of 1440 minutes (24 hours)")
		AppConfig.JWTExpirationMinutes = 1440
	}
}
