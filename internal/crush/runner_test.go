package crush

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/dukedelaet/crush-bot/internal/roster"
)

func fakeBin(t *testing.T) string {
	t.Helper()
	src := filepath.Join("testdata", "fake-crush.sh")
	dst := filepath.Join(t.TempDir(), "crush")
	b, err := os.ReadFile(src)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dst, b, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("FAKE_CRUSH_STATE", filepath.Join(t.TempDir(), "st"))
	return dst
}

func TestFakeVersionAndProviders(t *testing.T) {
	bin := fakeBin(t)
	if err := RequireMin(bin, MinVersion); err != nil {
		t.Fatal(err)
	}
	if err := HasProviders(bin); err != nil {
		t.Fatal(err)
	}
	if !HasRootYolo(bin) {
		t.Fatal("fake help should mention --yolo")
	}
}

func TestRunAndBootstrap(t *testing.T) {
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash required for fake crush")
	}
	bin := fakeBin(t)
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, "bots"), 0o700)
	bot, _, err := roster.Spawn(root, roster.SpawnOpts{Slug: "coder", Title: "Coder"})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	bot, err = Bootstrap(ctx, RunOpts{Bot: bot, Root: root, Bin: bin, Timeout: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	if bot.CanonicalSessionID != "11111111-1111-4111-8111-111111111111" {
		t.Fatalf("uuid %s", bot.CanonicalSessionID)
	}
	if bot.CanonicalSessionTitle != "bot:coder" {
		t.Fatalf("title %s", bot.CanonicalSessionTitle)
	}
}

func TestReclaimProcessing(t *testing.T) {
	home := t.TempDir()
	proc := filepath.Join(home, "inbox", "processing")
	pend := filepath.Join(home, "inbox", "pending")
	os.MkdirAll(proc, 0o700)
	os.MkdirAll(pend, 0o700)
	os.WriteFile(filepath.Join(proc, "01.json"), []byte(`{"attempt":0}`), 0o600)
	os.WriteFile(TurnPath(home), []byte(`{"crush_pid":99999999,"bot":"x"}`), 0o600)
	ok, err := ReclaimStale(home)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected reclaim")
	}
	if _, err := os.Stat(filepath.Join(pend, "01.json")); err != nil {
		t.Fatal("not moved")
	}
	body, _ := os.ReadFile(filepath.Join(pend, "01.json"))
	if string(body) != `{"attempt":0}` {
		t.Fatalf("attempt changed: %s", body)
	}
}
