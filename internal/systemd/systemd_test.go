package systemd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderUserUnitFiles(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	routaBin := filepath.Join(t.TempDir(), "routa")
	units, err := RenderUserUnitFiles(1053, routaBin)
	if err != nil {
		t.Fatal(err)
	}
	if len(units) != 2 {
		t.Fatalf("got %d units, want 2: %#v", len(units), units)
	}

	byName := map[string]string{}
	for _, unit := range units {
		byName[unit.Name] = unit.Content
	}

	dns := byName["routa-dns.service"]
	for _, want := range []string{
		"Description=routa DNS responder for *.test",
		"ExecStart=" + routaBin + " serve-dns --addr 127.0.0.1:1053",
		"WantedBy=default.target",
	} {
		if !strings.Contains(dns, want) {
			t.Fatalf("DNS unit missing %q:\n%s", want, dns)
		}
	}

	caddy := byName["routa-caddy.service"]
	for _, want := range []string{
		"Description=routa Caddy reverse proxy",
		"After=network.target routa-dns.service",
		"ExecStart=/usr/bin/caddy run --config " + filepath.Join(os.Getenv("XDG_DATA_HOME"), "routa", "Caddyfile") + " --adapter caddyfile",
		"ExecReload=/usr/bin/caddy reload --config " + filepath.Join(os.Getenv("XDG_DATA_HOME"), "routa", "Caddyfile") + " --adapter caddyfile --force",
		"LimitNOFILE=1048576",
	} {
		if !strings.Contains(caddy, want) {
			t.Fatalf("Caddy unit missing %q:\n%s", want, caddy)
		}
	}
}
