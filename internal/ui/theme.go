package ui

import (
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-isatty"
	"github.com/muesli/termenv"
)

// ANSI palette indexes are intentional here. Portico supplies semantic roles;
// the terminal theme decides the actual displayed shades.
const (
	ansiError   = "1" // red
	ansiSuccess = "2" // green
	ansiWarning = "3" // yellow
	ansiAccent  = "4" // blue
	ansiInfo    = "6" // cyan
	ansiMuted   = "8" // bright black / theme-defined muted foreground
)

type SemanticStyles struct {
	Success  lipgloss.Style
	Error    lipgloss.Style
	Warning  lipgloss.Style
	Info     lipgloss.Style
	Accent   lipgloss.Style
	Muted    lipgloss.Style
	Selected lipgloss.Style
	Disabled lipgloss.Style
}

func CurrentStyles() SemanticStyles {
	if !ColorEnabled() {
		plain := lipgloss.NewStyle()
		return SemanticStyles{
			Success:  plain,
			Error:    plain,
			Warning:  plain,
			Info:     plain,
			Accent:   plain,
			Muted:    plain,
			Selected: plain,
			Disabled: plain,
		}
	}

	return SemanticStyles{
		Success:  lipgloss.NewStyle().Foreground(lipgloss.Color(ansiSuccess)),
		Error:    lipgloss.NewStyle().Foreground(lipgloss.Color(ansiError)),
		Warning:  lipgloss.NewStyle().Foreground(lipgloss.Color(ansiWarning)),
		Info:     lipgloss.NewStyle().Foreground(lipgloss.Color(ansiInfo)),
		Accent:   lipgloss.NewStyle().Foreground(lipgloss.Color(ansiAccent)),
		Muted:    lipgloss.NewStyle().Faint(true),
		Selected: lipgloss.NewStyle().Foreground(lipgloss.Color(ansiAccent)).Bold(true),
		Disabled: lipgloss.NewStyle().Faint(true),
	}
}

func Success(text string) string {
	return CurrentStyles().Success.Render(text)
}

func Error(text string) string {
	return CurrentStyles().Error.Render(text)
}

func Warning(text string) string {
	return CurrentStyles().Warning.Render(text)
}

func Info(text string) string {
	return CurrentStyles().Info.Render(text)
}

func Accent(text string) string {
	return CurrentStyles().Accent.Render(text)
}

func Muted(text string) string {
	return CurrentStyles().Muted.Render(text)
}

func Selected(text string) string {
	return CurrentStyles().Selected.Render(text)
}

func Disabled(text string) string {
	return CurrentStyles().Disabled.Render(text)
}

func ColorEnabled() bool {
	return colorEnabledForTerminal(stdoutIsTerminal())
}

func colorEnabledForTerminal(isTerminal bool) bool {
	if !isTerminal {
		return false
	}

	if _, disabled := os.LookupEnv("NO_COLOR"); disabled {
		return false
	}

	if strings.EqualFold(strings.TrimSpace(os.Getenv("TERM")), "dumb") {
		return false
	}

	if strings.TrimSpace(os.Getenv("CLICOLOR")) == "0" {
		return false
	}

	return true
}

func stdoutIsTerminal() bool {
	fd := os.Stdout.Fd()
	return isatty.IsTerminal(fd) || isatty.IsCygwinTerminal(fd)
}

func newSemanticProgress() progress.Model {
	options := []progress.Option{
		progress.WithSolidFill(ansiAccent),
	}

	if !ColorEnabled() {
		options = append(options, progress.WithColorProfile(termenv.Ascii))
	}

	bar := progress.New(options...)
	bar.EmptyColor = ansiMuted
	bar.PercentageStyle = CurrentStyles().Muted

	return bar
}

func styleSpinner(spinStyle *lipgloss.Style) {
	if spinStyle == nil {
		return
	}

	*spinStyle = CurrentStyles().Accent
}
