package ui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/dukedelaet/crush-bot/internal/roster"
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("205"))
	mutedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("245"))
	keyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("212"))
	boxStyle = lipgloss.NewStyle().
			Padding(1, 2)
)

// Model is the v1 mesh placeholder (empty roster).
type Model struct {
	width  int
	height int
	home   string
}

func New(home string) Model {
	return Model{home: home}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case tea.KeyPressMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m Model) View() tea.View {
	var b strings.Builder
	fmt.Fprintln(&b, titleStyle.Render("crushbot"))
	fmt.Fprintln(&b, mutedStyle.Render("Hermes-style bot roster · Crush backend"))
	fmt.Fprintln(&b)
	bots, err := roster.List(m.home, false)
	if err != nil {
		fmt.Fprintln(&b, err.Error())
	} else if len(bots) == 0 {
		fmt.Fprintln(&b, "No bots yet.")
		fmt.Fprintln(&b, mutedStyle.Render("crushbot spawn <slug>"))
	} else {
		for _, bot := range bots {
			line := keyStyle.Render(bot.Slug) + "  " + bot.Title
			if bot.Description != "" {
				line += "  " + mutedStyle.Render(bot.Description)
			}
			fmt.Fprintln(&b, line)
		}
	}
	if m.home != "" {
		fmt.Fprintln(&b)
		fmt.Fprintln(&b, mutedStyle.Render("home  "+m.home))
	}
	fmt.Fprintln(&b)
	fmt.Fprintf(&b, "%s quit\n", keyStyle.Render("q"))
	body := b.String()
	if m.width > 0 {
		body = boxStyle.Width(m.width).Render(body)
	} else {
		body = boxStyle.Render(body)
	}
	v := tea.NewView(body)
	v.AltScreen = true
	return v
}

func Run(home string) error {
	p := tea.NewProgram(New(home))
	_, err := p.Run()
	return err
}
