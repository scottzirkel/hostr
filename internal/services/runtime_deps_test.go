package services

import (
	"strings"
	"testing"
)

func TestPackagesForLibrariesArch(t *testing.T) {
	got := packagesForLibraries("arch", []string{"libaio.so.1", "libnuma.so.1", "libaio.so.1"})
	want := []string{"libaio", "numactl"}
	if len(got) != len(want) {
		t.Fatalf("packages = %#v, want %#v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("packages = %#v, want %#v", got, want)
		}
	}
}

func TestRuntimeDependencyErrorIncludesInstallCommand(t *testing.T) {
	err := (&RuntimeDependencyError{
		Service:   "MySQL",
		Version:   "8.0",
		Binary:    "/home/scott/.local/share/routa/binaries/mysql/8.0/bin/mysqld",
		Libraries: []string{"libaio.so.1"},
	}).Error()

	for _, want := range []string{
		"MySQL 8.0 is installed under routa",
		"libaio.so.1",
		"Checked binary:",
	} {
		if !strings.Contains(err, want) {
			t.Fatalf("error missing %q:\n%s", want, err)
		}
	}
}
