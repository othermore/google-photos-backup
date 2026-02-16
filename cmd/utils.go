package cmd

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
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

func calculateHash(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}
