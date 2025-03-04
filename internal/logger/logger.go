package logger

import (
	"log"
	"os"
	"parallel-calculator/internal/config"
	"sync"
)

var (
	fileMutex sync.Mutex
	INFO      *log.Logger
	ERROR     *log.Logger
	logFile   *os.File
)

func LogINFO(s string) {
	if INFO == nil {
		return
	}
	INFO.Println(s)
}

func LogERROR(s string) {
	if ERROR == nil {
		return
	}
	ERROR.Println(s)
}

type lockedFile struct {
	file *os.File
}

func (lf *lockedFile) Write(p []byte) (n int, err error) {
	fileMutex.Lock()
	defer fileMutex.Unlock()
	return lf.file.Write(p)
}

func InitAgentLogger() {
	config := config.AppConfig

	if config.AgentLogFilePath == "" {
		log.Fatal("agent log file path not set")
	}

	logFile, err := os.OpenFile(config.AgentLogFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Printf("Failed to open log file: %v. We will use standard output", err)
		INFO = log.New(os.Stdout, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
		ERROR = log.New(os.Stdout, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
		return
	}

	writer := &lockedFile{file: logFile}
	INFO = log.New(writer, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	ERROR = log.New(writer, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
}

func InitClientLogger() {
	config := config.AppConfig

	if config.ClientLogFilePath == "" {
		log.Fatal("client log file path not set")
	}

	logFile, err := os.OpenFile(config.ClientLogFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Printf("Failed to open log file: %v. We will use standard output", err)
		INFO = log.New(os.Stdout, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
		ERROR = log.New(os.Stdout, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
		return
	}

	writer := &lockedFile{file: logFile}
	INFO = log.New(writer, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	ERROR = log.New(writer, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
}

func CloseLogger() {
	if logFile != nil {
		logFile.Close()
	}
}
