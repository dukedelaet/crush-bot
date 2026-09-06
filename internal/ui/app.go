package ui

import (
	"context"
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

var (
	titleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	mutedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	keyStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("212"))
	selStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("229")).Bold(true)
	boxStyle   = lipgloss.NewStyle().Padding(1, 2)
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
}

func New(home string) Model {
	m := Model{home: home}
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

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case spawnDoneMsg:
		m.reload()
		// tea.Exec reports RestoreTerminal errors even when spawn succeeded.
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
	case tea.KeyPressMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
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
			if len(m.rows) == 0 {
				break
			}
			bot := m.rows[m.cursor].bot
			p := config.ResolvePaths()
			cfg, err := config.Load(p)
			if err != nil {
				m.status = err.Error()
				break
			}
			bin, err := crush.LookPath(cfg.CrushPath)
			if err != nil {
				m.status = err.Error()
				break
			}
			err = crush.Chat(context.Background(), crush.RunOpts{Bot: bot, Root: m.home, Bin: bin, Timeout: cfg.TurnLockTimeout})
			if err != nil {
				m.status = err.Error()
			}
			m.reload()
		}
	}
	return m, nil
}

func (m Model) View() tea.View {
	var b strings.Builder
	fmt.Fprintln(&b, titleStyle.Render("crushbot"))
	fmt.Fprintln(&b, mutedStyle.Render("Charm Crush Powered Bot Mesh"))
	fmt.Fprintln(&b, mutedStyle.Render(m.status))
	fmt.Fprintln(&b)
	if len(m.rows) == 0 {
		fmt.Fprintln(&b, "No bots yet.")
		fmt.Fprintln(&b, mutedStyle.Render("press n, or crushbot spawn <slug>"))
	}
	for i, r := range m.rows {
		mark := " "
		if i == m.cursor {
			mark = "▸"
		}
		busy := ""
		if r.busy {
			busy = " busy"
		}
		line := fmt.Sprintf("%s @%s  %s  pending %d%s", mark, r.bot.Slug, r.bot.Title, r.pending, busy)
		if i == m.cursor {
			line = selStyle.Render(line)
		}
		fmt.Fprintln(&b, line)
	}
	fmt.Fprintln(&b)
	fmt.Fprintf(&b, "%s move  %s chat  %s new  %s refresh  %s quit\n",
		keyStyle.Render("j/k"), keyStyle.Render("enter"), keyStyle.Render("n"), keyStyle.Render("r"), keyStyle.Render("q"))
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
	// Never feed tea.Exec's cancelreader to Huh — that aborts the form
	// before Create runs, so refresh has nothing to list.
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
