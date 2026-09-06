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
