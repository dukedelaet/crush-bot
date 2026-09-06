package daemon

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/hocoder-agents/crush-bot/internal/lock"
)

func pidPath(root string) string  { return filepath.Join(root, "daemon.pid") }
func lockPath(root string) string { return filepath.Join(root, "daemon.lock") }
func logPath(root string) string  { return filepath.Join(root, "logs", "daemon.log") }

func Live(root string) bool {
	b, err := os.ReadFile(pidPath(root))
	if err != nil {
		return false
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(b)))
	if err != nil || pid <= 0 {
		return false
	}
	return syscall.Kill(pid, 0) == nil
}

func PID(root string) (int, error) {
	b, err := os.ReadFile(pidPath(root))
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(strings.TrimSpace(string(b)))
}

func WritePID(root string, pid int) error {
	return os.WriteFile(pidPath(root), []byte(strconv.Itoa(pid)+"\n"), 0o600)
}

func AcquireSingleton(root string) (*lock.Lock, error) {
	if Live(root) {
		pid, _ := PID(root)
		return nil, fmt.Errorf("daemon already running (pid %d)", pid)
	}
	return lock.Acquire(lockPath(root), time.Second, true)
}

func Stop(root string) error {
	pid, err := PID(root)
	if err != nil {
		return fmt.Errorf("daemon not running")
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	_ = proc.Signal(syscall.SIGTERM)
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if syscall.Kill(pid, 0) != nil {
			_ = os.Remove(pidPath(root))
			return nil
		}
		time.Sleep(50 * time.Millisecond)
	}
	_ = proc.Signal(syscall.SIGKILL)
	_ = os.Remove(pidPath(root))
	return nil
}
