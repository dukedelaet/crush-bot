package ui

import (
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/hocoder-agents/crush-bot/internal/version"
)

var (
	pink     = lipgloss.Color("#FF6BB5")
	lavender = lipgloss.Color("#C4B5FD")
)

func brandLine() string {
	heart := "💗"
	name := gradientText("crushbot")
	ver := mutedStyle.Foreground(lavender).Render(version.Version)
	return heart + " " + name + "  " + ver
}

func gradientText(s string) string {
	runes := []rune(s)
	if len(runes) == 0 {
		return ""
	}
	stops := lipgloss.Blend1D(len(runes), pink, lavender)
	bold := lipgloss.NewStyle().Bold(true)
	var b strings.Builder
	for i, r := range runes {
		b.WriteString(bold.Foreground(stops[i]).Render(string(r)))
	}
	return b.String()
}
