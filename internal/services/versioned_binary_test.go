package services

import (
	"fmt"
	"testing"
)

func TestVersionLabelMatches(t *testing.T) {
	tests := []struct {
		actual    string
		requested string
		want      bool
	}{
		{actual: "16.3", requested: "16", want: true},
		{actual: "16.3", requested: "16.3", want: true},
		{actual: "16.3.1", requested: "16.3", want: true},
		{actual: "16.3", requested: "1", want: false},
		{actual: "12.2.2", requested: "11.4", want: false},
	}

	for _, tt := range tests {
		got := versionLabelMatches(tt.actual, tt.requested)
		if got != tt.want {
			t.Fatalf("versionLabelMatches(%q, %q) = %v, want %v", tt.actual, tt.requested, got, tt.want)
		}
	}
}

func TestFindPostgresBinaryWithRejectsMismatchedUnversionedBinary(t *testing.T) {
	lookPath := fakeLookPath(map[string]string{"postgres": "/usr/bin/postgres"})
	output := fakeCommandOutput(map[string]string{"/usr/bin/postgres": "postgres (PostgreSQL) 15.7"})

	_, err := findPostgresBinaryWith("16", lookPath, output)
	if err == nil {
		t.Fatal("expected mismatch error")
	}
}

func TestFindPostgresBinaryWithUsesMatchingVersionedBinary(t *testing.T) {
	lookPath := fakeLookPath(map[string]string{
		"postgres-16": "/usr/lib/postgresql/16/bin/postgres",
		"postgres":    "/usr/bin/postgres",
	})
	output := fakeCommandOutput(map[string]string{
		"/usr/lib/postgresql/16/bin/postgres": "postgres (PostgreSQL) 16.3",
		"/usr/bin/postgres":                   "postgres (PostgreSQL) 15.7",
	})

	got, err := findPostgresBinaryWith("16", lookPath, output)
	if err != nil {
		t.Fatal(err)
	}
	if got != "/usr/lib/postgresql/16/bin/postgres" {
		t.Fatalf("binary = %q", got)
	}
}

func TestFindMariaDBBinaryWithRejectsMismatchedUnversionedBinary(t *testing.T) {
	lookPath := fakeLookPath(map[string]string{"mariadbd": "/usr/bin/mariadbd"})
	output := fakeCommandOutput(map[string]string{"/usr/bin/mariadbd": "mariadbd  Ver 12.2.2-MariaDB"})

	_, err := findMariaDBBinaryWith("11.4", lookPath, output)
	if err == nil {
		t.Fatal("expected mismatch error")
	}
}

func TestFindMariaDBBinaryWithUsesMatchingUnversionedBinary(t *testing.T) {
	lookPath := fakeLookPath(map[string]string{"mariadbd": "/usr/bin/mariadbd"})
	output := fakeCommandOutput(map[string]string{"/usr/bin/mariadbd": "2026-05-04 15:02:04 0 [Warning] failed to retrieve the MAC address\nmariadbd  Ver 12.2.2-MariaDB"})

	got, err := findMariaDBBinaryWith("12.2", lookPath, output)
	if err != nil {
		t.Fatal(err)
	}
	if got != "/usr/bin/mariadbd" {
		t.Fatalf("binary = %q", got)
	}
}

func fakeLookPath(paths map[string]string) lookPathFunc {
	return func(name string) (string, error) {
		if path, ok := paths[name]; ok {
			return path, nil
		}
		return "", fmt.Errorf("%s not found", name)
	}
}

func fakeCommandOutput(outputs map[string]string) commandOutputFunc {
	return func(name string, _ ...string) ([]byte, error) {
		if output, ok := outputs[name]; ok {
			return []byte(output), nil
		}
		return nil, fmt.Errorf("%s output not configured", name)
	}
}
