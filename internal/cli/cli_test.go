package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHelp(t *testing.T) {
	var out, errb bytes.Buffer
	code := run(IO{Out: &out, Err: &errb}, []string{"--help"})
	if code != 0 {
		t.Fatalf("exit %d err=%s", code, errb.String())
	}
	if !strings.Contains(out.String(), "crushbot") {
		t.Fatalf("help: %s", out.String())
	}
	if strings.Contains(strings.ToLower(out.String()), "cobra") {
		t.Fatal("help must not mention cobra")
	}
}

func TestUnknownCommand(t *testing.T) {
	var out, errb bytes.Buffer
	code := run(IO{Out: &out, Err: &errb}, []string{"nope"})
	if code != 2 {
		t.Fatalf("exit %d", code)
	}
	if !strings.Contains(errb.String(), "unknown command") {
		t.Fatalf("err: %s", errb.String())
	}
}

func TestVersion(t *testing.T) {
	var out bytes.Buffer
	code := run(IO{Out: &out, Err: &out}, []string{"version"})
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	if !strings.Contains(out.String(), Version) {
		t.Fatalf("version: %s", out.String())
	}
}

func TestInit(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CRUSHBOT_HOME", filepath.Join(dir, "home"))
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(dir, "cfg"))
	var out, errb bytes.Buffer
	code := run(IO{Out: &out, Err: &errb}, []string{"init"})
	if code != 0 {
		t.Fatalf("exit %d err=%s", code, errb.String())
	}
	home := filepath.Join(dir, "home")
	if _, err := os.Stat(filepath.Join(home, "bots")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "cfg", "crushbot", "config.yaml")); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "initialized") {
		t.Fatalf("out: %s", out.String())
	}
}

func TestMeshPlain(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CRUSHBOT_HOME", filepath.Join(dir, "home"))
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(dir, "cfg"))
	var out, errb bytes.Buffer
	code := run(IO{Out: &out, Err: &errb}, []string{"mesh", "--plain"})
	if code != 0 {
		t.Fatalf("exit %d err=%s", code, errb.String())
	}
	if !strings.Contains(out.String(), "no bots") {
		t.Fatalf("out: %s", out.String())
	}
}

func TestSpawnListShowDelete(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CRUSHBOT_HOME", filepath.Join(dir, "home"))
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(dir, "cfg"))
	var out, errb bytes.Buffer
	io := IO{Out: &out, Err: &errb, In: strings.NewReader("")}
	if code := run(io, []string{"init"}); code != 0 {
		t.Fatalf("init %d %s", code, errb.String())
	}
	out.Reset()
	errb.Reset()
	if code := run(io, []string{"spawn", "researcher", "--title", "Researcher"}); code != 0 {
		t.Fatalf("spawn %d %s", code, errb.String())
	}
	soul := filepath.Join(dir, "home", "bots", "researcher", "soul.md")
	if _, err := os.Stat(soul); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	if code := run(io, []string{"list", "--json"}); code != 0 {
		t.Fatalf("list %d %s", code, errb.String())
	}
	if !strings.Contains(out.String(), `"slug": "researcher"`) {
		t.Fatalf("json: %s", out.String())
	}
	out.Reset()
	if code := run(io, []string{"show", "researcher"}); code != 0 {
		t.Fatal(errb.String())
	}
	out.Reset()
	if code := run(io, []string{"delete", "researcher", "--yes"}); code != 0 {
		t.Fatalf("delete %d %s", code, errb.String())
	}
	if _, err := os.Stat(filepath.Join(dir, "home", "bots", "researcher")); !os.IsNotExist(err) {
		t.Fatalf("still exists: %v", err)
	}
}
