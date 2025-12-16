package cmd

import (
	"log"
	"snitch/internal/config"
	"snitch/internal/tui"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

var (
	topTheme    string
	topInterval time.Duration
	topTCP      bool
	topUDP      bool
	topListen   bool
	topEstab    bool
)

var topCmd = &cobra.Command{
	Use:   "top",
	Short: "Live TUI for inspecting connections",
	Run: func(cmd *cobra.Command, args []string) {
		cfg := config.Get()

		theme := topTheme
		if theme == "" {
			theme = cfg.Defaults.Theme
		}

		opts := tui.Options{
			Theme:    theme,
			Interval: topInterval,
		}

		// if any filter flag is set, use exclusive mode
		if topTCP || topUDP || topListen || topEstab {
			opts.TCP = topTCP
			opts.UDP = topUDP
			opts.Listening = topListen
			opts.Established = topEstab
			opts.Other = false
			opts.FilterSet = true
		}

		m := tui.New(opts)

		p := tea.NewProgram(m, tea.WithAltScreen())
		if _, err := p.Run(); err != nil {
			log.Fatal(err)
		}
	},
}

func init() {
	rootCmd.AddCommand(topCmd)
	cfg := config.Get()
	topCmd.Flags().StringVar(&topTheme, "theme", cfg.Defaults.Theme, "Theme for TUI (dark, light, mono, auto)")
	topCmd.Flags().DurationVarP(&topInterval, "interval", "i", time.Second, "Refresh interval")
	topCmd.Flags().BoolVarP(&topTCP, "tcp", "t", false, "Show only TCP connections")
	topCmd.Flags().BoolVarP(&topUDP, "udp", "u", false, "Show only UDP connections")
	topCmd.Flags().BoolVarP(&topListen, "listen", "l", false, "Show only listening sockets")
	topCmd.Flags().BoolVarP(&topEstab, "established", "e", false, "Show only established connections")
}