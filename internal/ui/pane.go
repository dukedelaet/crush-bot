package ui

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/vt"
	"github.com/creack/pty"

	"github.com/hocoder-agents/crush-bot/internal/config"
	"github.com/hocoder-agents/crush-bot/internal/crush"
	"github.com/hocoder-agents/crush-bot/internal/roster"
)

type paneDrawMsg struct{}

type paneExitMsg struct {
	slug string
	err  error
	snap string
}

type crushPane struct {
	slug   string
	master *os.File
	em     *vt.SafeEmulator
	chat   *crush.ChatSession
	msgs   chan tea.Msg
	done   chan struct{}
	w, h   int
	closed bool
	dead   bool
}

func openCrushPane(home string, bot roster.Bot, w, h int) (*crushPane, tea.Cmd, error) {
	if w < 24 {
		w = 24
	}
	if h < 12 {
		h = 12
	}
	p := config.ResolvePaths()
	cfg, err := config.Load(p)
	if err != nil {
		return nil, nil, err
	}
	bin, err := crush.LookPath(cfg.CrushPath)
	if err != nil {
		return nil, nil, err
	}
	chat, err := crush.BeginChat(crush.RunOpts{Bot: bot, Root: home, Bin: bin, Timeout: cfg.TurnLockTimeout})
	if err != nil {
		return nil, nil, err
	}
	cmd := chat.Cmd
	master, err := pty.StartWithSize(cmd, &pty.Winsize{Rows: uint16(h), Cols: uint16(w)})
	if err != nil {
		chat.Close()
		return nil, nil, err
	}
	if cmd.Process != nil {
		_ = crush.UpdatePID(chat.Home, cmd.Process.Pid)
	}
	pane := &crushPane{
		slug:   bot.Slug,
		master: master,
		em:     vt.NewSafeEmulator(w, h),
		chat:   chat,
		msgs:   make(chan tea.Msg, 8),
		done:   make(chan struct{}),
		w:      w,
		h:      h,
	}
	go pane.copyOut()
	go pane.copyIn()
	go pane.reap()
	return pane, tea.Batch(pane.listen(), paneTick()), nil
}

func paneTick() tea.Cmd {
	return tea.Tick(50*time.Millisecond, func(time.Time) tea.Msg { return paneTickMsg{} })
}

type paneTickMsg struct{}

func (p *crushPane) listen() tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-p.msgs
		if !ok {
			return paneExitMsg{slug: p.slug}
		}
		return msg
	}
}

func (p *crushPane) copyIn() {
	if p.em == nil || p.master == nil {
		return
	}
	_, _ = io.Copy(p.master, p.em)
}

func (p *crushPane) copyOut() {
	buf := make([]byte, 8192)
	for {
		n, err := p.master.Read(buf)
		if n > 0 {
			_, _ = p.em.Write(buf[:n])
			select {
			case p.msgs <- paneDrawMsg{}:
			default:
			}
		}
		if err != nil {
			return
		}
	}
}

func (p *crushPane) reap() {
	defer close(p.done)
	err := p.chat.Cmd.Wait()
	if p.chat != nil {
		p.chat.Close()
	}
	snap := ""
	if p.em != nil {
		snap = p.em.String()
	}
	select {
	case p.msgs <- paneExitMsg{slug: p.slug, err: err, snap: snap}:
	default:
	}
}

func (p *crushPane) sendKey(msg tea.KeyPressMsg) {
	if p == nil || p.master == nil || p.dead {
		return
	}
	_, _ = p.master.Write(encodeKey(msg))
}

func (p *crushPane) sendMouse(kind string, m tea.Mouse) {
	if p == nil || p.master == nil || p.dead {
		return
	}
	_, _ = p.master.Write(encodeMouse(kind, m))
}

func (p *crushPane) resize(w, h int) {
	if p == nil || p.master == nil || w < 8 || h < 8 {
		return
	}
	if w == p.w && h == p.h {
		return
	}
	p.w, p.h = w, h
	p.em.Resize(w, h)
	_ = pty.Setsize(p.master, &pty.Winsize{Rows: uint16(h), Cols: uint16(w)})
}

func (p *crushPane) render() string {
	if p == nil || p.em == nil {
		return ""
	}
	s := p.em.Render()
	if strings.TrimSpace(p.em.String()) == "" {
		return ""
	}
	return s
}

func (p *crushPane) close() {
	if p == nil || p.closed {
		return
	}
	p.closed = true
	if p.chat != nil && p.chat.Cmd != nil && p.chat.Cmd.Process != nil && !p.dead {
		_ = p.chat.Cmd.Process.Signal(syscall.SIGINT)
	}
	if p.master != nil {
		_ = p.master.Close()
		p.master = nil
	}
	if p.done != nil {
		select {
		case <-p.done:
		case <-time.After(2 * time.Second):
			if p.chat != nil && p.chat.Cmd != nil && p.chat.Cmd.Process != nil {
				_ = p.chat.Cmd.Process.Kill()
			}
			<-p.done
		}
	}
	if p.em != nil {
		_ = p.em.Close()
		p.em = nil
	}
}

func encodeKey(msg tea.KeyPressMsg) []byte {
	k := msg.Key()
	if k.Text != "" && k.Mod == 0 {
		return []byte(k.Text)
	}
	switch k.Code {
	case tea.KeyEnter:
		return []byte("\r")
	case tea.KeyTab:
		if k.Mod&tea.ModShift != 0 {
			return []byte("\x1b[Z")
		}
		return []byte("\t")
	case tea.KeyBackspace:
		return []byte{0x7f}
	case tea.KeyEscape:
		return []byte{0x1b}
	case tea.KeyUp:
		return []byte("\x1b[A")
	case tea.KeyDown:
		return []byte("\x1b[B")
	case tea.KeyRight:
		return []byte("\x1b[C")
	case tea.KeyLeft:
		return []byte("\x1b[D")
	case tea.KeyHome:
		return []byte("\x1b[H")
	case tea.KeyEnd:
		return []byte("\x1b[F")
	case tea.KeyPgUp:
		return []byte("\x1b[5~")
	case tea.KeyPgDown:
		return []byte("\x1b[6~")
	case tea.KeyInsert:
		return []byte("\x1b[2~")
	case tea.KeyDelete:
		return []byte("\x1b[3~")
	}
	if k.Mod&tea.ModCtrl != 0 && k.Code >= 'a' && k.Code <= 'z' {
		return []byte{byte(k.Code - 'a' + 1)}
	}
	if k.Code > 0 && utf8.ValidRune(k.Code) && k.Mod == 0 {
		return []byte(string(k.Code))
	}
	if k.Text != "" {
		return []byte(k.Text)
	}
	return nil
}

func encodeMouse(kind string, m tea.Mouse) []byte {
	btn := 0
	switch m.Button {
	case tea.MouseLeft:
		btn = 0
	case tea.MouseMiddle:
		btn = 1
	case tea.MouseRight:
		btn = 2
	case tea.MouseWheelUp:
		btn = 64
	case tea.MouseWheelDown:
		btn = 65
	default:
		btn = 0
	}
	if kind == "motion" {
		btn |= 32
	}
	end := 'M'
	if kind == "release" {
		end = 'm'
	}
	return fmt.Appendf(nil, "\x1b[<%d;%d;%d%c", btn, m.X+1, m.Y+1, end)
}

func lastSnapLine(snap string) string {
	snap = strings.TrimSpace(snap)
	if snap == "" {
		return ""
	}
	lines := strings.Split(snap, "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		s := strings.TrimSpace(lines[i])
		if s != "" {
			if len(s) > 80 {
				return s[:80]
			}
			return s
		}
	}
	return ""
}

func formatPaneErr(err error) string {
	if err == nil {
		return ""
	}
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		return fmt.Sprintf("exit status %d", ee.ExitCode())
	}
	return err.Error()
}
