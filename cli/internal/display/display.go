// Package display owns every styled byte the CLI writes to a terminal. Callers
// pass text and pick a line kind; no other package composes ANSI escapes.
package display

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	Title   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	Success = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	Error   = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	Warn    = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
	Dim     = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	Logo    = lipgloss.NewStyle().Foreground(lipgloss.Color("12"))
)

func versionSuffix(version string) string {
	if version == "" || version == "dev" {
		return ""
	}
	return " " + version
}

func Banner(version string) string {
	return Title.Render("Archcore"+versionSuffix(version)) + Dim.Render(" — Git-native context for AI coding agents")
}

func WelcomeBanner(version string) string {
	logoLines := []string{
		"╔══════╗",
		"║      ║",
		"║  ╔═══╣",
		"║  ║ ╔═╣",
		"╚══╩═╩═╝",
	}
	logo := Logo.Render(strings.Join(logoLines, "\n"))

	textLines := []string{
		Title.Render("Archcore" + versionSuffix(version) + " — Git-native context for AI coding agents"),
		Dim.Render("Spec-driven development & context engineering"),
		Dim.Render("https://archcore.ai"),
	}
	text := strings.Join(textLines, "\n")

	return lipgloss.JoinHorizontal(lipgloss.Center, logo, "   ", text)
}

func CheckLine(msg string) string {
	return Success.Render("  ✓ ") + msg
}

func FailLine(msg string) string {
	return Error.Render("  ✗ ") + msg
}

func WarnLine(msg string) string {
	return Warn.Render("  ! ") + msg
}

func HintLine(msg string) string {
	return Dim.Render("    → ") + Dim.Render(msg)
}

func KeyValue(key, value string) string {
	return fmt.Sprintf("  %s %s", Dim.Render(key+":"), value)
}

// HookConnectedLine is the SessionStart banner. A non-zero globalDocCount
// appends the mounted total so the user-facing count and the GLOBALS block in
// the context agree (session-globals-disclosure.spec clause 15).
func HookConnectedLine(version string, docCount, globalDocCount int) string {
	if globalDocCount > 0 {
		return fmt.Sprintf("Archcore %s · MCP connected · %d docs + %d global", version, docCount, globalDocCount)
	}
	return fmt.Sprintf("Archcore %s · MCP connected · %d docs", version, docCount)
}
