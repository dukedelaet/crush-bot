package ui

import (
	"errors"
	"fmt"
	"io"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/hocoder-agents/crush-bot/internal/config"
	"github.com/hocoder-agents/crush-bot/internal/crush"
	"github.com/hocoder-agents/crush-bot/internal/daemon"
	"github.com/hocoder-agents/crush-bot/internal/envelope"
	"github.com/hocoder-agents/crush-bot/internal/roster"
	"github.com/hocoder-agents/crush-bot/internal/spawn"
)

const (
	helpRows   = 1
	dividerW   = 1
	minSideW   = 16
	maxSideW   = 28
	focusSide  = 0
	focusCrush = 1
)

var (
	titleStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	mutedStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	keyStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("212"))
	selStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("229")).Bold(true)
	sideStyle   = lipgloss.NewStyle().Padding(0, 1)
	helpStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Padding(0, 1)
	divStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("238"))
	divHotStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	paneStyle   = lipgloss.NewStyle()
	boxStyle    = lipgloss.NewStyle().Padding(1, 2)
)

type row struct {
	bot     roster.Bot
	pending int
	busy    bool
}

type Model struct {
	width, height int
	home          string
	rows          []row
	cursor        int
	status        string
	focus         int
	pane          *crushPane
}

func New(home string) Model {
	m := Model{home: home, focus: focusSide}
	m.reload()
	return m
}

func (m *Model) reload() {
	bots, err := roster.List(m.home, false)
	if err != nil {
		m.status = err.Error()
		m.rows = nil
		return
	}
	m.rows = m.rows[:0]
	for _, b := range bots {
		home := roster.Home(m.home, b.Slug)
		envs, _, _ := envelope.List(envelope.PendingDir(home))
		busy := false
		if t, err := crush.ReadTurn(home); err == nil {
			busy = crush.PIDAlive(t.CrushPID)
		}
		m.rows = append(m.rows, row{bot: b, pending: len(envs), busy: busy})
	}
	if m.cursor >= len(m.rows) {
		m.cursor = 0
	}
	if daemon.Live(m.home) {
		m.status = "daemon up"
	} else {
		m.status = "daemon down — crushbot daemon start"
	}
}

func (m Model) Init() tea.Cmd { return nil }

type spawnDoneMsg struct {
	err  error
	slug string
}

func layout(width, height int) (sideW, crushW, bodyH int) {
	if width <= 0 {
		width = 80
	}
	if height <= 0 {
		height = 24
	}
	bodyH = height - helpRows
	if bodyH < 4 {
		bodyH = 4
	}
	sideW = 24
	if width < 70 {
		sideW = minSideW
	}
	if sideW > maxSideW {
		sideW = maxSideW
	}
	if sideW > width/2 {
		sideW = width / 2
	}
	if sideW < minSideW && width > minSideW+10 {
		sideW = minSideW
	}
	crushW = width - sideW - dividerW
	if crushW < 10 {
		crushW = 10
		sideW = width - crushW - dividerW
		if sideW < 8 {
			sideW = 8
		}
	}
	return sideW, crushW, bodyH
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		_, crushW, bodyH := layout(m.width, m.height)
		if m.pane != nil {
			m.pane.resize(crushW, bodyH)
		}
	case paneDrawMsg:
		if m.pane != nil {
			return m, m.pane.listen()
		}
	case paneExitMsg:
		if m.pane != nil && (msg.slug == "" || msg.slug == m.pane.slug) {
			m.pane.close()
			m.pane = nil
			m.focus = focusSide
			m.reload()
			if msg.err != nil {
				m.status = "crush exited: " + msg.err.Error()
			} else {
				m.status = "crush closed"
			}
		}
	case spawnDoneMsg:
		m.reload()
		if msg.slug != "" {
			m.status = "spawned @" + msg.slug
			for i, r := range m.rows {
				if r.bot.Slug == msg.slug {
					m.cursor = i
					break
				}
			}
			break
		}
		switch {
		case msg.err != nil && (errors.Is(msg.err, spawn.ErrAborted) || msg.err.Error() == "cancelled"):
			m.status = "spawn cancelled"
		case msg.err != nil:
			m.status = "spawn failed: " + msg.err.Error()
		default:
			m.status = "spawned"
		}
	case tea.MouseClickMsg:
		return m.handleMouse("click", msg.Mouse())
	case tea.MouseReleaseMsg:
		return m.handleMouse("release", msg.Mouse())
	case tea.MouseWheelMsg:
		return m.handleMouse("wheel", msg.Mouse())
	case tea.MouseMotionMsg:
		return m.handleMouse("motion", msg.Mouse())
	case tea.KeyPressMsg:
		if msg.String() == "ctrl+g" {
			if m.pane != nil {
				if m.focus == focusCrush {
					m.focus = focusSide
				} else {
					m.focus = focusCrush
				}
			}
			return m, nil
		}
		if m.focus == focusCrush && m.pane != nil {
			m.pane.sendKey(msg)
			return m, nil
		}
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			if m.pane != nil {
				m.pane.close()
				m.pane = nil
			}
			return m, tea.Quit
		case "j", "down":
			if len(m.rows) > 0 {
				m.cursor = (m.cursor + 1) % len(m.rows)
			}
		case "k", "up":
			if len(m.rows) > 0 {
				m.cursor = (m.cursor - 1 + len(m.rows)) % len(m.rows)
			}
		case "r":
			m.reload()
		case "n":
			wiz := &spawnWizard{home: m.home}
			return m, tea.Exec(wiz, func(err error) tea.Msg {
				return spawnDoneMsg{err: err, slug: wiz.slug}
			})
		case "enter":
			return m.openSelected()
		}
	}
	return m, nil
}

func (m Model) handleMouse(kind string, mouse tea.Mouse) (tea.Model, tea.Cmd) {
	sideW, _, bodyH := layout(m.width, m.height)
	if mouse.Y >= bodyH {
		return m, nil
	}
	if mouse.X < sideW {
		m.focus = focusSide
		if kind == "click" {
			idx := mouse.Y - 4 // title, subtitle, status, blank
			if idx >= 0 && idx < len(m.rows) {
				m.cursor = idx
			}
		}
		return m, nil
	}
	if m.pane == nil {
		return m, nil
	}
	m.focus = focusCrush
	mx := mouse.X - sideW - dividerW
	my := mouse.Y
	if mx < 0 || my < 0 {
		return m, nil
	}
	m.pane.sendMouse(kind, tea.Mouse{X: mx, Y: my, Button: mouse.Button, Mod: mouse.Mod})
	return m, nil
}

func (m Model) openSelected() (tea.Model, tea.Cmd) {
	if len(m.rows) == 0 {
		return m, nil
	}
	bot := m.rows[m.cursor].bot
	if m.pane != nil && m.pane.slug == bot.Slug {
		m.focus = focusCrush
		return m, nil
	}
	if m.pane != nil {
		m.pane.close()
		m.pane = nil
	}
	_, crushW, bodyH := layout(m.width, m.height)
	pane, cmd, err := openCrushPane(m.home, bot, crushW, bodyH)
	if err != nil {
		m.status = "crush failed: " + err.Error()
		m.reload()
		return m, nil
	}
	m.pane = pane
	m.focus = focusCrush
	m.status = "crush @" + bot.Slug
	return m, cmd
}

func (m Model) View() tea.View {
	sideW, crushW, bodyH := layout(m.width, m.height)
	sidebar := m.sidebarView(sideW, bodyH)
	crushPane := m.crushView(crushW, bodyH)
	div := m.divider(bodyH)
	body := lipgloss.JoinHorizontal(lipgloss.Top, sidebar, div, crushPane)
	help := m.helpView(m.width)
	frame := lipgloss.JoinVertical(lipgloss.Left, body, help)
	v := tea.NewView(frame)
	v.AltScreen = true
	if m.focus == focusCrush && m.pane != nil {
		v.MouseMode = tea.MouseModeCellMotion
	}
	return v
}

func (m Model) sidebarView(width, height int) string {
	var b strings.Builder
	fmt.Fprintln(&b, titleStyle.Render("crushbot"))
	fmt.Fprintln(&b, mutedStyle.Render("Charm Crush Powered Bot Mesh"))
	fmt.Fprintln(&b, mutedStyle.Render(m.status))
	fmt.Fprintln(&b)
	if len(m.rows) == 0 {
		fmt.Fprintln(&b, "No bots yet.")
		fmt.Fprintln(&b, mutedStyle.Render("press n to spawn"))
	}
	for i, r := range m.rows {
		mark := " "
		if i == m.cursor {
			mark = "▸"
		}
		open := ""
		if m.pane != nil && m.pane.slug == r.bot.Slug {
			open = " ●"
		}
		busy := ""
		if r.busy {
			busy = " busy"
		}
		line := fmt.Sprintf("%s @%s%s%s", mark, r.bot.Slug, open, busy)
		if r.pending > 0 {
			line += fmt.Sprintf("  %d", r.pending)
		}
		if i == m.cursor && m.focus == focusSide {
			line = selStyle.Render(line)
		}
		fmt.Fprintln(&b, line)
	}
	return sideStyle.Width(width).Height(height).MaxHeight(height).MaxWidth(width).Render(b.String())
}

func (m Model) crushView(width, height int) string {
	var body string
	if m.pane != nil {
		body = m.pane.render()
	} else if len(m.rows) == 0 {
		body = mutedStyle.Render("spawn a bot, then press enter")
	} else {
		body = mutedStyle.Render("press enter to open Crush")
	}
	return paneStyle.Width(width).Height(height).MaxHeight(height).MaxWidth(width).Render(body)
}

func (m Model) divider(height int) string {
	st := divStyle
	if m.focus == focusCrush {
		st = divHotStyle
	}
	col := strings.Repeat("│\n", height)
	col = strings.TrimSuffix(col, "\n")
	return st.Width(dividerW).Height(height).Render(col)
}

func (m Model) helpView(width int) string {
	var s string
	if m.focus == focusCrush && m.pane != nil {
		s = fmt.Sprintf("%s sidebar   keys go to Crush", keyStyle.Render("ctrl+g"))
	} else {
		s = fmt.Sprintf("%s move  %s open  %s crush  %s new  %s refresh  %s quit",
			keyStyle.Render("j/k"), keyStyle.Render("enter"), keyStyle.Render("ctrl+g"),
			keyStyle.Render("n"), keyStyle.Render("r"), keyStyle.Render("q"))
	}
	return helpStyle.Width(width).MaxWidth(width).Render(s)
}

type spawnWizard struct {
	home string
	slug string
	in   io.Reader
	out  io.Writer
	err  io.Writer
}

func (w *spawnWizard) Run() error {
	p := config.ResolvePaths()
	cfg, err := config.Load(p)
	if err != nil {
		return err
	}
	if err := config.EnsureHome(p); err != nil {
		return err
	}
	tty, err := spawn.OpenTTY()
	if err != nil {
		return fmt.Errorf("spawn form needs a terminal: %w", err)
	}
	defer tty.Close()
	res, err := spawn.FromFormAccessible(w.home, cfg, tty, tty)
	if err != nil {
		return err
	}
	w.slug = res.Bot.Slug
	return nil
}

func (w *spawnWizard) SetStdin(r io.Reader)  { w.in = r }
func (w *spawnWizard) SetStdout(o io.Writer) { w.out = o }
func (w *spawnWizard) SetStderr(e io.Writer) { w.err = e }

func Run(home string) error {
	p := tea.NewProgram(New(home))
	_, err := p.Run()
	return err
}
