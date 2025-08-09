package theme

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

// CloudWorkstationTheme is a custom theme for the CloudWorkstation application
type CloudWorkstationTheme struct {
	baseTheme fyne.Theme
}

// CloudWorkstationColors defines the color palette for the application
var CloudWorkstationColors = struct {
	// Primary Colors
	PrimaryBlue   color.NRGBA
	SecondaryBlue color.NRGBA
	AccentGreen   color.NRGBA
	WarningYellow color.NRGBA
	ErrorRed      color.NRGBA

	// Neutral Colors
	DarkGray   color.NRGBA
	MediumGray color.NRGBA
	LightGray  color.NRGBA
	White      color.NRGBA

	// State Colors - Light Theme
	RunningLight   color.NRGBA
	StoppedLight   color.NRGBA
	PendingLight   color.NRGBA
	TerminatedLight color.NRGBA

	// State Colors - Dark Theme
	RunningDark   color.NRGBA
	StoppedDark   color.NRGBA
	PendingDark   color.NRGBA
	TerminatedDark color.NRGBA
}{
	// Primary Colors
	PrimaryBlue:   color.NRGBA{R: 25, G: 118, B: 210, A: 255}, // #1976D2
	SecondaryBlue: color.NRGBA{R: 33, G: 150, B: 243, A: 255}, // #2196F3
	AccentGreen:   color.NRGBA{R: 76, G: 175, B: 80, A: 255},  // #4CAF50
	WarningYellow: color.NRGBA{R: 255, G: 193, B: 7, A: 255},  // #FFC107
	ErrorRed:      color.NRGBA{R: 244, G: 67, B: 54, A: 255},  // #F44336

	// Neutral Colors
	DarkGray:   color.NRGBA{R: 51, G: 51, B: 51, A: 255},    // #333333
	MediumGray: color.NRGBA{R: 117, G: 117, B: 117, A: 255}, // #757575
	LightGray:  color.NRGBA{R: 238, G: 238, B: 238, A: 255}, // #EEEEEE
	White:      color.NRGBA{R: 255, G: 255, B: 255, A: 255}, // #FFFFFF

	// State Colors - Light Theme
	RunningLight:    color.NRGBA{R: 76, G: 175, B: 80, A: 255},  // #4CAF50
	StoppedLight:    color.NRGBA{R: 255, G: 193, B: 7, A: 255},  // #FFC107
	PendingLight:    color.NRGBA{R: 255, G: 152, B: 0, A: 255},  // #FF9800
	TerminatedLight: color.NRGBA{R: 244, G: 67, B: 54, A: 255},  // #F44336

	// State Colors - Dark Theme
	RunningDark:    color.NRGBA{R: 102, G: 187, B: 106, A: 255}, // #66BB6A
	StoppedDark:    color.NRGBA{R: 255, G: 202, B: 40, A: 255},  // #FFCA28
	PendingDark:    color.NRGBA{R: 255, G: 167, B: 38, A: 255},  // #FFA726
	TerminatedDark: color.NRGBA{R: 239, G: 83, B: 80, A: 255},   // #EF5350
}

// NewCloudWorkstationTheme creates a new CloudWorkstation theme
func NewCloudWorkstationTheme(isDarkMode bool) fyne.Theme {
	baseTheme := theme.DefaultTheme()
	if isDarkMode {
		baseTheme = theme.DarkTheme()
	} else {
		baseTheme = theme.LightTheme()
	}
	
	return &CloudWorkstationTheme{
		baseTheme: baseTheme,
	}
}

// Color returns the theme-appropriate color for the specified ColorName
func (t *CloudWorkstationTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	// Override default colors with our custom colors
	switch name {
	case theme.ColorNamePrimary:
		return CloudWorkstationColors.PrimaryBlue
	case theme.ColorNameForeground:
		if variant == theme.VariantDark {
			return CloudWorkstationColors.White
		}
		return CloudWorkstationColors.DarkGray
	case theme.ColorNameDisabled:
		if variant == theme.VariantDark {
			return CloudWorkstationColors.MediumGray
		}
		return CloudWorkstationColors.LightGray
	default:
		return t.baseTheme.Color(name, variant)
	}
}

// Icon returns the icon for the specified theme IconName
func (t *CloudWorkstationTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return t.baseTheme.Icon(name)
}

// Font returns the font resource for the specified text style
func (t *CloudWorkstationTheme) Font(style fyne.TextStyle) fyne.Resource {
	return t.baseTheme.Font(style)
}

// Size returns the size for the specified theme SizeName
func (t *CloudWorkstationTheme) Size(name fyne.ThemeSizeName) float32 {
	// Consistent sizing scheme for professional appearance
	switch name {
	case theme.SizeNameText:
		return 13 // Base font size - slightly smaller for better density
	case theme.SizeNameHeadingText:
		return 18 // Heading size - more proportional
	case theme.SizeNameSubText:
		return 11 // Sub-text size - for secondary information
	case theme.SizeNameCaptionText:
		return 11 // Caption size - consistent with sub-text
	case theme.SizeNameInlineIcon:
		return 16 // Icon size - proportional to text
	case theme.SizeNamePadding:
		return 6 // Base padding unit - tighter for professional look
	case theme.SizeNameScrollBar:
		return 12 // Scroll bar width - more visible
	case theme.SizeNameScrollBarSmall:
		return 6 // Small scroll bar width
	case theme.SizeNameSeparatorThickness:
		return 1 // Thin separators
	case theme.SizeNameInputBorder:
		return 1 // Consistent input borders
	default:
		return t.baseTheme.Size(name)
	}
}

// GetStateColor returns the appropriate color for an instance state based on theme variant
func GetStateColor(state string, variant fyne.ThemeVariant) color.Color {
	if variant == theme.VariantDark {
		switch state {
		case "running":
			return CloudWorkstationColors.RunningDark
		case "stopped":
			return CloudWorkstationColors.StoppedDark
		case "pending", "stopping":
			return CloudWorkstationColors.PendingDark
		case "terminated", "error":
			return CloudWorkstationColors.TerminatedDark
		default:
			return CloudWorkstationColors.MediumGray
		}
	} else {
		switch state {
		case "running":
			return CloudWorkstationColors.RunningLight
		case "stopped":
			return CloudWorkstationColors.StoppedLight
		case "pending", "stopping":
			return CloudWorkstationColors.PendingLight
		case "terminated", "error":
			return CloudWorkstationColors.TerminatedLight
		default:
			return CloudWorkstationColors.MediumGray
		}
	}
}

// GetStateEmoji returns the appropriate emoji for an instance state
func GetStateEmoji(state string) string {
	switch state {
	case "running":
		return "🟢"
	case "stopped":
		return "🟡"
	case "pending", "stopping":
		return "🟠"
	case "terminated", "error":
		return "🔴"
	default:
		return "⚫"
	}
}