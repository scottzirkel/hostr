package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/scottzirkel/routa/internal/services"
)

func TestSearchEngineVersionArgs(t *testing.T) {
	for _, args := range [][]string{
		{"meilisearch", "1.12"},
		{"typesense", "28"},
	} {
		if err := searchEngineVersionArgs(searchInstallCmd, args); err != nil {
			t.Fatalf("args %#v: %v", args, err)
		}
	}

	for _, args := range [][]string{
		{"elastic", "8"},
		{"meilisearch", "../1.12"},
		{"typesense"},
	} {
		if err := searchEngineVersionArgs(searchInstallCmd, args); err == nil {
			t.Fatalf("expected error for args %#v", args)
		}
	}
}

func TestSearchPortFromCommandAcceptsOnAlias(t *testing.T) {
	got, err := searchPortFromCommand(searchStartCmd, []string{"on", "7710"}, "meilisearch", "")
	if err != nil {
		t.Fatal(err)
	}
	if got != "7710" {
		t.Fatalf("port = %q", got)
	}
}

func TestSearchListShowsInstances(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	if err := os.MkdirAll(services.MeilisearchDataDir("1.12"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(services.TypesenseDataDir("28"), 0o755); err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	searchListCmd.SetOut(&out)
	searchListCmd.SetErr(&bytes.Buffer{})
	t.Cleanup(func() {
		searchListCmd.SetOut(os.Stdout)
		searchListCmd.SetErr(os.Stderr)
	})

	if err := searchListCmd.RunE(searchListCmd, nil); err != nil {
		t.Fatal(err)
	}

	body := out.String()
	for _, want := range []string{
		"ENGINE",
		"meilisearch",
		"1.12",
		services.MeilisearchDataDir("1.12"),
		"typesense",
		"28",
		services.TypesenseDataDir("28"),
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("search list output missing %q:\n%s", want, body)
		}
	}
}

func TestStorageMinIOVersionArgs(t *testing.T) {
	if err := storageMinIOVersionArgs(storageInstallCmd, []string{"minio", "RELEASE.2026-05-01T00-00-00Z"}); err != nil {
		t.Fatal(err)
	}

	for _, args := range [][]string{
		{"s3", "1"},
		{"minio", "../RELEASE"},
		{"minio"},
	} {
		if err := storageMinIOVersionArgs(storageInstallCmd, args); err == nil {
			t.Fatalf("expected error for args %#v", args)
		}
	}
}

func TestMinIOPortsFromCommandAcceptsOnAlias(t *testing.T) {
	port, consolePort, err := minIOPortsFromCommand(storageStartCmd, []string{"on", "9010"}, "", "9011")
	if err != nil {
		t.Fatal(err)
	}
	if port != "9010" || consolePort != "9011" {
		t.Fatalf("ports = %q, %q", port, consolePort)
	}
}

func TestStorageListShowsMinIOInstances(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	version := "RELEASE.2026-05-01T00-00-00Z"
	if err := os.MkdirAll(services.MinIODataDir(version), 0o755); err != nil {
		t.Fatal(err)
	}
	unitPath := filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "systemd", "user", services.MinIOUnitName(version))
	if err := os.MkdirAll(filepath.Dir(unitPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(unitPath, []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	storageListCmd.SetOut(&out)
	storageListCmd.SetErr(&bytes.Buffer{})
	t.Cleanup(func() {
		storageListCmd.SetOut(os.Stdout)
		storageListCmd.SetErr(os.Stderr)
	})

	if err := storageListCmd.RunE(storageListCmd, nil); err != nil {
		t.Fatal(err)
	}

	body := out.String()
	for _, want := range []string{
		"ENGINE",
		"minio",
		version,
		services.MinIOUnitName(version),
		services.MinIODataDir(version),
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("storage list output missing %q:\n%s", want, body)
		}
	}
}
