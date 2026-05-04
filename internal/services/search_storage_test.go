package services

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderMeilisearchUnit(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_STATE_HOME", t.TempDir())

	unit, err := RenderMeilisearchUnit("1.12", "/usr/bin/meilisearch")
	if err != nil {
		t.Fatal(err)
	}

	for _, want := range []string{
		"Description=routa Meilisearch 1.12",
		"ExecStart=/usr/bin/meilisearch --env development --http-addr 127.0.0.1:7700 --db-path " + filepath.Join(os.Getenv("XDG_DATA_HOME"), "routa", "services", "meilisearch", "1.12"),
		"StandardOutput=append:" + filepath.Join(os.Getenv("XDG_STATE_HOME"), "routa", "log", "meilisearch-1.12.log"),
		"WantedBy=default.target",
	} {
		if !strings.Contains(unit, want) {
			t.Fatalf("Meilisearch unit missing %q:\n%s", want, unit)
		}
	}
}

func TestRenderTypesenseUnit(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_STATE_HOME", t.TempDir())

	unit, err := RenderTypesenseUnit("28", "/usr/bin/typesense-server")
	if err != nil {
		t.Fatal(err)
	}

	for _, want := range []string{
		"Description=routa Typesense 28",
		"ExecStart=/usr/bin/typesense-server --data-dir " + filepath.Join(os.Getenv("XDG_DATA_HOME"), "routa", "services", "typesense", "28") + " --api-key routa-local-dev --listen-address 127.0.0.1 --api-port 8108 --enable-cors",
		"StandardError=append:" + filepath.Join(os.Getenv("XDG_STATE_HOME"), "routa", "log", "typesense-28.log"),
		"WantedBy=default.target",
	} {
		if !strings.Contains(unit, want) {
			t.Fatalf("Typesense unit missing %q:\n%s", want, unit)
		}
	}
}

func TestRenderMinIOUnitAndConfig(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_STATE_HOME", t.TempDir())

	version := "RELEASE.2026-05-01T00-00-00Z"
	unit, err := RenderMinIOUnit(version, "/usr/bin/minio")
	if err != nil {
		t.Fatal(err)
	}
	config, err := RenderMinIOConfig()
	if err != nil {
		t.Fatal(err)
	}

	for _, want := range []string{
		"Description=routa MinIO " + version,
		"EnvironmentFile=" + filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "routa", "services", "minio", version, "env"),
		"ExecStart=/usr/bin/minio server --address 127.0.0.1:9000 --console-address 127.0.0.1:9001 " + filepath.Join(os.Getenv("XDG_DATA_HOME"), "routa", "services", "minio", version),
		"StandardOutput=append:" + filepath.Join(os.Getenv("XDG_STATE_HOME"), "routa", "log", "minio-"+version+".log"),
		"WantedBy=default.target",
	} {
		if !strings.Contains(unit, want) {
			t.Fatalf("MinIO unit missing %q:\n%s", want, unit)
		}
	}
	for _, want := range []string{
		"MINIO_ROOT_USER=routa",
		"MINIO_ROOT_PASSWORD=routa-local-dev",
	} {
		if !strings.Contains(config, want) {
			t.Fatalf("MinIO config missing %q:\n%s", want, config)
		}
	}
}

func TestVersionedCommandMatchesLiteralReleaseLabel(t *testing.T) {
	lookPath := fakeLookPath(map[string]string{"minio": "/usr/bin/minio"})
	output := fakeCommandOutput(map[string]string{"/usr/bin/minio": "minio version RELEASE.2026-05-01T00-00-00Z"})

	got, err := findMinIOBinaryWith("RELEASE.2026-05-01T00-00-00Z", lookPath, output)
	if err != nil {
		t.Fatal(err)
	}
	if got != "/usr/bin/minio" {
		t.Fatalf("binary = %q", got)
	}
}
