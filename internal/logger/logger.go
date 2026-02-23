package logger

import (
	"fmt"
	"os"
	"path/filepath"
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

	// Extract the directory and try to create it if it doesn't exist
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		fmt.Printf("⚠️  Warning: could not create log directory: %v\n", err)
		return err
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("⚠️  Warning: could not initialize log file: %v\n", err)
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
		msg := fmt.Sprintf("[DEBUG] "+format, args...)
		fmt.Println(msg)
		LogToFile("%s", msg)
	}
}

// Info always prints
func Info(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Println(msg)
	LogToFile("INFO: %s", msg)
}

// Warn always prints with a warning icon
func Warn(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Println("⚠️  " + msg)
	LogToFile("WARN: %s", msg)
}

// Error always prints
func Error(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Println("❌ " + msg)
	LogToFile("ERROR: %s", msg)
}

// Fatalf prints an error and exits the program with code 1
func Fatalf(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Println("❌ " + msg)
	LogToFile("FATAL: %s", msg)
	os.Exit(1)
}
