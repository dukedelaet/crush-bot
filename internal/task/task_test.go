package task

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/dukedelaet/crush-bot/internal/roster"
)

func TestAssignIdempotent(t *testing.T) {
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, "bots"), 0o700)
	_, _, err := roster.Spawn(root, roster.SpawnOpts{Slug: "alpha"})
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = roster.Spawn(root, roster.SpawnOpts{Slug: "beta"})
	if err != nil {
		t.Fatal(err)
	}
	a, dup, err := Assign(root, AssignOpts{From: "alpha", To: "beta", Title: "t", Body: "b", Key: "k1", Hop: 1})
	if err != nil || dup {
		t.Fatalf("%+v %v %v", a, dup, err)
	}
	b, dup, err := Assign(root, AssignOpts{From: "alpha", To: "beta", Title: "t", Body: "b", Key: "k1", Hop: 1})
	if err != nil || !dup || b.ID != a.ID {
		t.Fatalf("dup %+v %v", b, dup)
	}
}

func TestCompleteReceipt(t *testing.T) {
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, "bots"), 0o700)
	roster.Spawn(root, roster.SpawnOpts{Slug: "alpha"})
	roster.Spawn(root, roster.SpawnOpts{Slug: "beta"})
	a, _, err := Assign(root, AssignOpts{From: "alpha", To: "beta", Title: "do it", Body: "please", Hop: 1})
	if err != nil {
		t.Fatal(err)
	}
	got, owner, err := Find(root, a.ID)
	if err != nil || owner != "beta" {
		t.Fatalf("%s %v", owner, err)
	}
	got.Status = "done"
	if err := Save(root, owner, got); err != nil {
		t.Fatal(err)
	}
	loaded, _ := Load(root, "beta", a.ID)
	if loaded.Status != "done" {
		t.Fatalf("%s", loaded.Status)
	}
}
