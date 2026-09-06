package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/hocoder-agents/crush-bot/internal/envelope"
	"github.com/hocoder-agents/crush-bot/internal/roster"
)

func TestInboxFolderCycle(t *testing.T) {
	var s inboxState
	if inboxFolders[s.folder] != "pending" {
		t.Fatal(s.folder)
	}
	s.nextFolder()
	if inboxFolders[s.folder] != "processing" {
		t.Fatal(inboxFolders[s.folder])
	}
	s.nextFolder()
	s.nextFolder()
	s.nextFolder()
	if inboxFolders[s.folder] != "pending" {
		t.Fatal(inboxFolders[s.folder])
	}
}

func TestInboxEmptyRender(t *testing.T) {
	s := inboxState{slug: "sophie"}
	out := renderInbox(s, 40, 8)
	if !containsAll(out, "@sophie", "empty") {
		t.Fatalf("%q", out)
	}
}

func TestRetryFailedMovesToPending(t *testing.T) {
	root := t.TempDir()
	home := roster.Home(root, "sophie")
	if err := os.MkdirAll(envelope.FailedDir(home), 0o700); err != nil {
		t.Fatal(err)
	}
	env := envelope.Envelope{
		ID:        "mail1",
		Kind:      "dm",
		From:      "researcher",
		To:        "sophie",
		Hop:       1,
		Body:      "hello",
		CreatedAt: time.Now().UTC(),
		Trace:     []string{"researcher"},
	}
	if _, err := envelope.Write(envelope.FailedDir(home), env); err != nil {
		t.Fatal(err)
	}
	if err := retryFailed(root, "sophie", "mail1"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(envelope.FailedDir(home), "mail1.json")); !os.IsNotExist(err) {
		t.Fatal("failed file still there")
	}
	pend, _, err := envelope.List(envelope.PendingDir(home))
	if err != nil {
		t.Fatal(err)
	}
	if len(pend) != 1 || pend[0].ID != "mail1" || pend[0].Attempt != 0 {
		t.Fatalf("%+v", pend)
	}
}

func TestLoadInbox(t *testing.T) {
	root := t.TempDir()
	home := roster.Home(root, "alpha")
	if err := os.MkdirAll(envelope.PendingDir(home), 0o700); err != nil {
		t.Fatal(err)
	}
	_, err := envelope.WritePending(home, envelope.Envelope{Kind: "dm", From: "beta", To: "alpha", Body: "hi"})
	if err != nil {
		t.Fatal(err)
	}
	s := loadInbox(root, "alpha")
	if len(s.envs[0]) != 1 {
		t.Fatalf("pending %d", len(s.envs[0]))
	}
	s.move(1)
	if s.cursor != 0 {
		t.Fatalf("cursor wrap %d", s.cursor)
	}
}

func containsAll(s string, parts ...string) bool {
	for _, p := range parts {
		if !strings.Contains(s, p) {
			return false
		}
	}
	return true
}
