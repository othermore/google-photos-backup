package utils

import (
	"os/exec"
	"runtime"
)

// OpenBrowser abre la URL especificada en el navegador por defecto del sistema
func OpenBrowser(url string) error {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start"}
	case "darwin": // Mac OS
		cmd = "open"
	default: // Linux, BSD, etc
		cmd = "xdg-open"
	}
	
	args = append(args, url)
	return exec.Command(cmd, args...).Start()
}