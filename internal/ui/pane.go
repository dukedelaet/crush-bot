package ui

import (
	"context"
	"io"
	"syscall"
	"time"

	tea "charm.land/bubbletea/v2"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/vt"
	"github.com/charmbracelet/x/xpty"

	"github.com/hocoder-agents/crush-bot/internal/config"
	"github.com/hocoder-agents/crush-bot/internal/crush"
	"github.com/hocoder-agents/crush-bot/internal/roster"
)

type paneDrawMsg struct{}

type paneExitMsg struct {
	slug string
	err  error
}

type crushPane struct {
	slug   string
	pty    xpty.Pty
	em     *vt.SafeEmulator
	chat   *crush.ChatSession
	msgs   chan tea.Msg
	done   chan struct{}
	w, h   int
	closed bool
}

func openCrushPane(home string, bot roster.Bot, w, h int) (*crushPane, tea.Cmd, error) {
	if w < 8 {
		w = 8
	}
	if h < 8 {
		h = 8
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
	pty, err := xpty.NewPty(w, h)
	if err != nil {
		chat.Close()
		return nil, nil, err
	}
	cmd := chat.Cmd
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true, Setctty: true}
	if err := pty.Start(cmd); err != nil {
		_ = pty.Close()
		chat.Close()
		return nil, nil, err
	}
	if cmd.Process != nil {
		_ = crush.UpdatePID(chat.Home, cmd.Process.Pid)
	}
	pane := &crushPane{
		slug: bot.Slug,
		pty:  pty,
		em:   vt.NewSafeEmulator(w, h),
		chat: chat,
		msgs: make(chan tea.Msg, 8),
		done: make(chan struct{}),
		w:    w,
		h:    h,
	}
	go pane.copyIn()
	go pane.copyOut()
	go pane.reap()
	return pane, pane.listen(), nil
}

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
	_, _ = io.Copy(p.pty, p.em)
}

func (p *crushPane) copyOut() {
	buf := make([]byte, 8192)
	for {
		n, err := p.pty.Read(buf)
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
	err := xpty.WaitProcess(context.Background(), p.chat.Cmd)
	if p.chat != nil {
		p.chat.Close()
	}
	select {
	case p.msgs <- paneExitMsg{slug: p.slug, err: err}:
	default:
	}
}

func (p *crushPane) sendKey(msg tea.KeyPressMsg) {
	if p == nil || p.em == nil {
		return
	}
	k := msg.Key()
	p.em.SendKey(uv.KeyPressEvent{
		Text:        k.Text,
		Mod:         uv.KeyMod(k.Mod),
		Code:        k.Code,
		ShiftedCode: k.ShiftedCode,
		BaseCode:    k.BaseCode,
		IsRepeat:    k.IsRepeat,
	})
}

func (p *crushPane) sendMouse(kind string, m tea.Mouse) {
	if p == nil || p.em == nil {
		return
	}
	ev := uv.Mouse{X: m.X, Y: m.Y, Button: uv.MouseButton(m.Button), Mod: uv.KeyMod(m.Mod)}
	switch kind {
	case "click":
		p.em.SendMouse(uv.MouseClickEvent(ev))
	case "release":
		p.em.SendMouse(uv.MouseReleaseEvent(ev))
	case "wheel":
		p.em.SendMouse(uv.MouseWheelEvent(ev))
	default:
		p.em.SendMouse(uv.MouseMotionEvent(ev))
	}
}

func (p *crushPane) resize(w, h int) {
	if p == nil || p.pty == nil || w < 8 || h < 8 {
		return
	}
	if w == p.w && h == p.h {
		return
	}
	p.w, p.h = w, h
	p.em.Resize(w, h)
	_ = p.pty.Resize(w, h)
}

func (p *crushPane) render() string {
	if p == nil || p.em == nil {
		return ""
	}
	return p.em.Render()
}

func (p *crushPane) close() {
	if p == nil || p.closed {
		return
	}
	p.closed = true
	if p.chat != nil && p.chat.Cmd != nil && p.chat.Cmd.Process != nil {
		_ = p.chat.Cmd.Process.Signal(syscall.SIGINT)
	}
	if p.pty != nil {
		_ = p.pty.Close()
		p.pty = nil
	}
	if p.em != nil {
		_ = p.em.Close()
		p.em = nil
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
}
