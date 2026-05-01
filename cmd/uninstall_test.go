package cmd

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestPHPUnitsForUninstallDiscoversEnabledAndRuntimeInstances(t *testing.T) {
	systemdDir := t.TempDir()
	runDir := t.TempDir()

	wantsDir := filepath.Join(systemdDir, "default.target.wants")
	if err := os.MkdirAll(wantsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{
		filepath.Join(systemdDir, "routa-php@.service"),
		filepath.Join(wantsDir, "routa-php@8.4.service"),
		filepath.Join(runDir, "php-fpm-8.3.conf"),
		filepath.Join(runDir, "php-fpm-8.3.sock"),
	} {
		if err := os.WriteFile(path, []byte(""), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	got := phpUnitsForUninstall(systemdDir, runDir)
	want := []string{"routa-php@8.3.service", "routa-php@8.4.service"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("phpUnitsForUninstall() = %#v, want %#v", got, want)
	}
}

func TestPHPUnitsForUninstallIgnoresTemplatesAndMalformedRuntimeFiles(t *testing.T) {
	systemdDir := t.TempDir()
	runDir := t.TempDir()

	for _, path := range []string{
		filepath.Join(systemdDir, "routa-php@.service"),
		filepath.Join(systemdDir, "routa-php@.service.bak"),
		filepath.Join(runDir, "php-fpm-.conf"),
		filepath.Join(runDir, "php-fpm-8.4.log"),
		filepath.Join(runDir, "php-fpm-8.4.conf"),
	} {
		if err := os.WriteFile(path, []byte(""), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	got := phpUnitsForUninstall(systemdDir, runDir)
	want := []string{"routa-php@8.4.service"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("phpUnitsForUninstall() = %#v, want %#v", got, want)
	}
}

func TestRoutaUnitsForUninstallIncludesDiscoveredPHPUnits(t *testing.T) {
	configHome := t.TempDir()
	stateHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)
	t.Setenv("XDG_STATE_HOME", stateHome)

	if err := os.MkdirAll(filepath.Join(configHome, "systemd", "user", "default.target.wants"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(stateHome, "routa", "run"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{
		filepath.Join(configHome, "systemd", "user", "default.target.wants", "routa-php@8.3.service"),
		filepath.Join(stateHome, "routa", "run", "php-fpm-8.4.sock"),
	} {
		if err := os.WriteFile(path, []byte(""), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	got := routaUnitsForUninstall()
	want := []string{"routa-caddy.service", "routa-dns.service", "routa-php@8.3.service", "routa-php@8.4.service"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("routaUnitsForUninstall() = %#v, want %#v", got, want)
	}
}

func TestRoutaDirsForPurgeUsesXDGRoutaDirs(t *testing.T) {
	configHome := t.TempDir()
	dataHome := t.TempDir()
	stateHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)
	t.Setenv("XDG_DATA_HOME", dataHome)
	t.Setenv("XDG_STATE_HOME", stateHome)

	got := routaDirsForPurge()
	want := []string{
		filepath.Join(dataHome, "routa"),
		filepath.Join(stateHome, "routa"),
		filepath.Join(configHome, "routa"),
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("routaDirsForPurge() = %#v, want %#v", got, want)
	}
}

func TestPurgeRoutaDirRemovesOnlyRoutaNamedDirectory(t *testing.T) {
	root := t.TempDir()
	routaDir := filepath.Join(root, "routa")
	keepDir := filepath.Join(root, "not-routa")
	if err := os.MkdirAll(filepath.Join(routaDir, "nested"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(keepDir, 0o755); err != nil {
		t.Fatal(err)
	}

	if err := purgeRoutaDir(routaDir); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(routaDir); !os.IsNotExist(err) {
		t.Fatalf("routa dir should be removed, stat err = %v", err)
	}
	if _, err := os.Stat(keepDir); err != nil {
		t.Fatalf("unrelated dir should remain: %v", err)
	}

	err := purgeRoutaDir(keepDir)
	if err == nil {
		t.Fatal("expected refusal for non-routa directory")
	}
	if !strings.Contains(err.Error(), "refusing to purge non-routa directory") {
		t.Fatalf("unexpected error: %v", err)
	}
}
