package ui

import "github.com/charmbracelet/lipgloss"

// Color palette
var (
	// Primary colors
	ColorPrimary   = lipgloss.Color("#7C3AED") // Purple
	ColorSecondary = lipgloss.Color("#06B6D4") // Cyan
	ColorAccent    = lipgloss.Color("#10B981") // Green
	ColorDanger    = lipgloss.Color("#EF4444") // Red
	ColorWarning   = lipgloss.Color("#F59E0B") // Amber

	// Neutral colors
	ColorText    = lipgloss.Color("#E5E7EB") // Light gray
	ColorTextDim = lipgloss.Color("#9CA3AF") // Medium gray
	ColorBorder  = lipgloss.Color("#4B5563") // Dark gray
	ColorBg      = lipgloss.Color("#1F2937") // Very dark gray
	ColorBgAlt   = lipgloss.Color("#111827") // Almost black
)

// Styles defines all UI styles
type Styles struct {
	// Layout
	App         lipgloss.Style
	Header      lipgloss.Style
	Footer      lipgloss.Style
	Panel       lipgloss.Style
	PanelBorder lipgloss.Style

	// List
	ListItem         lipgloss.Style
	ListItemSelected lipgloss.Style
	ListHeader       lipgloss.Style

	// Details
	DetailLabel lipgloss.Style
	DetailValue lipgloss.Style
	DetailRow   lipgloss.Style

	// Status indicators
	StatusReady   lipgloss.Style
	StatusWarning lipgloss.Style
	StatusError   lipgloss.Style

	// Help/Keybindings
	HelpKey   lipgloss.Style
	HelpValue lipgloss.Style
	HelpSep   lipgloss.Style

	// Modal
	Modal        lipgloss.Style
	ModalTitle   lipgloss.Style
	ModalOverlay lipgloss.Style

	// Spotlight
	Spotlight         lipgloss.Style
	SpotlightInput    lipgloss.Style
	SpotlightItem     lipgloss.Style
	SpotlightSelected lipgloss.Style

	// Login
	LoginBox lipgloss.Style

	// Form
	FormLabel         lipgloss.Style
	FormInput         lipgloss.Style
	FormInputFocused  lipgloss.Style
	FormButton        lipgloss.Style
	FormButtonFocused lipgloss.Style

	// General
	Title    lipgloss.Style
	Subtitle lipgloss.Style
	Error    lipgloss.Style
	Success  lipgloss.Style
	Info     lipgloss.Style
	Warning  lipgloss.Style
}

// NewStyles creates a new Styles instance with default styling
func NewStyles() *Styles {
	s := &Styles{}

	// Layout
	s.App = lipgloss.NewStyle().
		Padding(0)

	s.Header = lipgloss.NewStyle().
		Foreground(ColorPrimary).
		Bold(true).
		Padding(0, 1).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(ColorBorder).
		BorderBottom(true)

	s.Footer = lipgloss.NewStyle().
		Foreground(ColorTextDim).
		Padding(0, 1).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(ColorBorder).
		BorderTop(true)

	s.Panel = lipgloss.NewStyle().
		Padding(1, 2)

	s.PanelBorder = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(ColorBorder).
		Padding(1, 2)

	// List
	s.ListItem = lipgloss.NewStyle().
		Foreground(ColorText).
		Padding(0, 2)

	s.ListItemSelected = lipgloss.NewStyle().
		Foreground(ColorBgAlt).
		Background(ColorPrimary).
		Bold(true).
		Padding(0, 2)

	s.ListHeader = lipgloss.NewStyle().
		Foreground(ColorSecondary).
		Bold(true).
		Underline(true).
		Padding(0, 2).
		MarginBottom(1)

	// Details
	s.DetailLabel = lipgloss.NewStyle().
		Foreground(ColorTextDim).
		Width(15).
		Align(lipgloss.Right).
		MarginRight(1)

	s.DetailValue = lipgloss.NewStyle().
		Foreground(ColorText).
		Bold(true)

	s.DetailRow = lipgloss.NewStyle().
		MarginBottom(0)

	// Status indicators
	s.StatusReady = lipgloss.NewStyle().
		Foreground(ColorAccent).
		Bold(true)

	s.StatusWarning = lipgloss.NewStyle().
		Foreground(ColorWarning).
		Bold(true)

	s.StatusError = lipgloss.NewStyle().
		Foreground(ColorDanger).
		Bold(true)

	// Help/Keybindings
	s.HelpKey = lipgloss.NewStyle().
		Foreground(ColorSecondary).
		Bold(true)

	s.HelpValue = lipgloss.NewStyle().
		Foreground(ColorTextDim)

	s.HelpSep = lipgloss.NewStyle().
		Foreground(ColorBorder).
		SetString(" • ")

	// Modal
	s.Modal = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(ColorPrimary).
		Padding(1, 2).
		Width(60)

	s.ModalTitle = lipgloss.NewStyle().
		Foreground(ColorPrimary).
		Bold(true).
		Padding(0, 0, 1, 0)

	s.ModalOverlay = lipgloss.NewStyle().
		Background(lipgloss.Color("#000000"))

	// Spotlight
	s.Spotlight = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(ColorSecondary).
		Padding(1, 0).
		Width(60)

	s.SpotlightInput = lipgloss.NewStyle().
		Foreground(ColorText).
		Padding(0, 1).
		BorderBottom(true).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(ColorBorder).
		MarginBottom(1)

	s.SpotlightItem = lipgloss.NewStyle().
		Foreground(ColorTextDim).
		Padding(0, 2)

	s.SpotlightSelected = lipgloss.NewStyle().
		Foreground(ColorBgAlt).
		Background(ColorSecondary).
		Bold(true).
		Padding(0, 2)

	// Login
	s.LoginBox = lipgloss.NewStyle().
		BorderStyle(lipgloss.DoubleBorder()).
		BorderForeground(ColorPrimary).
		Padding(1, 2).
		Width(50).
		Align(lipgloss.Center)

	// Form
	s.FormLabel = lipgloss.NewStyle().
		Foreground(ColorTextDim).
		Width(15).
		Align(lipgloss.Right).
		MarginRight(1)

	s.FormInput = lipgloss.NewStyle().
		Foreground(ColorText).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(ColorBorder).
		Padding(0, 1)

	s.FormInputFocused = lipgloss.NewStyle().
		Foreground(ColorText).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(ColorPrimary).
		Padding(0, 1)

	s.FormButton = lipgloss.NewStyle().
		Foreground(ColorText).
		Background(ColorBg).
		Padding(0, 2).
		MarginRight(1)

	s.FormButtonFocused = lipgloss.NewStyle().
		Foreground(ColorBgAlt).
		Background(ColorPrimary).
		Bold(true).
		Padding(0, 2).
		MarginRight(1)

	// General
	s.Title = lipgloss.NewStyle().
		Foreground(ColorPrimary).
		Bold(true).
		MarginBottom(1)

	s.Subtitle = lipgloss.NewStyle().
		Foreground(ColorTextDim).
		Italic(true)

	s.Error = lipgloss.NewStyle().
		Foreground(ColorDanger).
		Bold(true)

	s.Success = lipgloss.NewStyle().
		Foreground(ColorAccent).
		Bold(true)

	s.Info = lipgloss.NewStyle().
		Foreground(ColorSecondary).
		Bold(true)

	s.Warning = lipgloss.NewStyle().
		Foreground(ColorWarning).
		Bold(true)

	return s
}
