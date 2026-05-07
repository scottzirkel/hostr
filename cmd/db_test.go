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
	if err := dbEngineVersionArgs(dbInstallCmd, []string{"mysql", "8.0"}); err != nil {
		t.Fatal(err)
	}
	if err := dbEngineVersionArgs(dbInstallCmd, []string{"postgres", "16"}); err != nil {
		t.Fatal(err)
	}

	for _, args := range [][]string{
		{"sqlite", "3"},
		{"mariadb", "../11.4"},
		{"mysql", "../8.0"},
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

func TestDatabaseTargetFromArgsAcceptsNamedInstanceWithPortAlias(t *testing.T) {
	target, portArgs, err := databaseTargetFromArgs(dbStartCmd, []string{"mysql", "8.0", "affiliate-platform", "on", "3309"}, true)
	if err != nil {
		t.Fatal(err)
	}
	if target.Engine != "mysql" || target.Version != "8.0" || target.Instance != "affiliate-platform" {
		t.Fatalf("target = %#v", target)
	}
	if strings.Join(portArgs, " ") != "on 3309" {
		t.Fatalf("port args = %#v", portArgs)
	}

	unit := databaseUnitName(target.Engine, target.Version, target.Instance)
	if unit != "routa-mysql@8.0_affiliate-platform.service" {
		t.Fatalf("unit = %q", unit)
	}
}

func TestDatabaseTargetFromArgsKeepsDefaultInstancePortAlias(t *testing.T) {
	target, portArgs, err := databaseTargetFromArgs(dbStartCmd, []string{"mysql", "8.0", "on", "3309"}, true)
	if err != nil {
		t.Fatal(err)
	}
	if target.Instance != "" {
		t.Fatalf("instance = %q", target.Instance)
	}
	if strings.Join(portArgs, " ") != "on 3309" {
		t.Fatalf("port args = %#v", portArgs)
	}
}

func TestDatabaseCredentialsFromFlags(t *testing.T) {
	creds, ok, err := databaseCredentialsFromFlags("mysql", "affiliate", "secret")
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected credentials")
	}
	if creds.User != "affiliate" || creds.Password != "secret" {
		t.Fatalf("credentials = %#v", creds)
	}
}

func TestDatabaseCredentialsFromFlagsRejectsNonMySQL(t *testing.T) {
	if _, _, err := databaseCredentialsFromFlags("postgres", "app", "secret"); err == nil {
		t.Fatal("expected non-mysql error")
	}
}

func TestDatabaseCredentialsFromFlagsRequiresUser(t *testing.T) {
	if _, _, err := databaseCredentialsFromFlags("mysql", "", "secret"); err == nil {
		t.Fatal("expected missing user error")
	}
}

func TestDBListShowsDatabaseInstances(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_STATE_HOME", t.TempDir())

	if err := services.WriteMariaDBConfigWithPort("11.4", "3314"); err != nil {
		t.Fatal(err)
	}
	if err := services.WriteMySQLConfigWithPort("8.0", "3308"); err != nil {
		t.Fatal(err)
	}
	if err := services.WritePostgresConfigWithPort("16", "5416"); err != nil {
		t.Fatal(err)
	}
	if err := services.WriteMySQLConfigForInstanceWithPort("8.0", "affiliate-platform", "3309"); err != nil {
		t.Fatal(err)
	}

	unitPath := filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "systemd", "user", services.MariaDBUnitName("10.11"))
	touchFile(t, unitPath)

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
		"INSTANCE",
		"PORT",
		"mariadb",
		"10.11",
		"11.4",
		"3314",
		"mysql",
		"8.0",
		"3308",
		"3309",
		"default",
		"affiliate-platform",
		"postgres",
		"16",
		"5416",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("db list output missing %q:\n%s", want, body)
		}
	}
	for _, unwanted := range []string{
		"UNIT",
		"DATA_DIR",
		services.MariaDBUnitName("10.11"),
		services.MariaDBDataDir("11.4"),
		services.MySQLDataDir("8.0"),
		services.MySQLDataDirForInstance("8.0", "affiliate-platform"),
		services.PostgresDataDir("16"),
	} {
		if strings.Contains(body, unwanted) {
			t.Fatalf("db list output should not include %q:\n%s", unwanted, body)
		}
	}
}

func touchFile(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}
}
