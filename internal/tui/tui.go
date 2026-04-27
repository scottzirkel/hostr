// Package tui is hostr's Bubble Tea dashboard.
// Rendered with lipgloss directly because bubbles/table v1 mishandles ANSI
// escapes inside cell text.
package tui

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/scottzirkel/hostr/internal/site"
)

// Column widths flex with the terminal. NAME, HTTPS, STAT always show.
// Remaining columns turn on in priority order as width allows: PHP, DOCROOT,
// LAT, KIND. Extra width goes 60/40 to DOCROOT/NAME (paths tend to be long).
const (
	cursorCol = 2 // "❯ " or "  " prefix on every body row
	minName   = 20
)

type layout struct {
	nameW, httpsW, kindW, phpW, statW, latW, docW int
	showKind, showPHP, showLat, showDoc           bool
}

func computeLayout(termWidth int) layout {
	l := layout{
		nameW:  minName,
		httpsW: 7,
		statW:  6,
	}
	usable := termWidth - cursorCol
	used := l.nameW + l.httpsW + l.statW

	// Add optional columns in priority order, each gated on having room
	// for ITSELF *plus* the minimum docroot we still hope to keep.
	if usable-used >= 6 {
		l.showPHP = true
		l.phpW = 6
		used += l.phpW
	}
	if usable-used >= 12 {
		l.showDoc = true
		l.docW = 12
		used += l.docW
	}
	if usable-used >= 8 {
		l.showLat = true
		l.latW = 8
		used += l.latW
	}
	if usable-used >= 8 {
		l.showKind = true
		l.kindW = 8
		used += l.kindW
	}

	if extra := usable - used; extra > 0 {
		if l.showDoc {
			docAdd := extra * 6 / 10
			l.nameW += extra - docAdd
			l.docW += docAdd
		} else {
			l.nameW += extra
		}
	}
	return l
}

var (
	bannerStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("141")) // bright purple, no bg
	taglineStyle = lipgloss.NewStyle().Italic(true).Foreground(lipgloss.Color("245"))
	ruleStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("141"))
	headerStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("245")).Underline(true)
	okStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("#04B575"))
	warnStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFB454"))
	errStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5555"))
	dimStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	selectedStyle = lipgloss.NewStyle().Background(lipgloss.Color("237"))
	cursorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4")).Bold(true)
	footerStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	chipStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4")).Bold(true)
	searchStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4"))
)

type probeResult struct {
	code    int
	latency time.Duration
}

type probeMsg struct {
	name string
	res  probeResult
}

type secureFilter int

const (
	secAll secureFilter = iota
	secYes
	secNo
)

func (s secureFilter) next() secureFilter { return (s + 1) % 3 }
func (s secureFilter) label() string {
	return [...]string{"", "secure", "insecure"}[s]
}

type kindFilter int

const (
	kindAll kindFilter = iota
	kindPHP
	kindStatic
)

func (k kindFilter) next() kindFilter { return (k + 1) % 3 }
func (k kindFilter) label() string {
	return [...]string{"", "php", "static"}[k]
}

type codeFilter int

const (
	codeAll codeFilter = iota
	code2xx
	code3xx
	code4xx
	code5xx
	codeErr
	codePending
)

func (c codeFilter) next() codeFilter { return (c + 1) % 7 }
func (c codeFilter) label() string {
	return [...]string{"", "2xx", "3xx", "4xx", "5xx", "err", "pending"}[c]
}

type filters struct {
	secure      secureFilter
	kind        kindFilter
	code        codeFilter
	missingOnly bool
	search      string
}

func (f filters) any() bool {
	return f.secure != secAll || f.kind != kindAll || f.code != codeAll || f.missingOnly || f.search != ""
}

type model struct {
	sites     []site.Resolved
	results   map[string]probeResult // missing key = pending
	docExists map[string]bool
	cursor    int
	offset    int
	width     int
	height    int
	filt      filters
	searching bool // true while user is typing into the search box
}

// --- Init / Update -------------------------------------------------------

func (m model) Init() tea.Cmd { return m.probeAll() }

func (m model) probeAll() tea.Cmd {
	cmds := make([]tea.Cmd, 0, len(m.sites))
	for _, s := range m.sites {
		cmds = append(cmds, probeCmd(s.Name))
	}
	return tea.Batch(cmds...)
}

func probeCmd(name string) tea.Cmd {
	return func() tea.Msg {
		c := &http.Client{
			Timeout: 2 * time.Second,
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse
			},
		}
		start := time.Now()
		resp, err := c.Head(siteURL(name))
		elapsed := time.Since(start)
		if err != nil {
			return probeMsg{name: name, res: probeResult{code: -1, latency: elapsed}}
		}
		resp.Body.Close()
		return probeMsg{name: name, res: probeResult{code: resp.StatusCode, latency: elapsed}}
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.fixOffset()
		return m, nil
	case probeMsg:
		m.results[msg.name] = msg.res
		return m, nil
	case tea.KeyMsg:
		// search-input mode: capture keystrokes into the filter string
		if m.searching {
			switch msg.Type {
			case tea.KeyEsc:
				m.searching = false
				m.filt.search = ""
				m.resetCursor()
			case tea.KeyEnter:
				m.searching = false
				m.resetCursor()
			case tea.KeyBackspace:
				if len(m.filt.search) > 0 {
					m.filt.search = m.filt.search[:len(m.filt.search)-1]
					m.resetCursor()
				}
			case tea.KeyRunes, tea.KeySpace:
				m.filt.search += string(msg.Runes)
				m.resetCursor()
			}
			return m, nil
		}
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
			m.fixOffset()
		case "down", "j":
			if m.cursor < m.filteredLen()-1 {
				m.cursor++
			}
			m.fixOffset()
		case "g", "home":
			m.cursor, m.offset = 0, 0
		case "G", "end":
			m.cursor = m.filteredLen() - 1
			if m.cursor < 0 {
				m.cursor = 0
			}
			m.fixOffset()
		case "pgup":
			m.cursor -= m.visibleRows()
			if m.cursor < 0 {
				m.cursor = 0
			}
			m.fixOffset()
		case "pgdown":
			m.cursor += m.visibleRows()
			if m.cursor >= m.filteredLen() {
				m.cursor = m.filteredLen() - 1
			}
			m.fixOffset()
		case "o", "enter":
			if sel := m.selected(); sel != nil {
				_ = exec.Command("xdg-open", siteURL(sel.Name)).Start()
			}
		case "l":
			if sel := m.selected(); sel != nil {
				bin, err := os.Executable()
				if err != nil {
					bin = "hostr"
				}
				return m, tea.ExecProcess(
					exec.Command(bin, "logs", sel.Name),
					func(error) tea.Msg { return nil },
				)
			}
		case "r":
			m.results = map[string]probeResult{}
			m.docExists = checkDocs(m.sites)
			return m, m.probeAll()
		// filters
		case "s":
			m.filt.secure = m.filt.secure.next()
			m.resetCursor()
		case "t":
			m.filt.kind = m.filt.kind.next()
			m.resetCursor()
		case "c":
			m.filt.code = m.filt.code.next()
			m.resetCursor()
		case "m":
			m.filt.missingOnly = !m.filt.missingOnly
			m.resetCursor()
		case "x":
			m.filt = filters{}
			m.resetCursor()
		case "/":
			m.searching = true
		}
	}
	return m, nil
}

func (m *model) resetCursor()    { m.cursor, m.offset = 0, 0 }
func (m *model) fixOffset() {
	v := m.visibleRows()
	if m.cursor < m.offset {
		m.offset = m.cursor
	}
	if m.cursor >= m.offset+v {
		m.offset = m.cursor - v + 1
	}
	if m.offset < 0 {
		m.offset = 0
	}
}

// --- Filtering -----------------------------------------------------------

func (m model) matches(s site.Resolved) bool {
	switch m.filt.secure {
	case secYes:
		if !s.Secure {
			return false
		}
	case secNo:
		if s.Secure {
			return false
		}
	}
	switch m.filt.kind {
	case kindPHP:
		if s.Kind != site.KindPHP {
			return false
		}
	case kindStatic:
		if s.Kind != site.KindStatic {
			return false
		}
	}
	res, has := m.results[s.Name]
	switch m.filt.code {
	case codePending:
		if has {
			return false
		}
	case codeErr:
		if !has || res.code != -1 {
			return false
		}
	case code2xx:
		if !has || res.code < 200 || res.code >= 300 {
			return false
		}
	case code3xx:
		if !has || res.code < 300 || res.code >= 400 {
			return false
		}
	case code4xx:
		if !has || res.code < 400 || res.code >= 500 {
			return false
		}
	case code5xx:
		if !has || res.code < 500 || res.code == -1 {
			return false
		}
	}
	if m.filt.missingOnly && m.docExists[s.Name] {
		return false
	}
	if m.filt.search != "" {
		if !strings.Contains(strings.ToLower(s.Name), strings.ToLower(m.filt.search)) {
			return false
		}
	}
	return true
}

func (m model) filtered() []site.Resolved {
	if !m.filt.any() {
		return m.sites
	}
	out := make([]site.Resolved, 0, len(m.sites))
	for _, s := range m.sites {
		if m.matches(s) {
			out = append(out, s)
		}
	}
	return out
}

// displayItem is one row in the grouped view: either a real site (parent or
// child) or a synthetic group header used when the parent matches no filter
// but its children do.
type displayItem struct {
	site     *site.Resolved // nil = synthetic header
	isChild  bool
	isLast   bool   // last child in its group (for tree-corner rendering)
	rootName string // populated for synthetic
}

// rootOf returns the last dot-segment of a site name. "app.affiliate" → "affiliate".
func rootOf(name string) string {
	if i := strings.LastIndex(name, "."); i >= 0 {
		return name[i+1:]
	}
	return name
}

// displayItems builds the grouped, filtered, cursor-navigable list.
// All grouping is done by *parent root domain*, with filtering applied
// independently to parents and children.
func (m model) displayItems() []displayItem {
	type group struct {
		root     string
		parent   *site.Resolved
		children []site.Resolved
	}
	groups := map[string]*group{}
	roots := []string{}
	for _, s := range m.sites {
		r := rootOf(s.Name)
		g, ok := groups[r]
		if !ok {
			g = &group{root: r}
			groups[r] = g
			roots = append(roots, r)
		}
		if s.Name == r {
			cp := s
			g.parent = &cp
		} else {
			g.children = append(g.children, s)
		}
	}
	sort.Strings(roots)

	out := []displayItem{}
	for _, r := range roots {
		g := groups[r]
		var parent *site.Resolved
		if g.parent != nil && m.matches(*g.parent) {
			parent = g.parent
		}
		visibleChildren := make([]site.Resolved, 0, len(g.children))
		for _, c := range g.children {
			if m.matches(c) {
				visibleChildren = append(visibleChildren, c)
			}
		}
		if parent == nil && len(visibleChildren) == 0 {
			continue
		}
		sort.Slice(visibleChildren, func(i, j int) bool {
			return visibleChildren[i].Name < visibleChildren[j].Name
		})
		if parent != nil {
			out = append(out, displayItem{site: parent})
		} else {
			out = append(out, displayItem{rootName: r})
		}
		for i, c := range visibleChildren {
			cp := c
			out = append(out, displayItem{
				site:    &cp,
				isChild: true,
				isLast:  i == len(visibleChildren)-1,
			})
		}
	}
	return out
}

func (m model) filteredLen() int { return len(m.displayItems()) }

func (m model) selected() *site.Resolved {
	items := m.displayItems()
	if m.cursor < 0 || m.cursor >= len(items) {
		return nil
	}
	return items[m.cursor].site // nil for synthetic
}

func (m model) visibleRows() int {
	// banner(1) + summary(1) + filter chips(1) + header(1) + footer(1) + scroll(1) ≈ 6
	h := m.height - 6
	if h < 5 {
		h = 5
	}
	if l := m.filteredLen(); h > l {
		h = l
	}
	if h < 1 {
		h = 1
	}
	return h
}

// --- View ----------------------------------------------------------------

func (m model) View() string {
	if m.width == 0 {
		return "loading…"
	}

	lay := computeLayout(m.width)

	// Banner — bold purple HOSTR with heavy bars on either side, then a tagline.
	// No background color so it renders identically in every terminal theme.
	label := bannerStyle.Render("HOSTR")
	tagline := taglineStyle.Render("local web dev server")
	leadBars := ruleStyle.Render("━━━ ")
	gap := ruleStyle.Render(" · ")
	core := leadBars + label + gap + tagline + " "
	visible := 4 + 5 + 3 + lipgloss.Width(tagline) + 1 // bars + HOSTR + " · " + tagline + trailing space
	tailLen := m.width - visible - 1                   // 1 col leading space
	if tailLen < 0 {
		tailLen = 0
	}
	titleLine := " " + core + ruleStyle.Render(strings.Repeat("━", tailLen))

	all := m.sites
	filt := m.filtered()
	ok, warn, errC, pending, missing := 0, 0, 0, 0, 0
	for _, s := range all {
		r, has := m.results[s.Name]
		switch {
		case !has:
			pending++
		case r.code == -1 || r.code >= 500:
			errC++
		case r.code >= 400:
			warn++
		default:
			ok++
		}
		if !m.docExists[s.Name] {
			missing++
		}
	}

	// Line 1 — title + counts
	missingPart := ""
	if missing > 0 {
		missingPart = fmt.Sprintf("  %s missing", warnStyle.Render(strconv.Itoa(missing)))
	}
	summary := fmt.Sprintf(" %d/%d sites    %s ok  %s warn  %s err  %s pending%s",
		len(filt), len(all),
		okStyle.Render(strconv.Itoa(ok)),
		warnStyle.Render(strconv.Itoa(warn)),
		errStyle.Render(strconv.Itoa(errC)),
		dimStyle.Render(strconv.Itoa(pending)),
		missingPart)

	// Line 2 — filter chips / search input
	chips := m.filterLine()

	// Header (leading 2 cols reserved for the cursor marker on body rows)
	headParts := []string{pad("NAME", lay.nameW), pad("HTTPS", lay.httpsW)}
	if lay.showKind {
		headParts = append(headParts, pad("KIND", lay.kindW))
	}
	if lay.showPHP {
		headParts = append(headParts, pad("PHP", lay.phpW))
	}
	headParts = append(headParts, pad("STAT", lay.statW))
	if lay.showLat {
		headParts = append(headParts, pad("LAT", lay.latW))
	}
	if lay.showDoc {
		headParts = append(headParts, pad("DOCROOT", lay.docW))
	}
	header := "  " + headerStyle.Render(strings.Join(headParts, ""))

	// Body
	items := m.displayItems()
	var body strings.Builder
	if len(items) == 0 {
		body.WriteString(dimStyle.Render("  no sites match the current filters"))
		body.WriteString("\n")
	} else {
		visible := m.visibleRows()
		end := m.offset + visible
		if end > len(items) {
			end = len(items)
		}
		for i := m.offset; i < end; i++ {
			it := items[i]
			var content string
			if it.site == nil {
				content = m.renderSynthetic(it.rootName, lay)
			} else {
				content = m.renderRow(*it.site, it.isChild, it.isLast, lay)
			}
			var marker string
			if i == m.cursor {
				// Bright chevron at column 0 — bg-independent, so it stays visible
				// even where inner ANSI resets eat the selected-row background.
				marker = cursorStyle.Render("❯ ")
			} else {
				marker = "  "
			}
			line := marker + content
			if i == m.cursor {
				line = selectedStyle.Render(line)
			}
			body.WriteString(line)
			body.WriteString("\n")
		}
		if len(items) > visible {
			body.WriteString(dimStyle.Render(fmt.Sprintf("  %d–%d of %d", m.offset+1, end, len(items))))
			body.WriteString("\n")
		}
	}

	// Adaptive footer — drop hints from the right until it fits.
	footer := footerStyle.Render("  " + fitJoin(m.width-2, " · ",
		"↑↓ move",
		"o open",
		"l logs",
		"/ search",
		"s/t/c/m filter",
		"r refresh",
		"x clear",
		"g/G top/bot",
		"q quit",
	))

	return strings.Join([]string{
		titleLine, summary, chips, header,
		strings.TrimRight(body.String(), "\n"), // body builds rows with trailing "\n"; join adds another, which scrolls line 1 off
		footer,
	}, "\n")
}

// fitJoin packs as many parts as fit within width, joined by sep.
func fitJoin(width int, sep string, parts ...string) string {
	if width < 1 {
		return ""
	}
	out := ""
	for i, p := range parts {
		candidate := out
		if i > 0 {
			candidate += sep
		}
		candidate += p
		if lipgloss.Width(candidate) > width {
			break
		}
		out = candidate
	}
	return out
}

func (m model) filterLine() string {
	if m.searching {
		return " " + searchStyle.Render("/ ") + m.filt.search + searchStyle.Render("▏")
	}
	parts := []string{}
	if l := m.filt.secure.label(); l != "" {
		parts = append(parts, "https="+l)
	}
	if l := m.filt.kind.label(); l != "" {
		parts = append(parts, "kind="+l)
	}
	if l := m.filt.code.label(); l != "" {
		parts = append(parts, "code="+l)
	}
	if m.filt.missingOnly {
		parts = append(parts, "missing-docroot")
	}
	if m.filt.search != "" {
		parts = append(parts, "search="+strconv.Quote(m.filt.search))
	}
	if len(parts) == 0 {
		return dimStyle.Render(" filters: (none)")
	}
	return " " + chipStyle.Render("filters:") + " " + strings.Join(parts, "  ")
}

func (m model) renderSynthetic(rootName string, lay layout) string {
	name := pad(truncate(rootName+".test", lay.nameW-1), lay.nameW)
	name = dimStyle.Render(name)
	restW := lay.httpsW + lay.statW
	if lay.showKind {
		restW += lay.kindW
	}
	if lay.showPHP {
		restW += lay.phpW
	}
	if lay.showLat {
		restW += lay.latW
	}
	if lay.showDoc {
		restW += lay.docW
	}
	rest := dimStyle.Render(pad("(no parent site — children only)", restW))
	return name + rest
}

func (m model) renderRow(s site.Resolved, isChild, isLast bool, lay layout) string {
	displayName := s.Name + ".test"
	prefix := ""
	prefixLen := 0
	if isChild {
		corner := "├─"
		if isLast {
			corner = "└─"
		}
		prefix = "  " + corner + " "
		prefixLen = 5 // "  X─ " is 5 visible cols
	}
	truncated := truncate(displayName, lay.nameW-prefixLen-1)
	plain := prefix + truncated
	styled := dimStyle.Render(prefix) + truncated
	name := padStyled(styled, plain, lay.nameW)

	httpsText := "no"
	httpsStyle := warnStyle
	if s.Secure {
		httpsText = "yes"
		httpsStyle = okStyle
	}
	httpsCell := padStyled(httpsStyle.Render(httpsText), httpsText, lay.httpsW)

	php := s.PHP
	if php == "" {
		php = "-"
	}
	phpCell := pad(php, lay.phpW)

	res, has := m.results[s.Name]
	var statText, statStyled, latText string
	switch {
	case !has:
		statText = "-"
		statStyled = dimStyle.Render(statText)
		latText = "-"
	case res.code == -1:
		statText = "ERR"
		statStyled = errStyle.Render(statText)
		latText = formatDur(res.latency)
	default:
		statText = strconv.Itoa(res.code)
		switch {
		case res.code >= 500:
			statStyled = errStyle.Render(statText)
		case res.code >= 400:
			statStyled = warnStyle.Render(statText)
		case res.code >= 200:
			statStyled = okStyle.Render(statText)
		default:
			statStyled = statText
		}
		latText = formatDur(res.latency)
	}
	statCell := padStyled(statStyled, statText, lay.statW)
	latCell := pad(latText, lay.latW)

	var docPlain, docStyled string
	if s.Kind == site.KindProxy {
		docPlain = "→ " + s.Target
		docStyled = chipStyle.Render("→ ") + s.Target
	} else {
		docPlain = truncate(s.Docroot, lay.docW-2)
		docStyled = docPlain
		if !m.docExists[s.Name] {
			docStyled = errStyle.Render("✗ " + docPlain)
			docPlain = "✗ " + docPlain
		}
	}
	docCell := padStyled(docStyled, docPlain, lay.docW)

	parts := []string{name, httpsCell}
	if lay.showKind {
		parts = append(parts, pad(string(s.Kind), lay.kindW))
	}
	if lay.showPHP {
		parts = append(parts, phpCell)
	}
	parts = append(parts, statCell)
	if lay.showLat {
		parts = append(parts, latCell)
	}
	if lay.showDoc {
		parts = append(parts, docCell)
	}
	return strings.Join(parts, "")
}

// --- helpers -------------------------------------------------------------

func pad(s string, w int) string {
	visible := lipgloss.Width(s)
	if visible >= w {
		return s
	}
	return s + strings.Repeat(" ", w-visible)
}

func padStyled(styled, plain string, w int) string {
	visible := len([]rune(plain))
	if visible >= w {
		return styled
	}
	return styled + strings.Repeat(" ", w-visible)
}

func truncate(s string, w int) string {
	if w <= 0 {
		return ""
	}
	r := []rune(s)
	if len(r) <= w {
		return s
	}
	if w < 2 {
		return string(r[:w])
	}
	return string(r[:w-1]) + "…"
}

func formatDur(d time.Duration) string {
	switch {
	case d < time.Millisecond:
		return "<1ms"
	case d < time.Second:
		return fmt.Sprintf("%dms", d.Milliseconds())
	default:
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
}

func checkDocs(sites []site.Resolved) map[string]bool {
	out := make(map[string]bool, len(sites))
	for _, s := range sites {
		// Proxies don't have a docroot; mark as "exists" so they're never flagged missing.
		if s.Kind == site.KindProxy {
			out[s.Name] = true
			continue
		}
		_, err := os.Stat(s.Docroot)
		out[s.Name] = err == nil
	}
	return out
}

// --- entrypoint ----------------------------------------------------------

func Run() error {
	s, err := site.Load()
	if err != nil {
		return err
	}
	sites := s.Resolve()
	if len(sites) == 0 {
		return fmt.Errorf("no sites configured. Run `hostr park <dir>` or `hostr link [name]` first")
	}
	m := model{
		sites:     sites,
		results:   map[string]probeResult{},
		docExists: checkDocs(sites),
	}
	_, err = tea.NewProgram(m, tea.WithAltScreen()).Run()
	return err
}

// DebugRender returns one rendered View frame (no event loop) at the given width.
// Used by `hostr tui-render` to inspect output non-interactively.
func DebugRender(width int) string {
	s, err := site.Load()
	if err != nil {
		return "load error: " + err.Error()
	}
	sites := s.Resolve()
	m := model{
		sites:     sites,
		results:   map[string]probeResult{},
		docExists: checkDocs(sites),
		width:     width,
		height:    40,
	}
	return m.View()
}

func siteURL(name string) string {
	if portBound("127.0.0.1:443") || portBound(":443") {
		return fmt.Sprintf("https://%s.test", name)
	}
	return fmt.Sprintf("https://%s.test:8443", name)
}

func portBound(addr string) bool {
	c, err := net.DialTimeout("tcp", addr, 200*time.Millisecond)
	if err != nil {
		return false
	}
	c.Close()
	return true
}
