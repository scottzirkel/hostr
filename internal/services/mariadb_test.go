package services

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderMariaDBUnit(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	unit, err := RenderMariaDBUnit("11.4", "/usr/bin/mariadbd")
	if err != nil {
		t.Fatal(err)
	}

	for _, want := range []string{
		"Description=routa MariaDB 11.4",
		"ExecStart=/usr/bin/mariadbd --defaults-file=" + filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "routa", "services", "mariadb", "11.4", "my.cnf"),
		"Restart=on-failure",
		"WantedBy=default.target",
	} {
		if !strings.Contains(unit, want) {
			t.Fatalf("MariaDB unit missing %q:\n%s", want, unit)
		}
	}
}

func TestRenderMariaDBConfig(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_STATE_HOME", t.TempDir())

	conf, err := RenderMariaDBConfig("11.4")
	if err != nil {
		t.Fatal(err)
	}

	for _, want := range []string{
		"bind-address=127.0.0.1",
		"port=3306",
		"datadir=" + filepath.Join(os.Getenv("XDG_DATA_HOME"), "routa", "services", "mariadb", "11.4"),
		"socket=" + filepath.Join(os.Getenv("XDG_STATE_HOME"), "routa", "run", "mariadb-11.4.sock"),
		"pid-file=" + filepath.Join(os.Getenv("XDG_STATE_HOME"), "routa", "run", "mariadb-11.4.pid"),
		"log-error=" + filepath.Join(os.Getenv("XDG_STATE_HOME"), "routa", "log", "mariadb-11.4.log"),
		"[client]",
	} {
		if !strings.Contains(conf, want) {
			t.Fatalf("MariaDB config missing %q:\n%s", want, conf)
		}
	}
}

func TestMariaDBPathsAreVersionScoped(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_STATE_HOME", t.TempDir())

	version := "10.11"
	tests := map[string]string{
		"unit":   "routa-mariadb@10.11.service",
		"config": filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "routa", "services", "mariadb", version, "my.cnf"),
		"data":   filepath.Join(os.Getenv("XDG_DATA_HOME"), "routa", "services", "mariadb", version),
		"socket": filepath.Join(os.Getenv("XDG_STATE_HOME"), "routa", "run", "mariadb-"+version+".sock"),
		"pid":    filepath.Join(os.Getenv("XDG_STATE_HOME"), "routa", "run", "mariadb-"+version+".pid"),
		"log":    filepath.Join(os.Getenv("XDG_STATE_HOME"), "routa", "log", "mariadb-"+version+".log"),
	}

	got := map[string]string{
		"unit":   MariaDBUnitName(version),
		"config": MariaDBConfigPath(version),
		"data":   MariaDBDataDir(version),
		"socket": MariaDBSocketPath(version),
		"pid":    MariaDBPIDFile(version),
		"log":    MariaDBLogPath(version),
	}
	for name, want := range tests {
		if got[name] != want {
			t.Fatalf("%s path = %q, want %q", name, got[name], want)
		}
	}
}
