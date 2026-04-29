package site

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestWriteFragmentsQuotesPathsAndUsesHTTPForInsecureSites(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_STATE_HOME", t.TempDir())

	docroot := filepath.Join(t.TempDir(), "my project", "public")
	if err := os.MkdirAll(docroot, 0o755); err != nil {
		t.Fatal(err)
	}

	if err := WriteFragments([]Resolved{{
		Name:    "foo",
		Docroot: docroot,
		Kind:    KindStatic,
		Secure:  false,
	}}); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(os.Getenv("XDG_DATA_HOME"), "hostr", "sites", "foo.caddy"))
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	for _, want := range []string{
		"http://foo.test {",
		"root * " + strconv.Quote(docroot),
		"output file " + strconv.Quote(filepath.Join(os.Getenv("XDG_STATE_HOME"), "hostr", "log", "foo.log")),
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("rendered fragment missing %q:\n%s", want, content)
		}
	}
}

func TestWriteFragmentsRendersPHPSiteWithSocket(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_STATE_HOME", t.TempDir())

	docroot := filepath.Join(t.TempDir(), "public")
	if err := os.MkdirAll(docroot, 0o755); err != nil {
		t.Fatal(err)
	}

	if err := WriteFragments([]Resolved{{
		Name:    "app",
		Docroot: docroot,
		Kind:    KindPHP,
		PHP:     "8.4",
		Secure:  true,
	}}); err != nil {
		t.Fatal(err)
	}

	content := readFragment(t, "app")
	for _, want := range []string{
		"app.test {",
		"tls internal",
		"root * " + strconv.Quote(docroot),
		"php_fastcgi " + strconv.Quote("unix/"+filepath.Join(os.Getenv("XDG_STATE_HOME"), "hostr", "run", "php-fpm-8.4.sock")),
		"file_server",
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("rendered fragment missing %q:\n%s", want, content)
		}
	}
}

func TestWriteFragmentsRendersMissingPHPFallback(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_STATE_HOME", t.TempDir())

	if err := WriteFragments([]Resolved{{
		Name:    "app",
		Docroot: t.TempDir(),
		Kind:    KindPHP,
		Secure:  true,
	}}); err != nil {
		t.Fatal(err)
	}

	content := readFragment(t, "app")
	for _, want := range []string{
		"respond \"hostr: app is a PHP site but no PHP version is installed. Run 'hostr php install <ver>'.\" 503",
		"file_server",
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("rendered fragment missing %q:\n%s", want, content)
		}
	}
	if strings.Contains(content, "php_fastcgi") {
		t.Fatalf("missing-PHP fragment should not render php_fastcgi:\n%s", content)
	}
}

func TestWriteFragmentsRendersProxySite(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_STATE_HOME", t.TempDir())

	if err := WriteFragments([]Resolved{{
		Name:   "vite",
		Target: "127.0.0.1:5173",
		Kind:   KindProxy,
		Secure: true,
	}}); err != nil {
		t.Fatal(err)
	}

	content := readFragment(t, "vite")
	for _, want := range []string{
		"vite.test {",
		"tls internal",
		"reverse_proxy " + strconv.Quote("127.0.0.1:5173"),
		"output file " + strconv.Quote(filepath.Join(os.Getenv("XDG_STATE_HOME"), "hostr", "log", "vite.log")),
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("rendered fragment missing %q:\n%s", want, content)
		}
	}
	for _, unwanted := range []string{"root *", "file_server", "php_fastcgi"} {
		if strings.Contains(content, unwanted) {
			t.Fatalf("proxy fragment should not include %q:\n%s", unwanted, content)
		}
	}
}

func TestWriteFragmentsRemovesStaleFragments(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_STATE_HOME", t.TempDir())

	if err := WriteFragments([]Resolved{
		{Name: "old", Docroot: t.TempDir(), Kind: KindStatic, Secure: true},
		{Name: "new", Docroot: t.TempDir(), Kind: KindStatic, Secure: true},
	}); err != nil {
		t.Fatal(err)
	}
	if err := WriteFragments([]Resolved{
		{Name: "new", Docroot: t.TempDir(), Kind: KindStatic, Secure: true},
	}); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(fragmentPath("old")); !os.IsNotExist(err) {
		t.Fatalf("old fragment should be removed, stat err = %v", err)
	}
	if _, err := os.Stat(fragmentPath("new")); err != nil {
		t.Fatalf("new fragment should remain: %v", err)
	}
}

func TestWriteFragmentsRejectsInvalidSiteNames(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	err := WriteFragments([]Resolved{{
		Name:    "bad/name",
		Docroot: t.TempDir(),
		Kind:    KindStatic,
		Secure:  true,
	}})
	if err == nil {
		t.Fatal("expected invalid site name error")
	}
}

func TestResolvePathReturnsLongestMatchingSitePath(t *testing.T) {
	root := t.TempDir()
	parent := filepath.Join(root, "parent")
	child := filepath.Join(parent, "child")
	if err := os.MkdirAll(child, 0o755); err != nil {
		t.Fatal(err)
	}

	state := &State{
		Links: []Link{
			{Name: "parent", Path: parent, Secure: true},
			{Name: "child", Path: child, Secure: true},
			{Name: "child-api", Path: child, Secure: true},
		},
	}

	matches := state.ResolvePath(filepath.Join(child, "app"))
	if len(matches) != 2 {
		t.Fatalf("got %d matches, want 2: %#v", len(matches), matches)
	}
	if matches[0].Name != "child" || matches[1].Name != "child-api" {
		t.Fatalf("unexpected matches: %#v", matches)
	}
}

func readFragment(t *testing.T, name string) string {
	t.Helper()

	data, err := os.ReadFile(fragmentPath(name))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func fragmentPath(name string) string {
	return filepath.Join(os.Getenv("XDG_DATA_HOME"), "hostr", "sites", name+".caddy")
}
