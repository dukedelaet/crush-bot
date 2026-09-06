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
	}
	for _, p := range crushRuntimePaths(absCrush) {
		out = append(out, "--ro-bind-try", p, p)
	}
	if self != "" {
		out = append(out, "--ro-bind-try", self, self)
	}
	for _, p := range crushHostPaths() {
		out = append(out, "--ro-bind-try", p, p)
	}
	out = append(out,
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

func xdgDataHome() string {
	if v := os.Getenv("XDG_DATA_HOME"); v != "" {
		return v
	}
	h, _ := os.UserHomeDir()
	return filepath.Join(h, ".local", "share")
}

func xdgCacheHome() string {
	if v := os.Getenv("XDG_CACHE_HOME"); v != "" {
		return v
	}
	h, _ := os.UserHomeDir()
	return filepath.Join(h, ".cache")
}

// crushHostPaths are Crush's global config/data/cache dirs (not $HOME).
func crushHostPaths() []string {
	home, _ := os.UserHomeDir()
	seen := map[string]struct{}{}
	var out []string
	add := func(p string) {
		if p == "" {
			return
		}
		if _, ok := seen[p]; ok {
			return
		}
		seen[p] = struct{}{}
		out = append(out, p)
	}
	add("/etc/crush")
	add(filepath.Join(xdgConfigHome(), "crush"))
	add(filepath.Join(xdgDataHome(), "crush"))
	add(filepath.Join(xdgCacheHome(), "crush"))
	if home != "" {
		add(filepath.Join(home, ".config", "crush"))
		add(filepath.Join(home, ".local", "share", "crush"))
		add(filepath.Join(home, ".cache", "crush"))
	}
	return out
}

// crushRuntimePaths is the Crush install the sandbox must be able to read.
// The npm package is a symlink (~/.local/bin/crush → …/@charmland/crush/run-crush.js)
// plus sibling lib.js, node_modules, and bin/crush. Binding only the symlink
// yields EACCES on lib.js.
func crushRuntimePaths(crushBin string) []string {
	abs, err := exec.LookPath(crushBin)
	if err != nil || abs == "" {
		abs = crushBin
	}
	if abs == "" {
		return nil
	}
	if !filepath.IsAbs(abs) {
		if a, err := filepath.Abs(abs); err == nil {
			abs = a
		}
	}
	seen := map[string]struct{}{}
	var out []string
	add := func(p string) {
		if p == "" {
			return
		}
		if _, ok := seen[p]; ok {
			return
		}
		seen[p] = struct{}{}
		out = append(out, p)
	}
	add(abs)
	real, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return out
	}
	add(real)
	dir := filepath.Dir(real)
	if fileExists(filepath.Join(dir, "lib.js")) || fileExists(filepath.Join(dir, "package.json")) {
		add(dir)
	}
	nested := filepath.Join(dir, "bin", "crush")
	if fileExists(nested) {
		add(nested)
		if r2, err := filepath.EvalSymlinks(nested); err == nil {
			add(r2)
		}
	}
	return out
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
