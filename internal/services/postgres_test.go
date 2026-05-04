package services

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderPostgresUnit(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_STATE_HOME", t.TempDir())

	unit, err := RenderPostgresUnit("16", "/usr/bin/postgres")
	if err != nil {
		t.Fatal(err)
	}

	for _, want := range []string{
		"Description=routa Postgres 16",
		"ExecStart=/usr/bin/postgres -D " + filepath.Join(os.Getenv("XDG_DATA_HOME"), "routa", "services", "postgres", "16") + " -c config_file=" + filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "routa", "services", "postgres", "16", "postgresql.conf"),
		"StandardOutput=append:" + filepath.Join(os.Getenv("XDG_STATE_HOME"), "routa", "log", "postgres-16.log"),
		"StandardError=append:" + filepath.Join(os.Getenv("XDG_STATE_HOME"), "routa", "log", "postgres-16.log"),
		"Restart=on-failure",
		"WantedBy=default.target",
	} {
		if !strings.Contains(unit, want) {
			t.Fatalf("Postgres unit missing %q:\n%s", want, unit)
		}
	}
}

func TestRenderPostgresConfig(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())

	conf, err := RenderPostgresConfig("16")
	if err != nil {
		t.Fatal(err)
	}

	for _, want := range []string{
		"listen_addresses = '127.0.0.1'",
		"port = 5432",
		"unix_socket_directories = '" + filepath.Join(os.Getenv("XDG_STATE_HOME"), "routa", "run") + "'",
		"external_pid_file = '" + filepath.Join(os.Getenv("XDG_STATE_HOME"), "routa", "run", "postgres-16.pid") + "'",
		"logging_collector = off",
		"log_destination = 'stderr'",
	} {
		if !strings.Contains(conf, want) {
			t.Fatalf("Postgres config missing %q:\n%s", want, conf)
		}
	}
}

func TestPostgresPathsAreVersionScoped(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_STATE_HOME", t.TempDir())

	version := "15"
	tests := map[string]string{
		"unit":   "routa-postgres@15.service",
		"config": filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "routa", "services", "postgres", version, "postgresql.conf"),
		"data":   filepath.Join(os.Getenv("XDG_DATA_HOME"), "routa", "services", "postgres", version),
		"pid":    filepath.Join(os.Getenv("XDG_STATE_HOME"), "routa", "run", "postgres-"+version+".pid"),
		"log":    filepath.Join(os.Getenv("XDG_STATE_HOME"), "routa", "log", "postgres-"+version+".log"),
	}

	got := map[string]string{
		"unit":   PostgresUnitName(version),
		"config": PostgresConfigPath(version),
		"data":   PostgresDataDir(version),
		"pid":    PostgresPIDFile(version),
		"log":    PostgresLogPath(version),
	}
	for name, want := range tests {
		if got[name] != want {
			t.Fatalf("%s path = %q, want %q", name, got[name], want)
		}
	}
}
