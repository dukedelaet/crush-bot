package sandbox

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/dukedelaet/crush-bot/internal/roster"
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
	return fmt.Errorf("bwrap not found; install bubblewrap or set sandbox: off")
}

// Wrap returns the binary and args to exec. If sandbox is not required, returns bin/args unchanged.
func Wrap(bin string, args []string, bot roster.Bot, root string) (string, []string, error) {
	if !Required(bot) {
		return bin, args, nil
	}
	if err := Available(); err != nil {
		return "", nil, err
	}
	bwrap, err := exec.LookPath("bwrap")
	if err != nil {
		return "", nil, err
	}
	wrapped := BwrapArgs(bin, args, bot, root)
	return bwrap, wrapped, nil
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
