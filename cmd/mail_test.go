package cmd

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/scottzirkel/routa/internal/paths"
	"github.com/scottzirkel/routa/internal/services"
)

func TestMailpitProxyLinkDefaultsToMailTest(t *testing.T) {
	link, err := mailpitProxyLink(nil)
	if err != nil {
		t.Fatal(err)
	}

	if link.Name != "mail" || link.Target != services.MailpitWebAddr() || !link.Secure {
		t.Fatalf("link = %#v", link)
	}
}

func TestMailpitProxyLinkAcceptsCustomName(t *testing.T) {
	link, err := mailpitProxyLink([]string{"inbox.test"})
	if err != nil {
		t.Fatal(err)
	}

	if link.Name != "inbox" || link.Target != services.MailpitWebAddr() || !link.Secure {
		t.Fatalf("link = %#v", link)
	}
}

func TestMailpitProxyLinkUsesConfiguredWebPort(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	unit := `[Service]
ExecStart=/usr/bin/mailpit --listen 127.0.0.1:8026 --smtp 127.0.0.1:1026 --database /tmp/mailpit.db
`
	writeFile(t, filepath.Join(paths.SystemdUserDir(), services.MailpitUnitName), unit)

	link, err := mailpitProxyLink([]string{"inbox.test"})
	if err != nil {
		t.Fatal(err)
	}
	if link.Name != "inbox" || link.Target != "127.0.0.1:8026" || !link.Secure {
		t.Fatalf("link = %#v", link)
	}
}

func TestMailpitPortsFromCommandAcceptsOnAlias(t *testing.T) {
	webPort, smtpPort, err := mailpitPortsFromCommand(mailStartCmd, []string{"on", "8026"}, "", "1026")
	if err != nil {
		t.Fatal(err)
	}
	if webPort != "8026" || smtpPort != "1026" {
		t.Fatalf("ports = %q, %q", webPort, smtpPort)
	}
}

func TestMailpitConfiguredPortsReadCustomUnit(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	unit := `[Service]
ExecStart=/usr/bin/mailpit --listen 127.0.0.1:8026 --smtp 127.0.0.1:1026 --database /tmp/mailpit.db
`
	writeFile(t, filepath.Join(paths.SystemdUserDir(), services.MailpitUnitName), unit)

	webPort, smtpPort, err := mailpitConfiguredPorts()
	if err != nil {
		t.Fatal(err)
	}
	if webPort != "8026" || smtpPort != "1026" {
		t.Fatalf("ports = %q, %q", webPort, smtpPort)
	}
}

func TestMailpitStatusHeaderIncludesWebAndSMTPAddrs(t *testing.T) {
	got := mailpitStatusHeader("8026", "1026")
	for _, want := range []string{
		services.MailpitUnitName,
		"web 127.0.0.1:8026",
		"SMTP 127.0.0.1:1026",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("header missing %q: %s", want, got)
		}
	}
}
