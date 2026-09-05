package cli

import "charm.land/lipgloss/v2"

var (
	errStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("203")).Bold(true)
	okStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	cmdStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("212"))
	headStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	mutedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
)
