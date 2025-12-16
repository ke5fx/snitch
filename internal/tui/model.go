package tui

import (
	"snitch/internal/collector"
	"snitch/internal/theme"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type model struct {
	connections []collector.Connection
	cursor      int
	width       int
	height      int

	// filtering
	showTCP         bool
	showUDP         bool
	showListening   bool
	showEstablished bool
	showOther       bool
	searchQuery     string
	searchActive    bool

	// sorting
	sortField   collector.SortField
	sortReverse bool

	// ui state
	theme       *theme.Theme
	showHelp    bool
	showDetail  bool
	selected    *collector.Connection
	interval    time.Duration
	lastRefresh time.Time
	err         error
}

type Options struct {
	Theme       string
	Interval    time.Duration
	TCP         bool
	UDP         bool
	Listening   bool
	Established bool
	Other       bool
	FilterSet   bool // true if user specified any filter flags
}

func New(opts Options) model {
	interval := opts.Interval
	if interval == 0 {
		interval = time.Second
	}

	// default: show everything
	showTCP := true
	showUDP := true
	showListening := true
	showEstablished := true
	showOther := true

	// if user specified filters, use those instead
	if opts.FilterSet {
		showTCP = opts.TCP
		showUDP = opts.UDP
		showListening = opts.Listening
		showEstablished = opts.Established
		showOther = opts.Other

		// if only proto filters set, show all states
		if !opts.Listening && !opts.Established && !opts.Other {
			showListening = true
			showEstablished = true
			showOther = true
		}
		// if only state filters set, show all protos
		if !opts.TCP && !opts.UDP {
			showTCP = true
			showUDP = true
		}
	}

	return model{
		connections:     []collector.Connection{},
		showTCP:         showTCP,
		showUDP:         showUDP,
		showListening:   showListening,
		showEstablished: showEstablished,
		showOther:       showOther,
		sortField:       collector.SortByLport,
		theme:           theme.GetTheme(opts.Theme),
		interval:        interval,
		lastRefresh:     time.Now(),
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		tea.HideCursor,
		m.fetchData(),
		m.tick(),
	)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)

	case tickMsg:
		return m, tea.Batch(m.fetchData(), m.tick())

	case dataMsg:
		m.connections = msg.connections
		m.lastRefresh = time.Now()
		m.applySorting()
		m.clampCursor()
		return m, nil

	case errMsg:
		m.err = msg.err
		return m, nil
	}

	return m, nil
}

func (m model) View() string {
	if m.err != nil {
		return m.renderError()
	}
	if m.showHelp {
		return m.renderHelp()
	}
	if m.showDetail && m.selected != nil {
		return m.renderDetail()
	}
	return m.renderMain()
}

func (m *model) applySorting() {
	direction := collector.SortAsc
	if m.sortReverse {
		direction = collector.SortDesc
	}
	collector.SortConnections(m.connections, collector.SortOptions{
		Field:     m.sortField,
		Direction: direction,
	})
}

func (m *model) clampCursor() {
	visible := m.visibleConnections()
	if m.cursor >= len(visible) {
		m.cursor = len(visible) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
}

func (m model) visibleConnections() []collector.Connection {
	var result []collector.Connection

	for _, c := range m.connections {
		if !m.matchesFilters(c) {
			continue
		}
		if m.searchQuery != "" && !m.matchesSearch(c) {
			continue
		}
		result = append(result, c)
	}

	return result
}

func (m model) matchesFilters(c collector.Connection) bool {
	isTCP := c.Proto == "tcp" || c.Proto == "tcp6"
	isUDP := c.Proto == "udp" || c.Proto == "udp6"

	if isTCP && !m.showTCP {
		return false
	}
	if isUDP && !m.showUDP {
		return false
	}

	isListening := c.State == "LISTEN"
	isEstablished := c.State == "ESTABLISHED"
	isOther := !isListening && !isEstablished

	if isListening && !m.showListening {
		return false
	}
	if isEstablished && !m.showEstablished {
		return false
	}
	if isOther && !m.showOther {
		return false
	}

	return true
}

func (m model) matchesSearch(c collector.Connection) bool {
	return containsIgnoreCase(c.Process, m.searchQuery) ||
		containsIgnoreCase(c.Laddr, m.searchQuery) ||
		containsIgnoreCase(c.Raddr, m.searchQuery) ||
		containsIgnoreCase(c.User, m.searchQuery) ||
		containsIgnoreCase(c.Proto, m.searchQuery) ||
		containsIgnoreCase(c.State, m.searchQuery)
}
