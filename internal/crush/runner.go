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

	"github.com/dukedelaet/crush-bot/internal/lock"
	"github.com/dukedelaet/crush-bot/internal/roster"
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
	if opts.Bot.CanonicalSessionID != "" {
		args = append(args, "--session", opts.Bot.CanonicalSessionID)
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
	if opts.Bot.CanonicalSessionID != "" {
		args = append(args, "--session", opts.Bot.CanonicalSessionID)
	}
	return args
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
		SessionID:  opts.Bot.CanonicalSessionID,
		Kind:       kind,
		Trace:      trace,
		Inbound:    opts.Inbound,
		InboundHop: opts.InboundHop,
		ParentID:   opts.ParentID,
		MaxSends:   MaxSendsFor(kind),
		MaxHops:    opts.MaxHops,
	}
	if err := WriteTurn(home, turn); err != nil {
		return Result{}, err
	}
	defer RemoveTurn(home)

	cmd := exec.CommandContext(ctx, opts.Bin, runArgs(opts)...)
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

func Chat(ctx context.Context, opts RunOpts) error {
	home := BotHome(opts)
	l, err := acquireTurn(opts)
	if err != nil {
		return err
	}
	defer l.Unlock()

	turn := Turn{
		Bot:       opts.Bot.Slug,
		SessionID: opts.Bot.CanonicalSessionID,
		Kind:      "human_chat",
		Trace:     []string{"user"},
		MaxSends:  32,
		MaxHops:   8,
	}
	if err := WriteTurn(home, turn); err != nil {
		return err
	}
	defer RemoveTurn(home)

	cmd := exec.CommandContext(ctx, opts.Bin, chatArgs(opts)...)
	cmd.Dir = home
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
	if err := UpdatePID(home, cmd.Process.Pid); err != nil {
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
		return err
	}
	return cmd.Wait()
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
		return configured, nil
	}
	p, err := exec.LookPath(configured)
	if err != nil {
		return "", fmt.Errorf("crush not found on PATH (%s)", configured)
	}
	return p, nil
}

func IsBusy(err error) bool {
	return errors.Is(err, lock.ErrBusy) || (err != nil && strings.Contains(err.Error(), "bot busy"))
}
