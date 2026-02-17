package notifier

import (
	"fmt"
	"os/exec"
	"strings"

	"google-photos-backup/internal/config"
	"google-photos-backup/internal/i18n"
	"google-photos-backup/internal/logger"
)

// SendAlert sends an email alert using the system's msmtp binary.
// It assumes msmtp is configured correctly on the host.
func SendAlert(subject, body string) error {
	recipient := config.AppConfig.EmailAlertTo
	if recipient == "" {
		logger.Warn(i18n.T("notifier_skipped"))
		return nil
	}

	// Check if msmtp exists
	if _, err := exec.LookPath("msmtp"); err != nil {
		return fmt.Errorf(i18n.T("notifier_no_binary"))
	}

	// Construct email message
	// To: <recipient>
	// Subject: <subject>
	//
	// <body>
	msg := fmt.Sprintf("To: %s\r\nSubject: %s\r\n\r\n%s", recipient, subject, body)

	cmd := exec.Command("msmtp", recipient)
	cmd.Stdin = strings.NewReader(msg)

	logger.Info(i18n.T("notifier_sending"), recipient)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf(i18n.T("notifier_fail"), err, string(output))
	}

	return nil
}
