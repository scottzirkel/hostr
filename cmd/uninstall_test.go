package cmd

import (
	"os"
	"path/filepath"
	"reflect"
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
		filepath.Join(systemdDir, "hostr-php@.service"),
		filepath.Join(wantsDir, "hostr-php@8.4.service"),
		filepath.Join(runDir, "php-fpm-8.3.conf"),
		filepath.Join(runDir, "php-fpm-8.3.sock"),
	} {
		if err := os.WriteFile(path, []byte(""), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	got := phpUnitsForUninstall(systemdDir, runDir)
	want := []string{"hostr-php@8.3.service", "hostr-php@8.4.service"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("phpUnitsForUninstall() = %#v, want %#v", got, want)
	}
}
