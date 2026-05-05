package services

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseAndResolveMySQLReleasesPrefersLatestMinimal(t *testing.T) {
	body := `
(mysql-8.0.45-linux-glibc2.28-x86_64.tar.xz)
(mysql-8.0.45-linux-glibc2.17-x86_64-minimal.tar.xz)
(mysql-8.0.46-linux-glibc2.28-x86_64.tar.xz)
(mysql-8.0.46-linux-glibc2.17-x86_64-minimal.tar.xz)
`
	releases := ParseMySQLReleases(body)
	got, err := ResolveMySQLRelease("8.0", releases)
	if err != nil {
		t.Fatal(err)
	}
	if got.Version != "8.0.46" {
		t.Fatalf("version = %q", got.Version)
	}
	if !got.Minimal {
		t.Fatal("expected minimal tarball")
	}
	if !strings.HasSuffix(got.URL, "/mysql-8.0.46-linux-glibc2.17-x86_64-minimal.tar.xz") {
		t.Fatalf("url = %q", got.URL)
	}
}

func TestFindMySQLBinaryPrefersManagedBinary(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	bin := ManagedMySQLBinaryPath("8.0")
	if err := os.MkdirAll(filepath.Dir(bin), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(bin, []byte(""), 0o755); err != nil {
		t.Fatal(err)
	}

	lookPath := func(name string) (string, error) {
		return "", os.ErrNotExist
	}
	output := func(name string, args ...string) ([]byte, error) {
		if name != bin {
			t.Fatalf("unexpected version check for %s", name)
		}
		return []byte("mysqld  Ver 8.0.46 for Linux on x86_64 (MySQL Community Server - GPL)\n"), nil
	}

	got, err := findMySQLBinaryWith("8.0", lookPath, output)
	if err != nil {
		t.Fatal(err)
	}
	if got != bin {
		t.Fatalf("binary = %q, want %q", got, bin)
	}
}
