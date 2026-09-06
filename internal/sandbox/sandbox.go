package sandbox

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"syscall"

	"github.com/hocoder-agents/crush-bot/internal/roster"
)

func Required(bot roster.Bot) bool {
	if bot.Sandbox == "off" {
		return false
	}
	return bot.Tools.Bash || bot.Tools.Edit
}

func Available() error {
	if runtime.GOOS != "linux" {
		return fmt.Errorf("sandbox is Linux-only in v1 (got %s); set sandbox: off to override", runtime.GOOS)
	}
	if _, err := exec.LookPath("bwrap"); err == nil {
		return nil
	}
	if err := landlockAvailable(); err == nil {
		return nil
	}
	return fmt.Errorf("no sandbox backend (install bubblewrap, or a Landlock kernel); or set sandbox: off")
}

func Backend() string {
	if runtime.GOOS != "linux" {
		return "none"
	}
	if _, err := exec.LookPath("bwrap"); err == nil {
		return "bwrap"
	}
	if landlockAvailable() == nil {
		return "landlock"
	}
	return "none"
}

// Wrap returns the binary and args to exec. If sandbox is not required, returns bin/args unchanged.
func Wrap(bin string, args []string, bot roster.Bot, root string) (string, []string, error) {
	if !Required(bot) {
		return bin, args, nil
	}
	if err := Available(); err != nil {
		return "", nil, err
	}
	if _, err := exec.LookPath("bwrap"); err == nil {
		bwrap, err := exec.LookPath("bwrap")
		if err != nil {
			return "", nil, err
		}
		return bwrap, BwrapArgs(bin, args, bot, root), nil
	}
	self, err := os.Executable()
	if err != nil {
		return "", nil, fmt.Errorf("landlock wrapper needs crushbot executable: %w", err)
	}
	wbin, wargs := landlockExecCmd(self, bin, args, bot, root)
	return wbin, wargs, nil
}

// ExecLandlocked is crushbot sandbox-exec: apply Landlock then syscall.Exec Crush.
func ExecLandlocked(crushBin, root, slug string, crushArgs []string) error {
	bot, err := roster.Load(root, slug)
	if err != nil {
		return err
	}
	abs, err := exec.LookPath(crushBin)
	if err != nil {
		abs = crushBin
	}
	self, _ := os.Executable()
	if err := applyLandlock(bot, root, abs, self); err != nil {
		return err
	}
	argv := append([]string{abs}, crushArgs...)
	return syscall.Exec(abs, argv, os.Environ())
}

func BwrapArgs(crushBin string, crushArgs []string, bot roster.Bot, root string) []string {
	home := roster.Home(root, bot.Slug)
	absCrush, _ := exec.LookPath(crushBin)
	if absCrush == "" {
		absCrush = crushBin
	}
	self, _ := os.Executable()
	xdgCrush := filepath.Join(xdgConfigHome(), "crush")
	sandboxHome := filepath.Join(home, "sandbox-home")
	_ = os.MkdirAll(sandboxHome, 0o700)
	_ = os.MkdirAll(filepath.Join(home, "inbox", "pending"), 0o700)
	needs := filepath.Join(root, "needs_you.jsonl")
	if _, err := os.Stat(needs); os.IsNotExist(err) {
		_ = os.WriteFile(needs, nil, 0o600)
	}

	out := []string{
		"--die-with-parent", "--unshare-pid", "--unshare-uts",
		"--proc", "/proc", "--dev", "/dev", "--tmpfs", "/tmp",
		"--ro-bind", "/usr", "/usr",
		"--ro-bind", "/bin", "/bin",
		"--ro-bind", "/lib", "/lib",
		"--ro-bind-try", "/lib64", "/lib64",
		"--ro-bind-try", "/etc/ssl", "/etc/ssl",
		"--ro-bind-try", "/etc/resolv.conf", "/etc/resolv.conf",
		"--ro-bind-try", absCrush, absCrush,
	}
	if self != "" {
		out = append(out, "--ro-bind-try", self, self)
	}
	out = append(out,
		"--ro-bind-try", xdgCrush, xdgCrush,
		"--ro-bind", root, root,
		"--bind", home, home,
		"--bind", needs, needs,
		"--tmpfs", sandboxHome,
		"--setenv", "HOME", sandboxHome,
		"--chdir", home,
	)
	bots, _ := roster.List(root, true)
	for _, b := range bots {
		if b.Slug == bot.Slug {
			continue
		}
		pend := filepath.Join(roster.Home(root, b.Slug), "inbox", "pending")
		_ = os.MkdirAll(pend, 0o700)
		out = append(out, "--bind", pend, pend)
	}
	if bot.Project != "" {
		out = append(out, "--bind", bot.Project, bot.Project)
	}
	out = append(out, "--", absCrush)
	out = append(out, crushArgs...)
	return out
}

func xdgConfigHome() string {
	if v := os.Getenv("XDG_CONFIG_HOME"); v != "" {
		return v
	}
	h, _ := os.UserHomeDir()
	return filepath.Join(h, ".config")
}
