package services

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderRedisUnit(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	unit, err := RenderRedisUnit("/usr/bin/redis-server")
	if err != nil {
		t.Fatal(err)
	}

	for _, want := range []string{
		"Description=routa Redis",
		"ExecStart=/usr/bin/redis-server " + filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "routa", "services", "redis", "redis.conf"),
		"Restart=on-failure",
		"WantedBy=default.target",
	} {
		if !strings.Contains(unit, want) {
			t.Fatalf("Redis unit missing %q:\n%s", want, unit)
		}
	}
}

func TestRenderRedisConfig(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_STATE_HOME", t.TempDir())

	conf, err := RenderRedisConfig()
	if err != nil {
		t.Fatal(err)
	}

	for _, want := range []string{
		"bind 127.0.0.1 ::1",
		"protected-mode yes",
		"port 6379",
		"daemonize no",
		"dir " + filepath.Join(os.Getenv("XDG_DATA_HOME"), "routa", "services", "redis"),
		"appendonly yes",
		"pidfile " + filepath.Join(os.Getenv("XDG_STATE_HOME"), "routa", "run", "redis.pid"),
	} {
		if !strings.Contains(conf, want) {
			t.Fatalf("Redis config missing %q:\n%s", want, conf)
		}
	}
}

func TestEnsureWithBinaryWritesRedisFiles(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_STATE_HOME", t.TempDir())

	def := Redis()
	def.WriteConfig = WriteRedisConfig
	if err := WriteFiles(def, "/usr/bin/redis-server"); err != nil {
		t.Fatal(err)
	}

	for _, path := range []string{
		RedisConfigPath(),
		filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "systemd", "user", RedisUnitName),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected %s: %v", path, err)
		}
	}
}
