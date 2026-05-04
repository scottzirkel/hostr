package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/scottzirkel/routa/internal/services"
)

func TestDBEngineVersionArgs(t *testing.T) {
	if err := dbEngineVersionArgs(dbInstallCmd, []string{"mariadb", "11.4"}); err != nil {
		t.Fatal(err)
	}
	if err := dbEngineVersionArgs(dbInstallCmd, []string{"postgres", "16"}); err != nil {
		t.Fatal(err)
	}

	for _, args := range [][]string{
		{"sqlite", "3"},
		{"mariadb", "../11.4"},
		{"postgres", "../16"},
		{"mariadb"},
	} {
		if err := dbEngineVersionArgs(dbInstallCmd, args); err == nil {
			t.Fatalf("expected error for args %#v", args)
		}
	}
}

func TestDatabasePortFromCommandAcceptsOnAlias(t *testing.T) {
	got, err := databasePortFromCommand(dbStartCmd, []string{"on", "3307"}, "mariadb", "")
	if err != nil {
		t.Fatal(err)
	}
	if got != "3307" {
		t.Fatalf("port = %q", got)
	}
}

func TestDBListShowsMariaDBInstances(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	if err := os.MkdirAll(services.MariaDBDataDir("11.4"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(services.PostgresDataDir("16"), 0o755); err != nil {
		t.Fatal(err)
	}
	unitPath := filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "systemd", "user", services.MariaDBUnitName("10.11"))
	if err := os.MkdirAll(filepath.Dir(unitPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(unitPath, []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	dbListCmd.SetOut(&out)
	dbListCmd.SetErr(&bytes.Buffer{})
	t.Cleanup(func() {
		dbListCmd.SetOut(os.Stdout)
		dbListCmd.SetErr(os.Stderr)
	})

	if err := dbListCmd.RunE(dbListCmd, nil); err != nil {
		t.Fatal(err)
	}

	body := out.String()
	for _, want := range []string{
		"ENGINE",
		"mariadb",
		"10.11",
		"11.4",
		"postgres",
		"16",
		services.MariaDBUnitName("10.11"),
		services.MariaDBDataDir("11.4"),
		services.PostgresDataDir("16"),
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("db list output missing %q:\n%s", want, body)
		}
	}
}
