package theme

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Theme represents the visual styling for the TUI
type Theme struct {
	Name   string
	Styles Styles
}

// Styles contains all the styling definitions
type Styles struct {
	Header     lipgloss.Style
	Border     lipgloss.Style
	Selected   lipgloss.Style
	Watched    lipgloss.Style
	Normal     lipgloss.Style
	Error      lipgloss.Style
	Success    lipgloss.Style
	Warning    lipgloss.Style
	Proto      ProtoStyles
	State      StateStyles
	Footer     lipgloss.Style
	Background lipgloss.Style
}

// ProtoStyles contains protocol-specific colors
type ProtoStyles struct {
	TCP  lipgloss.Style
	UDP  lipgloss.Style
	Unix lipgloss.Style
	TCP6 lipgloss.Style
	UDP6 lipgloss.Style
}

// StateStyles contains connection state-specific colors
type StateStyles struct {
	Listen      lipgloss.Style
	Established lipgloss.Style
	TimeWait    lipgloss.Style
	CloseWait   lipgloss.Style
	SynSent     lipgloss.Style
	SynRecv     lipgloss.Style
	FinWait1    lipgloss.Style
	FinWait2    lipgloss.Style
	Closing     lipgloss.Style
	LastAck     lipgloss.Style
	Closed      lipgloss.Style
}

var (
	themes map[string]*Theme
)

func init() {
	themes = map[string]*Theme{
		"default": createAdaptiveTheme(),
		"mono":    createMonoTheme(),
	}
}

// GetTheme returns a theme by name, with auto-detection support
func GetTheme(name string) *Theme {
	if name == "auto" {
		// lipgloss handles adaptive colors, so we just return the default
		return themes["default"]
	}

	if theme, exists := themes[name]; exists {
		return theme
	}

	// a specific theme was requested (e.g. "dark", "light"), but we now use adaptive
	// so we can just return the default theme and lipgloss will handle it
	if name == "dark" || name == "light" {
		return themes["default"]
	}

	// fallback to default
	return themes["default"]
}

// ListThemes returns available theme names
func ListThemes() []string {
	var names []string
	for name := range themes {
		names = append(names, name)
	}
	return names
}

// createAdaptiveTheme creates a clean, minimal theme
func createAdaptiveTheme() *Theme {
	return &Theme{
		Name: "default",
		Styles: Styles{
			Header: lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.AdaptiveColor{Light: "#1F2937", Dark: "#F9FAFB"}),

			Watched: lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.AdaptiveColor{Light: "#D97706", Dark: "#F59E0B"}),

			Border: lipgloss.NewStyle().
				Foreground(lipgloss.AdaptiveColor{Light: "#D1D5DB", Dark: "#374151"}),

			Selected: lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.AdaptiveColor{Light: "#1F2937", Dark: "#F9FAFB"}),

			Normal: lipgloss.NewStyle().
				Foreground(lipgloss.AdaptiveColor{Light: "#6B7280", Dark: "#9CA3AF"}),

			Error: lipgloss.NewStyle().
				Foreground(lipgloss.AdaptiveColor{Light: "#DC2626", Dark: "#F87171"}),

			Success: lipgloss.NewStyle().
				Foreground(lipgloss.AdaptiveColor{Light: "#059669", Dark: "#34D399"}),

			Warning: lipgloss.NewStyle().
				Foreground(lipgloss.AdaptiveColor{Light: "#D97706", Dark: "#FBBF24"}),

			Footer: lipgloss.NewStyle().
				Foreground(lipgloss.AdaptiveColor{Light: "#9CA3AF", Dark: "#6B7280"}),

			Background: lipgloss.NewStyle(),

			Proto: ProtoStyles{
				TCP:  lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#059669", Dark: "#34D399"}),
				UDP:  lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#7C3AED", Dark: "#A78BFA"}),
				Unix: lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#6B7280", Dark: "#9CA3AF"}),
				TCP6: lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#059669", Dark: "#34D399"}),
				UDP6: lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#7C3AED", Dark: "#A78BFA"}),
			},

			State: StateStyles{
				Listen:      lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#059669", Dark: "#34D399"}),
				Established: lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#2563EB", Dark: "#60A5FA"}),
				TimeWait:    lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#D97706", Dark: "#FBBF24"}),
				CloseWait:   lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#D97706", Dark: "#FBBF24"}),
				SynSent:     lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#7C3AED", Dark: "#A78BFA"}),
				SynRecv:     lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#7C3AED", Dark: "#A78BFA"}),
				FinWait1:    lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#DC2626", Dark: "#F87171"}),
				FinWait2:    lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#DC2626", Dark: "#F87171"}),
				Closing:     lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#DC2626", Dark: "#F87171"}),
				LastAck:     lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#DC2626", Dark: "#F87171"}),
				Closed:      lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#9CA3AF", Dark: "#6B7280"}),
			},
		},
	}
}

// createMonoTheme creates a monochrome theme (no colors)
func createMonoTheme() *Theme {
	baseStyle := lipgloss.NewStyle()
	boldStyle := lipgloss.NewStyle().Bold(true)

	return &Theme{
		Name: "mono",
		Styles: Styles{
			Header:     boldStyle,
			Border:     baseStyle,
			Selected:   boldStyle,
			Normal:     baseStyle,
			Error:      boldStyle,
			Success:    boldStyle,
			Warning:    boldStyle,
			Footer:     baseStyle,
			Background: baseStyle,

			Proto: ProtoStyles{
				TCP:  baseStyle,
				UDP:  baseStyle,
				Unix: baseStyle,
				TCP6: baseStyle,
				UDP6: baseStyle,
			},

			State: StateStyles{
				Listen:      baseStyle,
				Established: baseStyle,
				TimeWait:    baseStyle,
				CloseWait:   baseStyle,
				SynSent:     baseStyle,
				SynRecv:     baseStyle,
				FinWait1:    baseStyle,
				FinWait2:    baseStyle,
				Closing:     baseStyle,
				LastAck:     baseStyle,
				Closed:      baseStyle,
			},
		},
	}
}

// GetProtoStyle returns the appropriate style for a protocol
func (s *Styles) GetProtoStyle(proto string) lipgloss.Style {
	switch strings.ToLower(proto) {
	case "tcp":
		return s.Proto.TCP
	case "udp":
		return s.Proto.UDP
	case "unix":
		return s.Proto.Unix
	case "tcp6":
		return s.Proto.TCP6
	case "udp6":
		return s.Proto.UDP6
	default:
		return s.Normal
	}
}

// GetStateStyle returns the appropriate style for a connection state
func (s *Styles) GetStateStyle(state string) lipgloss.Style {
	switch strings.ToUpper(state) {
	case "LISTEN":
		return s.State.Listen
	case "ESTABLISHED":
		return s.State.Established
	case "TIME_WAIT":
		return s.State.TimeWait
	case "CLOSE_WAIT":
		return s.State.CloseWait
	case "SYN_SENT":
		return s.State.SynSent
	case "SYN_RECV":
		return s.State.SynRecv
	case "FIN_WAIT1":
		return s.State.FinWait1
	case "FIN_WAIT2":
		return s.State.FinWait2
	case "CLOSING":
		return s.State.Closing
	case "LAST_ACK":
		return s.State.LastAck
	case "CLOSED":
		return s.State.Closed
	default:
		return s.Normal
	}
}
