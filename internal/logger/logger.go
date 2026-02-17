package logger

import (
	"fmt"

	"github.com/spf13/viper"
)

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
