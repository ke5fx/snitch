package tui

import (
	"snitch/internal/collector"

	tea "github.com/charmbracelet/bubbletea"
)

func (m model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// search mode captures all input
	if m.searchActive {
		return m.handleSearchKey(msg)
	}

	// detail view only allows closing
	if m.showDetail {
		return m.handleDetailKey(msg)
	}

	// help view only allows closing
	if m.showHelp {
		return m.handleHelpKey(msg)
	}

	return m.handleNormalKey(msg)
}

func (m model) handleSearchKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.searchActive = false
		m.searchQuery = ""
	case "enter":
		m.searchActive = false
		m.cursor = 0
	case "backspace":
		if len(m.searchQuery) > 0 {
			m.searchQuery = m.searchQuery[:len(m.searchQuery)-1]
		}
	default:
		if len(msg.String()) == 1 {
			m.searchQuery += msg.String()
		}
	}
	return m, nil
}

func (m model) handleDetailKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "enter", "q":
		m.showDetail = false
		m.selected = nil
	}
	return m, nil
}

func (m model) handleHelpKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "enter", "q", "?":
		m.showHelp = false
	}
	return m, nil
}

func (m model) handleNormalKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Sequence(tea.ShowCursor, tea.Quit)

	// navigation
	case "j", "down":
		m.moveCursor(1)
	case "k", "up":
		m.moveCursor(-1)
	case "g":
		m.cursor = 0
	case "G":
		visible := m.visibleConnections()
		if len(visible) > 0 {
			m.cursor = len(visible) - 1
		}
	case "ctrl+d":
		m.moveCursor(m.pageSize() / 2)
	case "ctrl+u":
		m.moveCursor(-m.pageSize() / 2)
	case "ctrl+f", "pgdown":
		m.moveCursor(m.pageSize())
	case "ctrl+b", "pgup":
		m.moveCursor(-m.pageSize())

	// filter toggles
	case "t":
		m.showTCP = !m.showTCP
		m.clampCursor()
	case "u":
		m.showUDP = !m.showUDP
		m.clampCursor()
	case "l":
		m.showListening = !m.showListening
		m.clampCursor()
	case "e":
		m.showEstablished = !m.showEstablished
		m.clampCursor()
	case "o":
		m.showOther = !m.showOther
		m.clampCursor()
	case "a":
		m.showTCP = true
		m.showUDP = true
		m.showListening = true
		m.showEstablished = true
		m.showOther = true

	// sorting
	case "s":
		m.cycleSort()
	case "S":
		m.sortReverse = !m.sortReverse
		m.applySorting()

	// search
	case "/":
		m.searchActive = true
		m.searchQuery = ""

	// actions
	case "enter", " ":
		visible := m.visibleConnections()
		if m.cursor < len(visible) {
			conn := visible[m.cursor]
			m.selected = &conn
			m.showDetail = true
		}
	case "r":
		return m, m.fetchData()
	case "?":
		m.showHelp = true
	}

	return m, nil
}

func (m *model) moveCursor(delta int) {
	visible := m.visibleConnections()
	m.cursor += delta
	if m.cursor < 0 {
		m.cursor = 0
	}
	if m.cursor >= len(visible) {
		m.cursor = len(visible) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
}

func (m model) pageSize() int {
	size := m.height - 6
	if size < 1 {
		return 10
	}
	return size
}

func (m *model) cycleSort() {
	fields := []collector.SortField{
		collector.SortByLport,
		collector.SortByProcess,
		collector.SortByPID,
		collector.SortByState,
		collector.SortByProto,
	}

	for i, f := range fields {
		if f == m.sortField {
			m.sortField = fields[(i+1)%len(fields)]
			m.applySorting()
			return
		}
	}

	m.sortField = collector.SortByLport
	m.applySorting()
}

