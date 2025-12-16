package tui

import (
	"snitch/internal/collector"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type tickMsg time.Time

type dataMsg struct {
	connections []collector.Connection
}

type errMsg struct {
	err error
}

func (m model) tick() tea.Cmd {
	return tea.Tick(m.interval, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m model) fetchData() tea.Cmd {
	return func() tea.Msg {
		conns, err := collector.GetConnections()
		if err != nil {
			return errMsg{err}
		}
		return dataMsg{connections: conns}
	}
}

