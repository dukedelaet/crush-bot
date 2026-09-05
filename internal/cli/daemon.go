package cli

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"path/filepath"

	"github.com/dukedelaet/crush-bot/internal/config"
	"github.com/dukedelaet/crush-bot/internal/daemon"
	"github.com/dukedelaet/crush-bot/internal/envelope"
	"github.com/dukedelaet/crush-bot/internal/roster"
)

func cmdDaemon(io IO, args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(io.Err, errStyle.Render("usage: crushbot daemon start|stop|status|logs|run"))
		return 2
	}
	switch args[0] {
	case "start":
		return daemonStart(io)
	case "stop":
		return daemonStop(io)
	case "status":
		return daemonStatus(io)
	case "logs":
		return daemonLogs(io)
	case "run":
		return daemonRun(io)
	default:
		fmt.Fprintln(io.Err, errStyle.Render("usage: crushbot daemon start|stop|status|logs|run"))
		return 2
	}
}

func daemonStart(io IO) int {
	p := config.ResolvePaths()
	if err := config.EnsureHome(p); err != nil {
		return fail(io, err)
	}
	if daemon.Live(p.Home) {
		pid, _ := daemon.PID(p.Home)
		fmt.Fprintln(io.Err, errStyle.Render(fmt.Sprintf("daemon already running (pid %d)", pid)))
		return 1
	}
	exe, err := os.Executable()
	if err != nil {
		return fail(io, err)
	}
	cmd := exec.Command(exe, "daemon", "run")
	cmd.Env = os.Environ()
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		return fail(io, err)
	}
	fmt.Fprintln(io.Out, okStyle.Render(fmt.Sprintf("daemon started pid %d", cmd.Process.Pid)))
	_ = cmd.Process.Release()
	return 0
}

func daemonStop(io IO) int {
	p := config.ResolvePaths()
	if err := daemon.Stop(p.Home); err != nil {
		return fail(io, err)
	}
	fmt.Fprintln(io.Out, okStyle.Render("daemon stopped"))
	return 0
}

func daemonStatus(io IO) int {
	p := config.ResolvePaths()
	if daemon.Live(p.Home) {
		pid, _ := daemon.PID(p.Home)
		fmt.Fprintln(io.Out, okStyle.Render(fmt.Sprintf("running pid %d", pid)))
		return 0
	}
	fmt.Fprintln(io.Out, mutedStyle.Render("not running"))
	return 1
}

func daemonLogs(io IO) int {
	p := config.ResolvePaths()
	b, err := os.ReadFile(filepath.Join(p.Home, "logs", "daemon.log"))
	if err != nil {
		return fail(io, err)
	}
	fmt.Fprint(io.Out, string(b))
	return 0
}

func daemonRun(io IO) int {
	p := config.ResolvePaths()
	cfg, err := config.Load(p)
	if err != nil {
		return fail(io, err)
	}
	if err := config.EnsureHome(p); err != nil {
		return fail(io, err)
	}
	lck, err := daemon.AcquireSingleton(p.Home)
	if err != nil {
		return fail(io, err)
	}
	defer lck.Unlock()
	bin, err := crushBin(cfg)
	if err != nil {
		return fail(io, err)
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	fmt.Fprintln(io.Err, mutedStyle.Render("daemon running"))
	if err := daemon.Run(ctx, daemon.Options{
		Root: p.Home,
		Bin:  bin,
		Cfg:  cfg,
		Poll: 200 * time.Millisecond,
	}); err != nil {
		return fail(io, err)
	}
	return 0
}

func cmdInbox(io IO, args []string) int {
	p := config.ResolvePaths()
	if len(args) >= 2 && args[0] == "retry" {
		return inboxRetry(io, p, args[1])
	}
	slug := ""
	if len(args) == 1 && args[0] != "retry" {
		slug = args[0]
	}
	var slugs []string
	if slug != "" {
		slugs = []string{slug}
	} else {
		bots, err := roster.List(p.Home, true)
		if err != nil {
			return fail(io, err)
		}
		for _, b := range bots {
			slugs = append(slugs, b.Slug)
		}
	}
	for _, s := range slugs {
		home := roster.Home(p.Home, s)
		fmt.Fprintln(io.Out, headStyle.Render(s))
		for _, st := range []string{"pending", "processing", "archive", "failed"} {
			envs, _, _ := envelope.List(envelope.Dir(home, st))
			fmt.Fprintf(io.Out, "  %s %d\n", st, len(envs))
			for _, e := range envs {
				fmt.Fprintf(io.Out, "    %s %s from=@%s hop=%d %s\n", e.ID, e.Kind, e.From, e.Hop, e.Body)
			}
		}
	}
	return 0
}

func inboxRetry(io IO, p config.Paths, id string) int {
	bots, err := roster.List(p.Home, true)
	if err != nil {
		return fail(io, err)
	}
	for _, b := range bots {
		home := roster.Home(p.Home, b.Slug)
		src := filepath.Join(home, "inbox", "failed", id+".json")
		env, err := envelope.ReadFile(src)
		if err != nil {
			continue
		}
		env.Attempt = 0
		if _, err := envelope.WritePending(home, env); err != nil {
			return fail(io, err)
		}
		_ = os.Remove(src)
		fmt.Fprintln(io.Out, okStyle.Render("retry "+id+" → @"+b.Slug+" pending"))
		return 0
	}
	return fail(io, fmt.Errorf("envelope %s not in failed/", id))
}
