package crush

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/hocoder-agents/crush-bot/internal/roster"
)

func TestStartStopServer(t *testing.T) {
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash")
	}
	src, err := os.ReadFile(filepath.Join("testdata", "fake-crush.sh"))
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	bin := filepath.Join(dir, "crush")
	if err := os.WriteFile(bin, src, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("FAKE_CRUSH_STATE", filepath.Join(dir, "st"))
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, "bots"), 0o700)
	bot, _, err := roster.Spawn(root, roster.SpawnOpts{Slug: "coder", KeepAlive: true})
	if err != nil {
		t.Fatal(err)
	}
	if err := StartServer(bin, bot, root); err != nil {
		t.Fatal(err)
	}
	home := roster.Home(root, "coder")
	b, err := os.ReadFile(ServerPIDPath(home))
	if err != nil {
		t.Fatal(err)
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(b)))
	if err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) && !PIDAlive(pid) {
		time.Sleep(20 * time.Millisecond)
	}
	if !PIDAlive(pid) {
		t.Fatal("server pid dead")
	}
	if err := StopServer(home); err != nil {
		t.Fatal(err)
	}
	deadline = time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) && PIDAlive(pid) {
		time.Sleep(20 * time.Millisecond)
	}
	if PIDAlive(pid) {
		t.Fatal("server still alive")
	}
}
