package migrate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadConfig(t *testing.T) {
	home := fakeHome(t)
	writeValetFile(t, "config.json", `{"paths":["/code"],"domain":"test"}`)

	cfg, err := ReadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Paths) != 1 || cfg.Paths[0] != "/code" {
		t.Fatalf("paths = %#v", cfg.Paths)
	}
	if cfg.Domain != "test" {
		t.Fatalf("domain = %q", cfg.Domain)
	}
	if !strings.HasPrefix(valetPath("config.json"), filepath.Join(home, ".valet")) {
		t.Fatalf("valetPath did not use fake HOME: %s", valetPath("config.json"))
	}
}

func TestBuildPlanImportsLinkedSites(t *testing.T) {
	fakeHome(t)
	app := filepath.Join(t.TempDir(), "app")
	api := filepath.Join(t.TempDir(), "api")
	for _, dir := range []string{app, api} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(valetPath("Sites"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(app, valetPath("Sites", "app")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(api, valetPath("Sites", "api")); err != nil {
		t.Fatal(err)
	}
	writeValetFile(t, filepath.Join("Sites", "not-a-link"), "")
	writeValetFile(t, filepath.Join("Certificates", "app.test.crt"), "cert")
	writeValetFile(t, filepath.Join("Nginx", "app.test"), `
server {
    root "`+filepath.Join(app, "public")+`";
    fastcgi_pass unix:/home/scott/.config/valet/valet84.sock;
}`)
	writeValetFile(t, filepath.Join("Nginx", "api.test"), `
server {
    fastcgi_pass unix:/home/scott/.config/valet/valet8.3.sock;
}`)

	plan, err := BuildPlan(&ValetConfig{
		Paths:  []string{"/code"},
		Domain: "local",
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(plan.Parked) != 1 || plan.Parked[0] != "/code" {
		t.Fatalf("parked = %#v", plan.Parked)
	}
	if len(plan.Links) != 2 {
		t.Fatalf("links = %#v", plan.Links)
	}
	appLink := plan.Links[0]
	apiLink := plan.Links[1]
	if appLink.Name != "app" {
		appLink, apiLink = apiLink, appLink
	}

	if appLink.Name != "app" || appLink.Path != app || !appLink.Secure || appLink.PHP != "8.4" || appLink.Root != "public" {
		t.Fatalf("app link = %#v", appLink)
	}
	if apiLink.Name != "api" || apiLink.Path != api || apiLink.Secure || apiLink.PHP != "8.3" || apiLink.Root != "" {
		t.Fatalf("api link = %#v", apiLink)
	}
	if len(plan.Warnings) != 1 || !strings.Contains(plan.Warnings[0], `valet domain is "local"`) {
		t.Fatalf("warnings = %#v", plan.Warnings)
	}
}

func TestBuildPlanToleratesMissingSitesDirectory(t *testing.T) {
	fakeHome(t)

	plan, err := BuildPlan(&ValetConfig{Paths: []string{"/code"}, Domain: "test"})
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Parked) != 1 || plan.Parked[0] != "/code" {
		t.Fatalf("parked = %#v", plan.Parked)
	}
	if len(plan.Links) != 0 {
		t.Fatalf("links = %#v", plan.Links)
	}
}

func TestNginxRootKeepsExternalRootsAbsolute(t *testing.T) {
	fakeHome(t)
	sitePath := filepath.Join(t.TempDir(), "app")
	externalRoot := filepath.Join(t.TempDir(), "public")
	writeValetFile(t, filepath.Join("Nginx", "app.test"), "root "+externalRoot+";")

	got := nginxRoot("app", sitePath)
	if got != externalRoot {
		t.Fatalf("nginxRoot() = %q, want %q", got, externalRoot)
	}
}

func fakeHome(t *testing.T) string {
	t.Helper()

	home := t.TempDir()
	t.Setenv("HOME", home)
	return home
}

func writeValetFile(t *testing.T, rel, content string) {
	t.Helper()

	path := valetPath(rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
