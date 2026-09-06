package crush

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/hocoder-agents/crush-bot/internal/lock"
	"github.com/hocoder-agents/crush-bot/internal/roster"
	"github.com/hocoder-agents/crush-bot/internal/sandbox"
)

type Result struct {
	ExitCode int
	Stdout   string
	Stderr   string
}

type RunOpts struct {
	Bot        roster.Bot
	Root       string // CRUSHBOT_HOME
	Bin        string
	Kind       string
	Prompt     string
	Nowait     bool
	Timeout    time.Duration
	Debug      bool
	Yolo       bool
	SessionID  string // if set, overrides bot.CanonicalSessionID
	NoSession  bool   // omit --session (mint a new Crush session)
	GroupID    string
	Stdin      io.Reader
	Stdout     io.Writer
	Stderr     io.Writer
	Inbound    []InboundRef
	InboundHop int
	Trace      []string
	ParentID   string
	MaxHops    int
	HeldLock   *lock.Lock // if set, caller owns flock
}

func BotHome(opts RunOpts) string {
	return roster.Home(opts.Root, opts.Bot.Slug)
}

func dataDir(opts RunOpts) string {
	return filepath.Join(BotHome(opts), ".crush")
}

func runArgs(opts RunOpts) []string {
	args := []string{}
	if opts.Yolo {
		args = append(args, "--yolo")
	}
	if opts.Debug {
		args = append(args, "--debug")
	}
	args = append(args, "run",
		"--cwd", BotHome(opts),
		"--data-dir", dataDir(opts),
		"--quiet",
	)
	if host := clientHost(opts); host != "" {
		args = append(args, "--host", host)
	}
	if sid := sessionOf(opts); sid != "" {
		args = append(args, "--session", sid)
	}
	if opts.Bot.Model != "" {
		args = append(args, "--model", opts.Bot.Model)
	}
	if opts.Prompt != "" {
		args = append(args, opts.Prompt)
	}
	return args
}

func chatArgs(opts RunOpts) []string {
	args := []string{}
	if opts.Debug {
		args = append(args, "--debug")
	}
	args = append(args,
		"--cwd", BotHome(opts),
		"--data-dir", dataDir(opts),
	)
	if host := clientHost(opts); host != "" {
		args = append(args, "--host", host)
	}
	if sid := sessionOf(opts); sid != "" {
		args = append(args, "--session", sid)
	}
	return args
}

func clientHost(opts RunOpts) string {
	home := BotHome(opts)
	if ServerLive(home) {
		return HostURL(home)
	}
	return ""
}

func sessionOf(opts RunOpts) string {
	if opts.NoSession {
		return ""
	}
	if opts.SessionID != "" {
		return opts.SessionID
	}
	return opts.Bot.CanonicalSessionID
}

func acquireTurn(opts RunOpts) (*lock.Lock, error) {
	home := BotHome(opts)
	_, _ = ReclaimStale(home)
	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = 120 * time.Second
	}
	l, err := lock.Acquire(LockPath(home), timeout, opts.Nowait)
	if err != nil {
		return nil, err
	}
	if t, err := ReadTurn(home); err == nil && PIDAlive(t.CrushPID) {
		l.Unlock()
		if opts.Nowait {
			return nil, fmt.Errorf("bot busy (pid %d from turn.json); crushbot stop %s", t.CrushPID, opts.Bot.Slug)
		}
		deadline := time.Now().Add(timeout)
		for PIDAlive(t.CrushPID) && time.Now().Before(deadline) {
			time.Sleep(100 * time.Millisecond)
			t, err = ReadTurn(home)
			if err != nil {
				break
			}
		}
		if t, err := ReadTurn(home); err == nil && PIDAlive(t.CrushPID) {
			return nil, fmt.Errorf("bot busy (pid %d from turn.json); crushbot stop %s", t.CrushPID, opts.Bot.Slug)
		}
		return acquireTurn(opts)
	}
	return l, nil
}

func Run(ctx context.Context, opts RunOpts) (Result, error) {
	home := BotHome(opts)
	if opts.HeldLock == nil {
		l, err := acquireTurn(opts)
		if err != nil {
			return Result{}, err
		}
		defer l.Unlock()
	}

	trace := opts.Trace
	if len(trace) == 0 {
		trace = []string{"user"}
	}
	kind := opts.Kind
	if kind == "" {
		kind = "human_say"
	}
	turn := Turn{
		Bot:        opts.Bot.Slug,
		SessionID:  sessionOf(opts),
		Kind:       kind,
		Trace:      trace,
		Inbound:    opts.Inbound,
		InboundHop: opts.InboundHop,
		ParentID:   opts.ParentID,
		MaxSends:   MaxSendsFor(kind),
		MaxHops:    opts.MaxHops,
	}
	if opts.GroupID != "" {
		gid := opts.GroupID
		turn.GroupID = &gid
	}
	if err := WriteTurn(home, turn); err != nil {
		return Result{}, err
	}
	defer RemoveTurn(home)

	bin, args, err := sandbox.Wrap(opts.Bin, runArgs(opts), opts.Bot, opts.Root)
	if err != nil {
		return Result{}, err
	}
	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Dir = home
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	if opts.Stdout != nil {
		cmd.Stdout = io.MultiWriter(&stdout, opts.Stdout)
	}
	cmd.Stderr = &stderr
	if opts.Stderr != nil {
		cmd.Stderr = io.MultiWriter(&stderr, opts.Stderr)
	}
	if opts.Stdin != nil {
		cmd.Stdin = opts.Stdin
	}
	if err := cmd.Start(); err != nil {
		_, _ = ReclaimStale(home)
		return Result{}, fmt.Errorf("start crush: %w", err)
	}
	if err := UpdatePID(home, cmd.Process.Pid); err != nil {
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
		return Result{}, err
	}
	werr := cmd.Wait()
	res := Result{Stdout: stdout.String(), Stderr: stderr.String()}
	if cmd.ProcessState != nil {
		res.ExitCode = cmd.ProcessState.ExitCode()
	}
	if werr != nil && res.ExitCode == 0 {
		return res, werr
	}
	if res.ExitCode != 0 {
		return res, fmt.Errorf("crush run exit %d: %s", res.ExitCode, strings.TrimSpace(res.Stderr+" "+res.Stdout))
	}
	return res, nil
}

// ChatSession is a Crush TUI process that holds turn.lock until Close.
type ChatSession struct {
	Cmd  *exec.Cmd
	Home string
	Slug string
	lock *lock.Lock
}

func BeginChat(opts RunOpts) (*ChatSession, error) {
	home := BotHome(opts)
	l, err := acquireTurn(opts)
	if err != nil {
		return nil, err
	}
	turn := Turn{
		Bot:       opts.Bot.Slug,
		SessionID: opts.Bot.CanonicalSessionID,
		Kind:      "human_chat",
		Trace:     []string{"user"},
		MaxSends:  32,
		MaxHops:   8,
	}
	if err := WriteTurn(home, turn); err != nil {
		l.Unlock()
		return nil, err
	}
	bin, args, err := sandbox.Wrap(opts.Bin, chatArgs(opts), opts.Bot, opts.Root)
	if err != nil {
		RemoveTurn(home)
		l.Unlock()
		return nil, err
	}
	cmd := exec.Command(bin, args...)
	cmd.Dir = home
	cmd.Env = overlayEnv(os.Environ(), "TERM=xterm-256color", "COLORTERM=truecolor")
	return &ChatSession{Cmd: cmd, Home: home, Slug: opts.Bot.Slug, lock: l}, nil
}

// overlayEnv copies base, dropping keys that kv will set, then appends kv.
// os.Getenv / C getenv use the first match, so appending TERM is not enough.
func overlayEnv(base []string, kv ...string) []string {
	drop := make(map[string]bool, len(kv))
	for _, x := range kv {
		k, _, _ := strings.Cut(x, "=")
		drop[k] = true
	}
	out := make([]string, 0, len(base)+len(kv))
	for _, e := range base {
		k, _, _ := strings.Cut(e, "=")
		if drop[k] {
			continue
		}
		out = append(out, e)
	}
	return append(out, kv...)
}

func (s *ChatSession) Close() {
	if s == nil {
		return
	}
	RemoveTurn(s.Home)
	if s.lock != nil {
		s.lock.Unlock()
		s.lock = nil
	}
}

func Chat(ctx context.Context, opts RunOpts) error {
	s, err := BeginChat(opts)
	if err != nil {
		return err
	}
	defer s.Close()
	cmd := s.Cmd
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if opts.Stdin != nil {
		cmd.Stdin = opts.Stdin
	}
	if opts.Stdout != nil {
		cmd.Stdout = opts.Stdout
	}
	if opts.Stderr != nil {
		cmd.Stderr = opts.Stderr
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start crush: %w", err)
	}
	if err := UpdatePID(s.Home, cmd.Process.Pid); err != nil {
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
		return err
	}
	wait := make(chan error, 1)
	go func() { wait <- cmd.Wait() }()
	select {
	case <-ctx.Done():
		_ = cmd.Process.Signal(syscall.SIGINT)
		<-wait
		return ctx.Err()
	case err := <-wait:
		return err
	}
}

func Stop(botHome string) error {
	t, err := ReadTurn(botHome)
	if os.IsNotExist(err) {
		return fmt.Errorf("no in-flight crush turn")
	}
	if err != nil {
		return err
	}
	pid := t.CrushPID
	if pid == 0 {
		time.Sleep(pidZeroGrace)
		t, err = ReadTurn(botHome)
		if err != nil {
			return err
		}
		pid = t.CrushPID
	}
	if pid == 0 {
		RemoveTurn(botHome)
		return nil
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	_ = proc.Signal(syscall.SIGINT)
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if !PIDAlive(pid) {
			RemoveTurn(botHome)
			return nil
		}
		time.Sleep(50 * time.Millisecond)
	}
	_ = proc.Signal(syscall.SIGKILL)
	RemoveTurn(botHome)
	return nil
}

func Bootstrap(ctx context.Context, opts RunOpts) (roster.Bot, error) {
	opts.Kind = "human_say"
	opts.Prompt = "You are coming online. Introduce yourself in one short paragraph."
	if _, err := Run(ctx, opts); err != nil {
		return opts.Bot, err
	}
	meta, err := SessionLast(opts.Bin, BotHome(opts), dataDir(opts))
	if err != nil {
		return opts.Bot, err
	}
	title := "bot:" + opts.Bot.Slug
	if err := SessionRename(opts.Bin, BotHome(opts), dataDir(opts), meta.UUID, title); err != nil {
		return opts.Bot, err
	}
	opts.Bot.CanonicalSessionID = meta.UUID
	opts.Bot.CanonicalSessionTitle = title
	if err := roster.Save(opts.Root, opts.Bot); err != nil {
		return opts.Bot, err
	}
	return opts.Bot, nil
}

func LookPath(configured string) (string, error) {
	if configured == "" {
		configured = "crush"
	}
	if filepath.IsAbs(configured) {
		if _, err := os.Stat(configured); err != nil {
			return "", fmt.Errorf("crush_path %s: %w", configured, err)
		}
		return preferNative(configured), nil
	}
	p, err := exec.LookPath(configured)
	if err != nil {
		return "", fmt.Errorf("crush not found on PATH (%s)", configured)
	}
	return preferNative(p), nil
}

// preferNative follows an npm wrapper (run-crush.js) to bin/crush so the
// Go TUI owns the PTY instead of Node spawnSync.
func preferNative(bin string) string {
	real, err := filepath.EvalSymlinks(bin)
	if err != nil {
		return bin
	}
	nested := filepath.Join(filepath.Dir(real), "bin", "crush")
	if st, err := os.Stat(nested); err == nil && !st.IsDir() {
		if r2, err := filepath.EvalSymlinks(nested); err == nil {
			return r2
		}
		return nested
	}
	return bin
}

func IsBusy(err error) bool {
	return errors.Is(err, lock.ErrBusy) || (err != nil && strings.Contains(err.Error(), "bot busy"))
}
