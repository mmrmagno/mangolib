package ui

import (
	"fmt"
	"os"

	"github.com/charmbracelet/lipgloss"
)

// Verbose disables styled UI and passes subprocess output through directly.
var Verbose bool

// Catppuccin Mocha palette
const (
	colorPeach    = "#fab387"
	colorGreen    = "#a6e3a1"
	colorRed      = "#f38ba8"
	colorMauve    = "#cba6f7"
	colorYellow   = "#f9e2af"
	colorLavender = "#b4befe"
	colorOverlay  = "#9399b2"
	colorText     = "#cdd6f4"
)

var (
	styleBannerBox = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(colorPeach)).
			Padding(0, 2)

	styleBannerTitle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color(colorPeach))

	styleStep = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorMauve))

	styleSuccess = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorGreen))

	styleWarn = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorYellow))

	styleFatal = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(colorRed))

	styleDim = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorOverlay))
)

func Banner(version string) {
	title := styleBannerTitle.Render("🥭 mangolib") +
		"  " + styleDim.Render(version)
	sub := styleDim.Render("music · download · sync")
	fmt.Println(styleBannerBox.Render(title + "\n" + sub))
}

func Step(msg string) {
	fmt.Println(styleStep.Render("→ " + msg))
}

func Success(msg string) {
	fmt.Println(styleSuccess.Render("✓ " + msg))
}

func Warn(msg string) {
	fmt.Fprintln(os.Stderr, styleWarn.Render("! "+msg))
}

func Fatal(msg string) {
	fmt.Fprintln(os.Stderr, styleFatal.Render("✗ "+msg))
	os.Exit(1)
}

func Fatalf(format string, args ...any) {
	Fatal(fmt.Sprintf(format, args...))
}

// Dim returns muted text for secondary information.
func Dim(msg string) string {
	return styleDim.Render(msg)
}
