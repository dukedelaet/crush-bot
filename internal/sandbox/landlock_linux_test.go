//go:build linux

package sandbox

import (
	"os"
	"path/filepath"
	"testing"
	"unsafe"

	"golang.org/x/sys/unix"
)

func TestAddPathFileWithDirectoryRights(t *testing.T) {
	if err := landlockAvailable(); err != nil {
		t.Skip(err)
	}
	attr := rulesetAttr{HandledAccessFS: handledFS}
	r1, _, errno := unix.Syscall(sysLandlockCreateRuleset, uintptr(unsafe.Pointer(&attr)), unsafe.Sizeof(attr), 0)
	if errno != 0 {
		t.Fatalf("create ruleset: %v", errno)
	}
	fd := int(r1)
	defer unix.Close(fd)

	dir := t.TempDir()
	file := filepath.Join(dir, "needs_you.jsonl")
	if err := os.WriteFile(file, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := addPath(fd, file, accessRW); err != nil {
		t.Fatalf("file rule: %v", err)
	}
	if err := addPath(fd, dir, accessRW); err != nil {
		t.Fatalf("dir rule: %v", err)
	}
}

func TestAddPathDevNull(t *testing.T) {
	if err := landlockAvailable(); err != nil {
		t.Skip(err)
	}
	attr := rulesetAttr{HandledAccessFS: handledFS}
	r1, _, errno := unix.Syscall(sysLandlockCreateRuleset, uintptr(unsafe.Pointer(&attr)), unsafe.Sizeof(attr), 0)
	if errno != 0 {
		t.Fatalf("create ruleset: %v", errno)
	}
	fd := int(r1)
	defer unix.Close(fd)
	if err := addPath(fd, "/dev/null", accessFile); err != nil {
		t.Fatalf("dev/null: %v", err)
	}
}

func TestWalkDirs(t *testing.T) {
	got := walkDirs("/home/duke/.local/share/crush/crush.json")
	want := map[string]bool{
		"/home/duke/.local/share/crush": true,
		"/home/duke/.local/share":       true,
		"/home/duke/.local":             true,
		"/home/duke":                    true,
		"/home":                         true,
		"/":                             true,
	}
	for _, p := range got {
		if !want[p] {
			t.Fatalf("unexpected %q in %v", p, got)
		}
		delete(want, p)
	}
	if len(want) != 0 {
		t.Fatalf("missing %v from %v", want, got)
	}
}
