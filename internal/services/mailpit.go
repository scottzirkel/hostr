package services

import (
	"fmt"
	"path/filepath"

	"github.com/scottzirkel/routa/internal/paths"
)

const (
	MailpitUnitName = "routa-mailpit.service"
	MailpitWebAddr  = "127.0.0.1:8025"
	MailpitSMTPAddr = "127.0.0.1:1025"
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
	return Definition{
		Name:        "mailpit",
		UnitName:    MailpitUnitName,
		BinaryName:  "mailpit",
		DataDir:     MailpitDataDir(),
		RenderUnit:  RenderMailpitUnit,
		WriteConfig: EnsureMailpitDataDir,
	}
}

func MailpitDataDir() string {
	return filepath.Join(paths.DataDir(), "services", "mailpit")
}

func MailpitDatabasePath() string {
	return filepath.Join(MailpitDataDir(), "mailpit.db")
}

func RenderMailpitUnit(binary string) (string, error) {
	if binary == "" {
		return "", fmt.Errorf("mailpit binary path cannot be empty")
	}
	return render("mailpit-unit", mailpitUnitTmpl, mailpitUnitData{
		Binary:       binary,
		WebAddr:      MailpitWebAddr,
		SMTPAddr:     MailpitSMTPAddr,
		DatabasePath: MailpitDatabasePath(),
	})
}

func EnsureMailpitDataDir() error {
	return ensureDir(MailpitDataDir())
}
