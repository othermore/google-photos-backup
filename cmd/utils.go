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

func calculateHash(filePath string) (hashStr string, err error) {
	file, openErr := os.Open(filePath)
	if openErr != nil {
		return "", openErr
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	}()

	hash := sha256.New()
	if _, copyErr := io.Copy(hash, file); copyErr != nil {
		return "", copyErr
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}
