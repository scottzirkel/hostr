package cmd

import "testing"

func TestCutoverSystemSideAppliedRequiresAllMarkers(t *testing.T) {
	tests := []struct {
		name       string
		resolvDone bool
		perLink    bool
		sysctl     bool
		want       bool
	}{
		{name: "all applied", resolvDone: true, perLink: true, sysctl: true, want: true},
		{name: "missing resolver", perLink: true, sysctl: true},
		{name: "missing per-link routing", resolvDone: true, sysctl: true},
		{name: "missing sysctl", resolvDone: true, perLink: true},
		{name: "none applied"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cutoverSystemSideApplied(tt.resolvDone, tt.perLink, tt.sysctl)
			if got != tt.want {
				t.Fatalf("cutoverSystemSideApplied() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRollbackSystemSideStillAppliedDetectsAnyMarker(t *testing.T) {
	tests := []struct {
		name          string
		resolvSystemd bool
		perLink       bool
		sysctl        bool
		want          bool
	}{
		{name: "resolver still applied", resolvSystemd: true, want: true},
		{name: "per-link still applied", perLink: true, want: true},
		{name: "sysctl still applied", sysctl: true, want: true},
		{name: "all reverted"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := rollbackSystemSideStillApplied(tt.resolvSystemd, tt.perLink, tt.sysctl)
			if got != tt.want {
				t.Fatalf("rollbackSystemSideStillApplied() = %v, want %v", got, tt.want)
			}
		})
	}
}
