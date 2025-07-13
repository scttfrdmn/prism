package styles

import (
	"github.com/charmbracelet/lipgloss"
)

// Theme contains all styling definitions for the TUI
type Theme struct {
	// Colors
	PrimaryColor   lipgloss.Color
	SecondaryColor lipgloss.Color
	AccentColor    lipgloss.Color
	ErrorColor     lipgloss.Color
	SuccessColor   lipgloss.Color
	WarningColor   lipgloss.Color
	TextColor      lipgloss.Color
	MutedColor     lipgloss.Color
	BorderColor    lipgloss.Color
	
	// Styles
	Title       lipgloss.Style
	Subtitle    lipgloss.Style
	SectionTitle lipgloss.Style
	SubTitle    lipgloss.Style
	Panel       lipgloss.Style
	PanelHeader lipgloss.Style
	TableHeader lipgloss.Style
	TableRow    lipgloss.Style
	Button      lipgloss.Style
	ActiveButton lipgloss.Style
	Label       lipgloss.Style
	InputField  lipgloss.Style
	StatusOK    lipgloss.Style
	StatusError lipgloss.Style
	StatusWarning lipgloss.Style
	Warning     lipgloss.Style
	Help        lipgloss.Style
	Pagination  lipgloss.Style
}

// DefaultTheme returns the default CloudWorkstation theme
func DefaultTheme() Theme {
	// Colors
	primaryColor := lipgloss.Color("#0074D9")   // Blue
	secondaryColor := lipgloss.Color("#7FDBFF") // Light blue
	accentColor := lipgloss.Color("#FF851B")    // Orange
	errorColor := lipgloss.Color("#FF4136")     // Red
	successColor := lipgloss.Color("#2ECC40")   // Green
	warningColor := lipgloss.Color("#FFDC00")   // Yellow
	textColor := lipgloss.Color("#FFFFFF")      // White
	mutedColor := lipgloss.Color("#AAAAAA")     // Light gray
	borderColor := lipgloss.Color("#555555")    // Gray
	
	t := Theme{
		// Colors
		PrimaryColor:   primaryColor,
		SecondaryColor: secondaryColor,
		AccentColor:    accentColor,
		ErrorColor:     errorColor,
		SuccessColor:   successColor,
		WarningColor:   warningColor,
		TextColor:      textColor,
		MutedColor:     mutedColor,
		BorderColor:    borderColor,
		
		// Styles
		Title: lipgloss.NewStyle().
			Foreground(primaryColor).
			Bold(true).
			MarginBottom(1),
		
		Subtitle: lipgloss.NewStyle().
			Foreground(secondaryColor).
			Bold(true),
		
		Panel: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(borderColor).
			Padding(1, 2).
			MarginRight(1).
			MarginBottom(1),
		
		PanelHeader: lipgloss.NewStyle().
			Foreground(primaryColor).
			Bold(true).
			BorderBottom(true).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(borderColor).
			MarginBottom(1).
			Padding(0, 1),
		
		TableHeader: lipgloss.NewStyle().
			Foreground(textColor).
			Background(primaryColor).
			Bold(true).
			Padding(0, 1),
		
		TableRow: lipgloss.NewStyle().
			Padding(0, 1),
		
		Button: lipgloss.NewStyle().
			Foreground(textColor).
			Background(primaryColor).
			Bold(true).
			Padding(0, 3).
			MarginRight(1),
		
		ActiveButton: lipgloss.NewStyle().
			Foreground(textColor).
			Background(accentColor).
			Bold(true).
			Padding(0, 3).
			MarginRight(1),
		
		Label: lipgloss.NewStyle().
			Foreground(textColor).
			Bold(true).
			MarginRight(1),
		
		InputField: lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(borderColor).
			Padding(0, 1),
		
		StatusOK: lipgloss.NewStyle().
			Foreground(successColor).
			Bold(true),
		
		StatusError: lipgloss.NewStyle().
			Foreground(errorColor).
			Bold(true),
		
		StatusWarning: lipgloss.NewStyle().
			Foreground(warningColor).
			Bold(true),
		
		Warning: lipgloss.NewStyle().
			Foreground(warningColor).
			Bold(true),
		
		Help: lipgloss.NewStyle().
			Foreground(mutedColor).
			Italic(true),
			
		SectionTitle: lipgloss.NewStyle().
			Foreground(primaryColor).
			Bold(true).
			Underline(true).
			MarginBottom(1),
			
		SubTitle: lipgloss.NewStyle().
			Foreground(secondaryColor).
			Bold(true),
			
		Pagination: lipgloss.NewStyle().
			Foreground(mutedColor),
	}
	
	return t
}

// CurrentTheme holds the active theme
var CurrentTheme = DefaultTheme()

// DarkMode returns a dark mode theme
func DarkMode() Theme {
	theme := DefaultTheme()
	// Adjust colors for dark mode
	return theme
}

// LightMode returns a light mode theme
func LightMode() Theme {
	theme := DefaultTheme()
	
	// Invert some colors for light mode
	theme.TextColor = lipgloss.Color("#333333")      // Dark gray
	theme.MutedColor = lipgloss.Color("#777777")     // Medium gray
	theme.BorderColor = lipgloss.Color("#CCCCCC")    // Light gray
	
	// Update styles that use TextColor
	theme.Label = theme.Label.Copy().Foreground(theme.TextColor)
	
	return theme
}

// SetTheme changes the current theme
func SetTheme(theme Theme) {
	CurrentTheme = theme
}
