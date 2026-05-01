package dev

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestDetectNodeUsesPackageManagerLockfile(t *testing.T) {
	dir := t.TempDir()
	write(t, filepath.Join(dir, "package.json"), `{"scripts":{"dev":"vite --host 127.0.0.1"}}`)
	write(t, filepath.Join(dir, "pnpm-lock.yaml"), "")

	spec, err := Detect(dir)
	if err != nil {
		t.Fatal(err)
	}
	if spec.Kind != "node" || spec.DefaultPort != 5173 {
		t.Fatalf("spec = %#v", spec)
	}
	if want := []string{"pnpm", "run", "dev"}; !reflect.DeepEqual(spec.Command, want) {
		t.Fatalf("command = %#v, want %#v", spec.Command, want)
	}
}

func TestDetectRailsPrefersBinDev(t *testing.T) {
	dir := t.TempDir()
	write(t, filepath.Join(dir, "Gemfile"), `gem "rails"`)
	if err := os.MkdirAll(filepath.Join(dir, "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(dir, "bin", "dev"), "#!/usr/bin/env sh\n")

	spec, err := Detect(dir)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"bin/dev"}; spec.Kind != "rails" || !reflect.DeepEqual(spec.Command, want) || spec.DefaultPort != 3000 {
		t.Fatalf("spec = %#v", spec)
	}
}

func TestDetectPhoenix(t *testing.T) {
	dir := t.TempDir()
	write(t, filepath.Join(dir, "mix.exs"), `{:phoenix, "~> 1.8"}`)

	spec, err := Detect(dir)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"mix", "phx.server"}; spec.Kind != "phoenix" || !reflect.DeepEqual(spec.Command, want) || spec.DefaultPort != 4000 {
		t.Fatalf("spec = %#v", spec)
	}
}

func TestDetectDjango(t *testing.T) {
	dir := t.TempDir()
	write(t, filepath.Join(dir, "manage.py"), "")

	spec, err := Detect(dir)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"python", "manage.py", "runserver", "127.0.0.1:8000"}; spec.Kind != "django" || !reflect.DeepEqual(spec.Command, want) || spec.DefaultPort != 8000 {
		t.Fatalf("spec = %#v", spec)
	}
}

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
