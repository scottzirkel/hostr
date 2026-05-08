package php

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRemoveVersionRemovesPatchDirectoryAndAliases(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	phpDir := filepath.Join(os.Getenv("XDG_DATA_HOME"), "routa", "php")
	if err := os.MkdirAll(filepath.Join(phpDir, "8.4.1", "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("8.4.1", filepath.Join(phpDir, "8.4")); err != nil {
		t.Fatal(err)
	}

	if err := RemoveVersion("8.4"); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{
		filepath.Join(phpDir, "8.4"),
		filepath.Join(phpDir, "8.4.1"),
	} {
		if _, err := os.Lstat(p); !os.IsNotExist(err) {
			t.Fatalf("%s still exists after removal", p)
		}
	}
}

func TestRemoveVersionResolvesMinorWhenAliasIsAlreadyGone(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	phpDir := filepath.Join(os.Getenv("XDG_DATA_HOME"), "routa", "php")
	if err := os.MkdirAll(filepath.Join(phpDir, "8.3.30", "bin"), 0o755); err != nil {
		t.Fatal(err)
	}

	if err := RemoveVersion("8.3"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(filepath.Join(phpDir, "8.3.30")); !os.IsNotExist(err) {
		t.Fatalf("8.3.30 still exists after removal")
	}
}

func TestRemoveVersionErrorsOnAmbiguousMinor(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	phpDir := filepath.Join(os.Getenv("XDG_DATA_HOME"), "routa", "php")
	for _, version := range []string{"8.3.29", "8.3.30"} {
		if err := os.MkdirAll(filepath.Join(phpDir, version, "bin"), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	if err := RemoveVersion("8.3"); err == nil {
		t.Fatal("expected ambiguous version error")
	}
}

func TestSymlinksSkipsDanglingAliases(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	phpDir := filepath.Join(os.Getenv("XDG_DATA_HOME"), "routa", "php")
	if err := os.MkdirAll(phpDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("8.4.1", filepath.Join(phpDir, "8.4")); err != nil {
		t.Fatal(err)
	}

	links, err := Symlinks()
	if err != nil {
		t.Fatal(err)
	}
	if len(links) != 0 {
		t.Fatalf("expected no valid links, got %#v", links)
	}
}

func TestDownloadAndExtractRetriesInterruptedBody(t *testing.T) {
	archive := testTarGz(t, "php binary")
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		attempts++
		w.Header().Set("Content-Type", "application/gzip")
		if attempts == 1 {
			w.Header().Set("Content-Length", "999999")
			_, _ = w.Write(archive[:len(archive)/2])
			return
		}
		w.Header().Set("Content-Length", fmt.Sprint(len(archive)))
		_, _ = w.Write(archive)
	}))
	defer server.Close()

	dest := filepath.Join(t.TempDir(), "php")
	var out bytes.Buffer
	if err := downloadAndExtract(context.Background(), server.URL, dest, &out); err != nil {
		t.Fatal(err)
	}
	if attempts != 2 {
		t.Fatalf("attempts = %d, want 2", attempts)
	}
	data, err := os.ReadFile(dest)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "php binary" {
		t.Fatalf("dest = %q", data)
	}
	if !strings.Contains(out.String(), "retrying download (2/3)") {
		t.Fatalf("retry output missing:\n%s", out.String())
	}
}

func TestDownloadAndExtractDoesNotRetryNotFound(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		attempts++
		http.NotFound(w, nil)
	}))
	defer server.Close()

	err := downloadAndExtract(context.Background(), server.URL, filepath.Join(t.TempDir(), "php"), io.Discard)
	if err == nil {
		t.Fatal("expected error")
	}
	if attempts != 1 {
		t.Fatalf("attempts = %d, want 1", attempts)
	}
}

func TestInstallXdebugDownloadsSharedExtension(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	archive := testTarGz(t, "xdebug so")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "routa_php_xdebug_8.4.20_linux_") {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/gzip")
		_, _ = w.Write(archive)
	}))
	defer server.Close()
	t.Setenv("ROUTA_PHP_XDEBUG_BASE_URL", server.URL)

	var out bytes.Buffer
	ok, err := InstallXdebug(context.Background(), "8.4.20", &out)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected Xdebug to be installed")
	}
	data, err := os.ReadFile(XdebugExtensionPath("8.4.20"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "xdebug so" {
		t.Fatalf("xdebug.so = %q", data)
	}
	if !strings.Contains(out.String(), "xdebug   8.4.20") {
		t.Fatalf("install output missing xdebug download:\n%s", out.String())
	}
}

func TestInstallXdebugSkipsMissingArtifact(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	server := httptest.NewServer(http.NotFoundHandler())
	defer server.Close()
	t.Setenv("ROUTA_PHP_XDEBUG_BASE_URL", server.URL)

	ok, err := InstallXdebug(context.Background(), "8.4.20", io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("expected missing artifact to be skipped")
	}
	if _, err := os.Stat(XdebugExtensionPath("8.4.20")); !os.IsNotExist(err) {
		t.Fatalf("xdebug.so exists after missing artifact: %v", err)
	}
}

func testTarGz(t *testing.T, content string) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	body := []byte(content)
	if err := tw.WriteHeader(&tar.Header{Name: "php", Mode: 0o755, Size: int64(len(body))}); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write(body); err != nil {
		t.Fatal(err)
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}
