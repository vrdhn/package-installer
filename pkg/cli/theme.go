package cli

import (
	"github.com/charmbracelet/lipgloss"
)

// Theme defines colors and symbols for the CLI using lipgloss
type Theme struct {
	Bold   lipgloss.Style
	Cyan   lipgloss.Style
	Green  lipgloss.Style
	Yellow lipgloss.Style
	Dim    lipgloss.Style
	Red    lipgloss.Style

	Bullet  string
	Arrow   string
	BoxTree string
	BoxLast string
	BoxItem string

	IconPkg   string
	IconCave  string
	IconDisk  string
	IconWorld string
	IconHelp  string
}

func DefaultTheme() *Theme {
	t := &Theme{
		Bold:   lipgloss.NewStyle().Bold(true),
		Cyan:   lipgloss.NewStyle().Foreground(lipgloss.Color("6")),
		Green:  lipgloss.NewStyle().Foreground(lipgloss.Color("2")),
		Yellow: lipgloss.NewStyle().Foreground(lipgloss.Color("3")),
		Dim:    lipgloss.NewStyle().Faint(true),
		Red:    lipgloss.NewStyle().Foreground(lipgloss.Color("1")),

		Bullet:  "•",
		Arrow:   "→",
		BoxTree: "├──",
		BoxLast: "└──",
		BoxItem: "│  ",

		IconPkg:   "📦",
		IconCave:  "🏔️",
		IconDisk:  "💾",
		IconWorld: "🌐",
		IconHelp:  "💡",
	}

	return t
}

func (t *Theme) Styled(style lipgloss.Style, text string) string {
	return style.Render(text)
}
