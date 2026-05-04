package services

import (
	"fmt"
	"path/filepath"

	"github.com/scottzirkel/routa/internal/paths"
)

const (
	MailpitUnitName = "routa-mailpit.service"
	MailpitWebPort  = "8025"
	MailpitSMTPPort = "1025"
)

const mailpitUnitTmpl = `[Unit]
Description=routa Mailpit
After=network.target

[Service]
Type=simple
ExecStart={{.Binary}} --listen {{.WebAddr}} --smtp {{.SMTPAddr}} --database {{.DatabasePath}}
Restart=on-failure
RestartSec=2
TimeoutStopSec=5

[Install]
WantedBy=default.target
`

type mailpitUnitData struct {
	Binary       string
	WebAddr      string
	SMTPAddr     string
	DatabasePath string
}

func Mailpit() Definition {
	return MailpitWithPorts(MailpitWebPort, MailpitSMTPPort)
}

func MailpitWithPorts(webPort, smtpPort string) Definition {
	return Definition{
		Name:        "mailpit",
		UnitName:    MailpitUnitName,
		BinaryName:  "mailpit",
		DataDir:     MailpitDataDir(),
		RenderUnit:  func(binary string) (string, error) { return RenderMailpitUnitWithPorts(binary, webPort, smtpPort) },
		WriteConfig: EnsureMailpitDataDir,
	}
}

func MailpitWebAddr() string {
	return "127.0.0.1:" + MailpitWebPort
}

func MailpitSMTPAddr() string {
	return "127.0.0.1:" + MailpitSMTPPort
}

func MailpitDataDir() string {
	return filepath.Join(paths.DataDir(), "services", "mailpit")
}

func MailpitDatabasePath() string {
	return filepath.Join(MailpitDataDir(), "mailpit.db")
}

func RenderMailpitUnit(binary string) (string, error) {
	return RenderMailpitUnitWithPorts(binary, MailpitWebPort, MailpitSMTPPort)
}

func RenderMailpitUnitWithPorts(binary, webPort, smtpPort string) (string, error) {
	if err := ValidateTCPPort("Mailpit web", webPort); err != nil {
		return "", err
	}
	if err := ValidateTCPPort("Mailpit SMTP", smtpPort); err != nil {
		return "", err
	}
	if binary == "" {
		return "", fmt.Errorf("mailpit binary path cannot be empty")
	}
	return render("mailpit-unit", mailpitUnitTmpl, mailpitUnitData{
		Binary:       binary,
		WebAddr:      "127.0.0.1:" + webPort,
		SMTPAddr:     "127.0.0.1:" + smtpPort,
		DatabasePath: MailpitDatabasePath(),
	})
}

func EnsureMailpitDataDir() error {
	return ensureDir(MailpitDataDir())
}
