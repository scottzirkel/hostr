package cutover

import (
	"strings"
	"testing"
)

func TestDetectPhase(t *testing.T) {
	tests := []struct {
		name       string
		resolvOK   bool
		perLink    bool
		caddyOnStd bool
		caddyOnAlt bool
		want       Phase
	}{
		{
			name:       "phase one on alt ports",
			caddyOnAlt: true,
			want:       PhaseOne,
		},
		{
			name:       "phase one can already use systemd resolved",
			resolvOK:   true,
			caddyOnAlt: true,
			want:       PhaseOne,
		},
		{
			name:       "phase two after cutover",
			resolvOK:   true,
			perLink:    true,
			caddyOnStd: true,
			want:       PhaseTwo,
		},
		{
			name:       "phase two tolerates stale alt listener",
			resolvOK:   true,
			perLink:    true,
			caddyOnStd: true,
			caddyOnAlt: true,
			want:       PhaseTwo,
		},
		{
			name:     "missing caddy listener is partial",
			resolvOK: true,
			perLink:  true,
			want:     PhasePartial,
		},
		{
			name:       "resolver changed without per-link routing is partial",
			resolvOK:   true,
			caddyOnStd: true,
			want:       PhasePartial,
		},
		{
			name:       "per-link routing without resolver stub is partial",
			perLink:    true,
			caddyOnStd: true,
			want:       PhasePartial,
		},
		{
			name: "nothing running is partial",
			want: PhasePartial,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detectPhase(tt.resolvOK, tt.perLink, tt.caddyOnStd, tt.caddyOnAlt)
			if got != tt.want {
				t.Fatalf("detectPhase() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSudoBlockConfiguresPerLinkRouting(t *testing.T) {
	block := SudoBlock()
	for _, want := range []string{
		"found_network=0",
		"hostr cutover needs at least one /etc/systemd/network/*.network file",
		"/etc/systemd/network/*.network",
		"DNS=127.0.0.1:1053",
		"Domains=~test",
		"systemctl restart systemd-resolved.service",
	} {
		if !strings.Contains(block, want) {
			t.Fatalf("sudo block missing %q:\n%s", want, block)
		}
	}
}

func TestSudoBlockChecksNetworkFilesBeforeMutatingSystem(t *testing.T) {
	block := SudoBlock()
	guard := strings.Index(block, "hostr cutover needs at least one /etc/systemd/network/*.network file")
	sysctl := strings.Index(block, "echo 'net.ipv4.ip_unprivileged_port_start=80'")
	resolv := strings.Index(block, "rm -f /etc/resolv.conf")
	if guard == -1 || sysctl == -1 || resolv == -1 {
		t.Fatalf("sudo block missing expected guard or mutations:\n%s", block)
	}
	if !(guard < sysctl && sysctl < resolv) {
		t.Fatalf("sudo block should check network files before sysctl and resolver changes:\n%s", block)
	}
}

func TestRollbackBlockRemovesHostrRouting(t *testing.T) {
	block := SudoRollbackBlock()
	for _, want := range []string{
		`rm -f "$d/hostr.conf"`,
		"rm -f /etc/systemd/resolved.conf.d/hostr.conf",
		"if [ -f /opt/valet-linux/resolv.conf ]; then",
		"ln -sf /opt/valet-linux/resolv.conf /etc/resolv.conf",
		"ln -sf /run/systemd/resolve/stub-resolv.conf /etc/resolv.conf",
	} {
		if !strings.Contains(block, want) {
			t.Fatalf("rollback block missing %q:\n%s", want, block)
		}
	}
}

func TestRollbackBlockRestoresResolverBeforeSysctl(t *testing.T) {
	block := SudoRollbackBlock()
	resolv := strings.Index(block, "rm -f /etc/resolv.conf")
	sysctl := strings.Index(block, "rm -f /etc/sysctl.d/50-hostr.conf")
	if resolv == -1 || sysctl == -1 {
		t.Fatalf("rollback block missing expected resolver or sysctl restoration:\n%s", block)
	}
	if resolv > sysctl {
		t.Fatalf("rollback block should restore resolver before sysctl cleanup:\n%s", block)
	}
}
