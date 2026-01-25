package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// prompt pide un dato al usuario. Si defaultVal no está vacío, lo muestra y lo usa si el usuario da Enter.
func prompt(label string, defaultVal string) string {
	msg := label
	if defaultVal != "" {
		msg = fmt.Sprintf("%s [%s]", label, defaultVal)
	}
	fmt.Printf("%s: ", msg)

	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	if input == "" {
		return defaultVal
	}
	return input
}
