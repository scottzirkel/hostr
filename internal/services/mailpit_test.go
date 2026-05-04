package services

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderMailpitUnit(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	unit, err := RenderMailpitUnit("/usr/bin/mailpit")
	if err != nil {
		t.Fatal(err)
	}

	for _, want := range []string{
		"Description=routa Mailpit",
		"ExecStart=/usr/bin/mailpit --listen 127.0.0.1:8025 --smtp 127.0.0.1:1025 --database " + filepath.Join(os.Getenv("XDG_DATA_HOME"), "routa", "services", "mailpit", "mailpit.db"),
		"Restart=on-failure",
		"WantedBy=default.target",
	} {
		if !strings.Contains(unit, want) {
			t.Fatalf("Mailpit unit missing %q:\n%s", want, unit)
		}
	}
}

func TestWriteFilesCreatesMailpitUnitAndDataDir(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	if err := WriteFiles(Mailpit(), "/usr/bin/mailpit"); err != nil {
		t.Fatal(err)
	}

	for _, path := range []string{
		MailpitDataDir(),
		filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "systemd", "user", MailpitUnitName),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected %s: %v", path, err)
		}
	}
}
