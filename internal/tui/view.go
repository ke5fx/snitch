package tui

import (
	"fmt"
	"snitch/internal/collector"
	"strings"
	"time"
)

func (m model) renderMain() string {
	var b strings.Builder

	b.WriteString("\n")
	b.WriteString(m.renderTitle())
	b.WriteString("\n")
	b.WriteString(m.renderFilters())
	b.WriteString("\n\n")
	b.WriteString(m.renderTableHeader())
	b.WriteString(m.renderSeparator())
	b.WriteString(m.renderConnections())
	b.WriteString("\n")
	b.WriteString(m.renderStatusLine())

	return b.String()
}

func (m model) renderTitle() string {
	visible := m.visibleConnections()
	total := len(m.connections)

	left := m.theme.Styles.Header.Render("snitch")

	ago := time.Since(m.lastRefresh).Round(time.Millisecond * 100)
	right := m.theme.Styles.Normal.Render(fmt.Sprintf("%d/%d connections  ↻ %s", len(visible), total, formatDuration(ago)))

	w := m.safeWidth()
	gap := w - len(stripAnsi(left)) - len(stripAnsi(right)) - 2
	if gap < 0 {
		gap = 0
	}

	return "  " + left + strings.Repeat(" ", gap) + right
}

func (m model) renderFilters() string {
	var parts []string

	if m.showTCP {
		parts = append(parts, m.theme.Styles.Success.Render("tcp"))
	} else {
		parts = append(parts, m.theme.Styles.Normal.Render("tcp"))
	}

	if m.showUDP {
		parts = append(parts, m.theme.Styles.Success.Render("udp"))
	} else {
		parts = append(parts, m.theme.Styles.Normal.Render("udp"))
	}

	parts = append(parts, m.theme.Styles.Border.Render("│"))

	if m.showListening {
		parts = append(parts, m.theme.Styles.Success.Render("listen"))
	} else {
		parts = append(parts, m.theme.Styles.Normal.Render("listen"))
	}

	if m.showEstablished {
		parts = append(parts, m.theme.Styles.Success.Render("estab"))
	} else {
		parts = append(parts, m.theme.Styles.Normal.Render("estab"))
	}

	if m.showOther {
		parts = append(parts, m.theme.Styles.Success.Render("other"))
	} else {
		parts = append(parts, m.theme.Styles.Normal.Render("other"))
	}

	left := "  " + strings.Join(parts, "  ")

	sortLabel := sortFieldLabel(m.sortField)
	sortDir := "↑"
	if m.sortReverse {
		sortDir = "↓"
	}

	var right string
	if m.searchActive {
		right = m.theme.Styles.Warning.Render(fmt.Sprintf("/%s▌", m.searchQuery))
	} else if m.searchQuery != "" {
		right = m.theme.Styles.Normal.Render(fmt.Sprintf("filter: %s", m.searchQuery))
	} else {
		right = m.theme.Styles.Normal.Render(fmt.Sprintf("sort: %s %s", sortLabel, sortDir))
	}

	w := m.safeWidth()
	gap := w - len(stripAnsi(left)) - len(stripAnsi(right)) - 2
	if gap < 0 {
		gap = 0
	}

	return left + strings.Repeat(" ", gap) + right + "  "
}

func (m model) renderTableHeader() string {
	cols := m.columnWidths()

	header := fmt.Sprintf("  %-*s  %-*s  %-*s  %-*s  %-*s  %s",
		cols.process, "PROCESS",
		cols.port, "PORT",
		cols.proto, "PROTO",
		cols.state, "STATE",
		cols.local, "LOCAL",
		"REMOTE")

	return m.theme.Styles.Header.Render(header) + "\n"
}

func (m model) renderSeparator() string {
	w := m.width - 4
	if w < 1 {
		w = 76
	}
	line := "  " + strings.Repeat("─", w)
	return m.theme.Styles.Border.Render(line) + "\n"
}

func (m model) renderConnections() string {
	var b strings.Builder
	visible := m.visibleConnections()
	pageSize := m.pageSize()

	if len(visible) == 0 {
		empty := "\n  " + m.theme.Styles.Normal.Render("no connections match filters") + "\n"
		return empty
	}

	start := m.scrollOffset(pageSize, len(visible))

	for i := 0; i < pageSize; i++ {
		idx := start + i
		if idx >= len(visible) {
			b.WriteString("\n")
			continue
		}

		isSelected := idx == m.cursor
		b.WriteString(m.renderRow(visible[idx], isSelected))
	}

	return b.String()
}

func (m model) renderRow(c collector.Connection, selected bool) string {
	cols := m.columnWidths()

	indicator := "  "
	if selected {
		indicator = m.theme.Styles.Success.Render("▸ ")
	}

	process := truncate(c.Process, cols.process)
	if process == "" {
		process = "–"
	}

	port := fmt.Sprintf("%d", c.Lport)
	proto := c.Proto
	state := c.State
	if state == "" {
		state = "–"
	}

	local := c.Laddr
	if local == "*" || local == "" {
		local = "*"
	}

	remote := formatRemote(c.Raddr, c.Rport)

	// apply styling
	protoStyled := m.theme.Styles.GetProtoStyle(proto).Render(fmt.Sprintf("%-*s", cols.proto, proto))
	stateStyled := m.theme.Styles.GetStateStyle(state).Render(fmt.Sprintf("%-*s", cols.state, truncate(state, cols.state)))

	row := fmt.Sprintf("%s%-*s  %-*s  %s  %s  %-*s  %s",
		indicator,
		cols.process, process,
		cols.port, port,
		protoStyled,
		stateStyled,
		cols.local, truncate(local, cols.local),
		truncate(remote, cols.remote))

	if selected {
		return m.theme.Styles.Selected.Render(row) + "\n"
	}

	return m.theme.Styles.Normal.Render(row) + "\n"
}

func (m model) renderStatusLine() string {
	left := "  " + m.theme.Styles.Normal.Render("t/u proto  l/e/o state  s sort  / search  ? help  q quit")

	return left
}

func (m model) renderError() string {
	return fmt.Sprintf("\n  %s\n\n  press q to quit\n",
		m.theme.Styles.Error.Render(fmt.Sprintf("error: %v", m.err)))
}

func (m model) renderHelp() string {
	help := `
  navigation
  ──────────
  j/k ↑/↓      move cursor
  g/G          jump to top/bottom
  ctrl+d/u     half page down/up
  enter        show connection details

  filters
  ───────
  t            toggle tcp
  u            toggle udp
  l            toggle listening
  e            toggle established
  o            toggle other states
  a            reset all filters

  sorting
  ───────
  s            cycle sort field
  S            reverse sort order

  other
  ─────
  /            search
  r            refresh now
  q            quit

  press ? or esc to close
`
	return m.theme.Styles.Normal.Render(help)
}

func (m model) renderDetail() string {
	if m.selected == nil {
		return ""
	}

	c := m.selected
	var b strings.Builder

	b.WriteString("\n")
	b.WriteString("  " + m.theme.Styles.Header.Render("connection details") + "\n")
	b.WriteString("  " + m.theme.Styles.Border.Render(strings.Repeat("─", 40)) + "\n\n")

	fields := []struct {
		label string
		value string
	}{
		{"process", c.Process},
		{"pid", fmt.Sprintf("%d", c.PID)},
		{"user", c.User},
		{"protocol", c.Proto},
		{"state", c.State},
		{"local", fmt.Sprintf("%s:%d", c.Laddr, c.Lport)},
		{"remote", fmt.Sprintf("%s:%d", c.Raddr, c.Rport)},
		{"interface", c.Interface},
		{"inode", fmt.Sprintf("%d", c.Inode)},
	}

	for _, f := range fields {
		val := f.value
		if val == "" || val == "0" || val == ":0" {
			val = "–"
		}
		line := fmt.Sprintf("  %-12s  %s\n", m.theme.Styles.Header.Render(f.label), val)
		b.WriteString(line)
	}

	b.WriteString("\n")
	b.WriteString("  " + m.theme.Styles.Normal.Render("press esc to close") + "\n")

	return b.String()
}

func (m model) scrollOffset(pageSize, total int) int {
	if total <= pageSize {
		return 0
	}

	// keep cursor roughly centered
	offset := m.cursor - pageSize/2
	if offset < 0 {
		offset = 0
	}
	if offset > total-pageSize {
		offset = total - pageSize
	}
	return offset
}

type columns struct {
	process int
	port    int
	proto   int
	state   int
	local   int
	remote  int
}

func (m model) columnWidths() columns {
	available := m.safeWidth() - 16

	c := columns{
		process: 16,
		port:    6,
		proto:   5,
		state:   11,
		local:   15,
		remote:  20,
	}

	used := c.process + c.port + c.proto + c.state + c.local + c.remote
	extra := available - used

	if extra > 0 {
		c.process += extra / 3
		c.remote += extra - extra/3
	}

	return c
}

func (m model) safeWidth() int {
	if m.width < 80 {
		return 80
	}
	return m.width
}

func formatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	return fmt.Sprintf("%.0fm", d.Minutes())
}
