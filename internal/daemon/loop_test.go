package daemon

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/hocoder-agents/crush-bot/internal/config"
	"github.com/hocoder-agents/crush-bot/internal/envelope"
	"github.com/hocoder-agents/crush-bot/internal/protocol"
	"github.com/hocoder-agents/crush-bot/internal/roster"
)

func fakeCrush(t *testing.T) string {
	t.Helper()
	src, err := os.ReadFile(filepath.Join("..", "crush", "testdata", "fake-crush.sh"))
	if err != nil {
		t.Fatal(err)
	}
	dst := filepath.Join(t.TempDir(), "crush")
	if err := os.WriteFile(dst, src, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("FAKE_CRUSH_STATE", filepath.Join(t.TempDir(), "st"))
	return dst
}

func TestOnceArchivesAndReceipt(t *testing.T) {
	bin := fakeCrush(t)
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, "bots"), 0o700)
	alpha, _, err := roster.Spawn(root, roster.SpawnOpts{Slug: "alpha"})
	if err != nil {
		t.Fatal(err)
	}
	beta, _, err := roster.Spawn(root, roster.SpawnOpts{Slug: "beta"})
	if err != nil {
		t.Fatal(err)
	}
	beta.CanonicalSessionID = "11111111-1111-4111-8111-111111111111"
	if err := roster.Save(root, beta); err != nil {
		t.Fatal(err)
	}
	_ = protocol.Write(protocol.Options{Root: root, Bot: alpha})
	_ = protocol.Write(protocol.Options{Root: root, Bot: beta})
	id, err := envelope.WritePending(roster.Home(root, "beta"), envelope.Envelope{
		Kind: "dm", From: "alpha", To: "beta", Hop: 1, Trace: []string{"user", "alpha"},
		Body: "please look this up", Attribution: "Message from alpha (@alpha):",
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	n, err := Once(ctx, Options{Root: root, Bin: bin, Cfg: config.Default()})
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("woke %d", n)
	}
	if _, err := os.Stat(filepath.Join(roster.Home(root, "beta"), "inbox", "archive", id+".json")); err != nil {
		t.Fatalf("not archived: %v", err)
	}
	pend, _, err := envelope.List(envelope.PendingDir(roster.Home(root, "alpha")))
	if err != nil {
		t.Fatal(err)
	}
	if len(pend) != 1 || pend[0].Kind != "receipt" {
		t.Fatalf("receipt: %+v", pend)
	}
	if pend[0].From != "beta" || pend[0].To != "alpha" {
		t.Fatalf("%+v", pend[0])
	}
	if pend[0].Body != "online" {
		t.Fatalf("receipt body: %q", pend[0].Body)
	}
}

func TestSkipReceiptOnReceiptWake(t *testing.T) {
	bin := fakeCrush(t)
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, "bots"), 0o700)
	a, _, _ := roster.Spawn(root, roster.SpawnOpts{Slug: "alpha"})
	b, _, _ := roster.Spawn(root, roster.SpawnOpts{Slug: "beta"})
	b.CanonicalSessionID = "11111111-1111-4111-8111-111111111111"
	_ = roster.Save(root, b)
	_ = protocol.Write(protocol.Options{Root: root, Bot: a})
	_ = protocol.Write(protocol.Options{Root: root, Bot: b})
	_, _ = envelope.WritePending(roster.Home(root, "beta"), envelope.Envelope{
		Kind: "receipt", From: "alpha", To: "beta", Hop: 2, Trace: []string{"user", "alpha"},
		Body: "FYI",
	})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := Once(ctx, Options{Root: root, Bin: bin, Cfg: config.Default()})
	if err != nil {
		t.Fatal(err)
	}
	pend, _, _ := envelope.List(envelope.PendingDir(roster.Home(root, "alpha")))
	if len(pend) != 0 {
		t.Fatalf("nested receipt: %+v", pend)
	}
}

func TestClassifyRetryable(t *testing.T) {
	r, ok := Classify(1, "HTTP 500 internal server error", "")
	if !ok || r != "provider_server_error" {
		t.Fatalf("%s %v", r, ok)
	}
	r, ok = Classify(1, "invalid api key", "")
	if ok || r != "provider_auth_or_access" {
		t.Fatalf("%s %v", r, ok)
	}
}
