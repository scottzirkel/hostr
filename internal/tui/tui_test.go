package tui

import (
	"testing"
	"time"

	"github.com/scottzirkel/routa/internal/site"
)

func TestProblemReasons(t *testing.T) {
	s := site.Resolved{Name: "app", Kind: site.KindStatic, Docroot: "/missing"}
	m := model{
		docExists: map[string]bool{"app": false},
		results:   map[string]probeResult{"app": {code: 500, latency: time.Millisecond}},
	}

	got := m.problemReasons(s)
	want := []string{"missing docroot", "HTTP 500"}
	if len(got) != len(want) {
		t.Fatalf("problemReasons() = %#v, want %#v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("problemReasons()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestActionEligibility(t *testing.T) {
	m := model{links: map[string]site.Link{"linked": {Name: "linked"}}}

	if !m.isExplicitLink("linked") {
		t.Fatal("linked site should be eligible for explicit-link actions")
	}
	if m.isExplicitLink("parked") {
		t.Fatal("parked site should not be eligible for explicit-link actions")
	}
	if m.canChangeRoot(site.Resolved{Name: "vite", Kind: site.KindProxy}) {
		t.Fatal("proxy site should not be eligible for root changes")
	}
}

func TestSortProblemsFirst(t *testing.T) {
	a := site.Resolved{Name: "ok", Kind: site.KindStatic}
	b := site.Resolved{Name: "bad", Kind: site.KindStatic}
	m := model{
		sort:      sortProblems,
		docExists: map[string]bool{"ok": true, "bad": false},
		results:   map[string]probeResult{},
	}

	if !m.siteLess(b, a) {
		t.Fatal("problem site should sort before non-problem site")
	}
}

func TestDisplayItemsCollapseGroup(t *testing.T) {
	m := model{
		sites: []site.Resolved{
			{Name: "app", Kind: site.KindStatic},
			{Name: "api.app", Kind: site.KindStatic},
		},
		docExists: map[string]bool{"app": true, "api.app": true},
		results:   map[string]probeResult{},
		collapsed: map[string]bool{"app": true},
	}

	items := m.displayItems()
	if len(items) != 1 {
		t.Fatalf("collapsed display item count = %d, want 1", len(items))
	}
	if !items[0].collapsed || items[0].childCount != 1 {
		t.Fatalf("collapsed parent metadata = %#v, want collapsed with one child", items[0])
	}
}
