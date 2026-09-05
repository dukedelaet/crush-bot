package crush

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/dukedelaet/crush-bot/internal/roster"
	"github.com/dukedelaet/crush-bot/internal/sandbox"
)

func SocketPath(botHome string) string { return filepath.Join(botHome, "crush.sock") }
func ServerPIDPath(botHome string) string {
	return filepath.Join(botHome, "server.pid")
}
func ServerLogPath(botHome string) string {
	return filepath.Join(botHome, "logs", "server.log")
}

func HostURL(botHome string) string {
	return "unix://" + SocketPath(botHome)
}

func ServerLive(botHome string) bool {
	b, err := os.ReadFile(ServerPIDPath(botHome))
	if err != nil {
		return false
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(b)))
	if err != nil || pid <= 0 {
		return false
	}
	if syscall.Kill(pid, 0) != nil {
		return false
	}
	if _, err := os.Stat(SocketPath(botHome)); err != nil {
		return false
	}
	return true
}

func StartServer(bin string, bot roster.Bot, root string) error {
	home := roster.Home(root, bot.Slug)
	if ServerLive(home) {
		return nil
	}
	_ = os.MkdirAll(filepath.Join(home, "logs"), 0o700)
	_ = os.Remove(SocketPath(home))
	args := []string{
		"server",
		"--cwd", home,
		"--data-dir", filepath.Join(home, ".crush"),
		"--host", HostURL(home),
	}
	wbin, wargs, err := sandbox.Wrap(bin, args, bot, root)
	if err != nil {
		return err
	}
	cmd := exec.Command(wbin, wargs...)
	cmd.Dir = home
	logf, err := os.OpenFile(ServerLogPath(home), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	cmd.Stdout = logf
	cmd.Stderr = logf
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		logf.Close()
		return fmt.Errorf("start crush server: %w", err)
	}
	pid := cmd.Process.Pid
	if err := os.WriteFile(ServerPIDPath(home), []byte(strconv.Itoa(pid)+"\n"), 0o600); err != nil {
		_ = cmd.Process.Kill()
		logf.Close()
		return err
	}
	_ = cmd.Process.Release()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if ServerLive(home) || PIDAlive(pid) {
			return nil
		}
		time.Sleep(50 * time.Millisecond)
	}
	return fmt.Errorf("crush server for @%s did not come up", bot.Slug)
}

func StopServer(botHome string) error {
	b, err := os.ReadFile(ServerPIDPath(botHome))
	if err != nil {
		return fmt.Errorf("no crush server")
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(b)))
	if err != nil {
		return err
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	_ = syscall.Kill(-pid, syscall.SIGTERM)
	_ = proc.Signal(syscall.SIGTERM)
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if syscall.Kill(pid, 0) != nil {
			_ = os.Remove(ServerPIDPath(botHome))
			_ = os.Remove(SocketPath(botHome))
			return nil
		}
		time.Sleep(40 * time.Millisecond)
	}
	_ = syscall.Kill(-pid, syscall.SIGKILL)
	_ = proc.Signal(syscall.SIGKILL)
	_ = os.Remove(ServerPIDPath(botHome))
	_ = os.Remove(SocketPath(botHome))
	return nil
}
