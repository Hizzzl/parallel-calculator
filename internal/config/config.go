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
			log.Println("AGENT_REQUEST_TIMEOUT_MS not a number. Auto set to 500ms")
			AppConfig.AgentRequestTimeout = 500 * time.Millisecond
		}
		AppConfig.AgentRequestTimeout = time.Duration(value) * time.Millisecond
	} else {
		log.Println("AGENT_REQUEST_TIMEOUT_MS not set. Auto set to 500ms")
		AppConfig.AgentRequestTimeout = 500 * time.Millisecond
	}

	if os.Getenv("SERVER_PORT") != "" {
		value := os.Getenv("SERVER_PORT")
		AppConfig.ServerPort = value
	} else {
		log.Println("SERVER_PORT not set. Auto set to 8080")
		AppConfig.ServerPort = "8080"
	}
}
