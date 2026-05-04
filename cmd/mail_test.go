package cmd

import (
	"testing"

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
