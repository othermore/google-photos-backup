package logger

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/spf13/viper"
)

var (
	logFile *os.File
	logMu   sync.Mutex
)

// InitLogFile opens the given path for appending logs.
func InitLogFile(path string) error {
	logMu.Lock()
	defer logMu.Unlock()
	if logFile != nil {
		logFile.Close()
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	logFile = f
	return nil
}

// CloseLogFile safely closes the global log file.
func CloseLogFile() error {
	logMu.Lock()
	defer logMu.Unlock()
	if logFile != nil {
		err := logFile.Close()
		logFile = nil
		return err
	}
	return nil
}

// LogToFile appends a timestamped log strictly to the file (if initialized).
func LogToFile(format string, args ...interface{}) {
	logMu.Lock()
	defer logMu.Unlock()
	if logFile == nil {
		return
	}
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(logFile, "[%s] %s\n", timestamp, msg)
}

// Debug prints only if verbose mode is enabled
func Debug(format string, args ...interface{}) {
	if viper.GetBool("verbose") {
		fmt.Printf("[DEBUG] "+format+"\n", args...)
	}
}

// Info always prints
func Info(format string, args ...interface{}) {
	fmt.Printf(format+"\n", args...)
}

// Warn always prints with a warning icon
func Warn(format string, args ...interface{}) {
	fmt.Printf("⚠️  "+format+"\n", args...)
}

// Error always prints
func Error(format string, args ...interface{}) {
	fmt.Printf("❌ "+format+"\n", args...)
}
