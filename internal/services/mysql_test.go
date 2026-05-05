package services

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderMySQLUnit(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	unit, err := RenderMySQLUnit("8.0", "/usr/bin/mysqld")
	if err != nil {
		t.Fatal(err)
	}

	for _, want := range []string{
		"Description=routa MySQL 8.0",
		"ExecStart=/usr/bin/mysqld --defaults-file=" + filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "routa", "services", "mysql", "8.0", "my.cnf"),
		"Restart=on-failure",
		"WantedBy=default.target",
	} {
		if !strings.Contains(unit, want) {
			t.Fatalf("MySQL unit missing %q:\n%s", want, unit)
		}
	}
}

func TestRenderMySQLConfig(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_STATE_HOME", t.TempDir())

	conf, err := RenderMySQLConfig("8.0")
	if err != nil {
		t.Fatal(err)
	}

	for _, want := range []string{
		"bind-address=127.0.0.1",
		"port=3306",
		"datadir=" + filepath.Join(os.Getenv("XDG_DATA_HOME"), "routa", "services", "mysql", "8.0"),
		"socket=" + filepath.Join(os.Getenv("XDG_STATE_HOME"), "routa", "run", "mysql-8.0.sock"),
		"pid-file=" + filepath.Join(os.Getenv("XDG_STATE_HOME"), "routa", "run", "mysql-8.0.pid"),
		"log-error=" + filepath.Join(os.Getenv("XDG_STATE_HOME"), "routa", "log", "mysql-8.0.log"),
		"[client]",
	} {
		if !strings.Contains(conf, want) {
			t.Fatalf("MySQL config missing %q:\n%s", want, conf)
		}
	}
}

func TestMySQLPathsAreVersionScoped(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_STATE_HOME", t.TempDir())

	version := "8.4"
	tests := map[string]string{
		"unit":   "routa-mysql@8.4.service",
		"config": filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "routa", "services", "mysql", version, "my.cnf"),
		"data":   filepath.Join(os.Getenv("XDG_DATA_HOME"), "routa", "services", "mysql", version),
		"socket": filepath.Join(os.Getenv("XDG_STATE_HOME"), "routa", "run", "mysql-"+version+".sock"),
		"pid":    filepath.Join(os.Getenv("XDG_STATE_HOME"), "routa", "run", "mysql-"+version+".pid"),
		"log":    filepath.Join(os.Getenv("XDG_STATE_HOME"), "routa", "log", "mysql-"+version+".log"),
	}

	got := map[string]string{
		"unit":   MySQLUnitName(version),
		"config": MySQLConfigPath(version),
		"data":   MySQLDataDir(version),
		"socket": MySQLSocketPath(version),
		"pid":    MySQLPIDFile(version),
		"log":    MySQLLogPath(version),
	}
	for name, want := range tests {
		if got[name] != want {
			t.Fatalf("%s path = %q, want %q", name, got[name], want)
		}
	}
}

func TestMySQLPathsAreInstanceScoped(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_STATE_HOME", t.TempDir())

	version := "8.0"
	instance := "affiliate-platform"
	tests := map[string]string{
		"unit":   "routa-mysql@8.0_affiliate-platform.service",
		"config": filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "routa", "services", "mysql", version, "instances", instance, "my.cnf"),
		"data":   filepath.Join(os.Getenv("XDG_DATA_HOME"), "routa", "services", "mysql", version, "instances", instance),
		"socket": filepath.Join(os.Getenv("XDG_STATE_HOME"), "routa", "run", "mysql-"+version+"_"+instance+".sock"),
		"pid":    filepath.Join(os.Getenv("XDG_STATE_HOME"), "routa", "run", "mysql-"+version+"_"+instance+".pid"),
		"log":    filepath.Join(os.Getenv("XDG_STATE_HOME"), "routa", "log", "mysql-"+version+"_"+instance+".log"),
	}

	got := map[string]string{
		"unit":   MySQLUnitNameForInstance(version, instance),
		"config": MySQLConfigPathForInstance(version, instance),
		"data":   MySQLDataDirForInstance(version, instance),
		"socket": MySQLSocketPathForInstance(version, instance),
		"pid":    MySQLPIDFileForInstance(version, instance),
		"log":    MySQLLogPathForInstance(version, instance),
	}
	for name, want := range tests {
		if got[name] != want {
			t.Fatalf("%s path = %q, want %q", name, got[name], want)
		}
	}
}

func TestFindMySQLBinaryRejectsMariaDBMysqld(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	lookPath := func(name string) (string, error) {
		if name == "mysqld" {
			return "/usr/bin/mysqld", nil
		}
		return "", errors.New("not found")
	}
	output := func(name string, args ...string) ([]byte, error) {
		return []byte("mysqld  Ver 12.2.2-MariaDB for Linux on x86_64\n"), nil
	}

	_, err := findMySQLBinaryWith("8.0", lookPath, output)
	if err == nil {
		t.Fatal("expected MariaDB mysqld to be rejected")
	}
	if !strings.Contains(err.Error(), "MariaDB") {
		t.Fatalf("error should mention MariaDB mismatch: %v", err)
	}
}

func TestFindMySQLBinaryAcceptsOracleMySQL(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	lookPath := func(name string) (string, error) {
		switch name {
		case "mysqld-8.0":
			return "/opt/mysql/8.0/bin/mysqld", nil
		default:
			return "", errors.New("not found")
		}
	}
	output := func(name string, args ...string) ([]byte, error) {
		return []byte("/opt/mysql/8.0/bin/mysqld  Ver 8.0.37 for Linux on x86_64 (MySQL Community Server - GPL)\n"), nil
	}

	got, err := findMySQLBinaryWith("8.0", lookPath, output)
	if err != nil {
		t.Fatal(err)
	}
	if got != "/opt/mysql/8.0/bin/mysqld" {
		t.Fatalf("binary = %q", got)
	}
}

func TestFindMySQLBinaryReportsMissingSharedLibrary(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	lookPath := func(name string) (string, error) {
		if name == "mysqld" {
			return "/opt/mysql/8.0/bin/mysqld", nil
		}
		return "", errors.New("not found")
	}
	output := func(name string, args ...string) ([]byte, error) {
		return []byte("/opt/mysql/8.0/bin/mysqld: error while loading shared libraries: libaio.so.1: cannot open shared object file: No such file or directory\n"), errors.New("exit 127")
	}

	_, err := findMySQLBinaryWith("8.0", lookPath, output)
	if err == nil {
		t.Fatal("expected missing shared library error")
	}
	if !strings.Contains(err.Error(), "missing shared library libaio.so.1") {
		t.Fatalf("error should mention missing shared library: %v", err)
	}
}

func TestFindManagedMySQLBinaryReturnsRuntimeDependencyError(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	bin := ManagedMySQLBinaryPath("8.0")
	if err := os.MkdirAll(filepath.Dir(bin), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(bin, []byte(""), 0o755); err != nil {
		t.Fatal(err)
	}
	lookPath := func(name string) (string, error) {
		return "", errors.New("not found")
	}
	output := func(name string, args ...string) ([]byte, error) {
		if name == "ldd" {
			return []byte("\tlibaio.so.1 => not found\n\tlibnuma.so.1 => not found\n"), nil
		}
		return []byte(bin + ": error while loading shared libraries: libaio.so.1: cannot open shared object file: No such file or directory\n"), errors.New("exit 127")
	}

	_, err := findMySQLBinaryWith("8.0", lookPath, output)
	if err == nil {
		t.Fatal("expected runtime dependency error")
	}
	depErr, ok := err.(*RuntimeDependencyError)
	if !ok {
		t.Fatalf("error type = %T, want *RuntimeDependencyError: %v", err, err)
	}
	if depErr.Binary != bin {
		t.Fatalf("binary = %q, want %q", depErr.Binary, bin)
	}
	if len(depErr.Libraries) != 2 || depErr.Libraries[0] != "libaio.so.1" || depErr.Libraries[1] != "libnuma.so.1" {
		t.Fatalf("libraries = %#v", depErr.Libraries)
	}
}

func TestWriteAndReadMySQLCredentials(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	creds := MySQLCredentials{User: "affiliate", Password: "secret"}
	if err := WriteMySQLCredentialsForInstance("8.0", "affiliate", creds); err != nil {
		t.Fatal(err)
	}
	got, ok, err := ReadMySQLCredentialsForInstance("8.0", "affiliate")
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected saved credentials")
	}
	if got != creds {
		t.Fatalf("credentials = %#v, want %#v", got, creds)
	}
	info, err := os.Stat(MySQLCredentialsPathForInstance("8.0", "affiliate"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("credentials file mode = %o, want 600", info.Mode().Perm())
	}
}

func TestValidateMySQLCredentialsRejectsRoot(t *testing.T) {
	err := ValidateMySQLCredentials(MySQLCredentials{User: "root", Password: "secret"})
	if err == nil {
		t.Fatal("expected root to be rejected")
	}
	if !strings.Contains(err.Error(), "root") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestMySQLCredentialsSQLEscapesPassword(t *testing.T) {
	sql := mysqlCredentialsSQL(MySQLCredentials{User: "app", Password: "pa'ss"})
	for _, want := range []string{
		"CREATE USER IF NOT EXISTS 'app'@'127.0.0.1' IDENTIFIED BY 'pa''ss'",
		"ALTER USER 'app'@'localhost' IDENTIFIED BY 'pa''ss'",
		"GRANT ALL PRIVILEGES ON *.* TO 'app'@'127.0.0.1'",
	} {
		if !strings.Contains(sql, want) {
			t.Fatalf("sql missing %q:\n%s", want, sql)
		}
	}
}
