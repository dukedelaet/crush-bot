package protocol

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dukedelaet/crush-bot/internal/roster"
)

func TestWriteCrushrcAndHooks(t *testing.T) {
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, "bots"), 0o700)
	bot, _, err := roster.Spawn(root, roster.SpawnOpts{Slug: "coder", Title: "Coder"})
	if err != nil {
		t.Fatal(err)
	}
	if err := Write(Options{Root: root, Bot: bot, SoulMax: 32768}); err != nil {
		t.Fatal(err)
	}
	home := roster.Home(root, "coder")
	wrap, _ := os.ReadFile(filepath.Join(home, "crushrc"))
	if !strings.Contains(string(wrap), "90-user.crushrc") || !strings.Contains(string(wrap), "[[ -f") {
		t.Fatalf("wrapper: %s", wrap)
	}
	host, _ := os.ReadFile(filepath.Join(home, "crushrc.d", "10-host.crushrc"))
	s := string(host)
	if !strings.Contains(s, "permissions deny bash") || !strings.Contains(s, "permissions deny edit") {
		t.Fatalf("deny missing: %s", s)
	}
	if strings.Contains(s, "mcp add") {
		t.Fatal("mcp should not be in PR 3 generator output by default")
	}
	if _, err := os.Stat(filepath.Join(home, "hooks", "identity.sh")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(home, "hooks", "identity.context.json")); err != nil {
		t.Fatal(err)
	}
	md, _ := os.ReadFile(filepath.Join(home, "CRUSH.md"))
	if !strings.Contains(string(md), "crushbot named coder") {
		t.Fatalf("CRUSH.md: %s", md)
	}
}

func TestCoderAllowsBash(t *testing.T) {
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, "bots"), 0o700)
	bot, _, err := roster.Spawn(root, roster.SpawnOpts{Slug: "dev", Coder: true})
	if err != nil {
		t.Fatal(err)
	}
	if err := Write(Options{Root: root, Bot: bot}); err != nil {
		t.Fatal(err)
	}
	host, _ := os.ReadFile(filepath.Join(roster.Home(root, "dev"), "crushrc.d", "10-host.crushrc"))
	s := string(host)
	if !strings.Contains(s, "permissions allow bash") || strings.Contains(s, "permissions deny bash") {
		t.Fatalf("%s", s)
	}
}
