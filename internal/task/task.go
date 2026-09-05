package task

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/dukedelaet/crush-bot/internal/envelope"
	"github.com/dukedelaet/crush-bot/internal/roster"
)

type Task struct {
	ID             string     `json:"id"`
	Title          string     `json:"title"`
	From           string     `json:"from"`
	To             string     `json:"to"`
	Status         string     `json:"status"`
	Priority       string     `json:"priority"`
	Body           string     `json:"body"`
	ParentID       *string    `json:"parent_id"`
	Hop            int        `json:"hop"`
	IdempotencyKey string     `json:"idempotency_key"`
	ClaimTTLS      int        `json:"claim_ttl_s"`
	ClaimedAt      *time.Time `json:"claimed_at"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	Result         *string    `json:"result"`
	Error          *string    `json:"error"`
	Reason         *string    `json:"reason"`
}

func Dir(botHome string) string { return filepath.Join(botHome, "tasks") }

func pathFor(root, ownerSlug, id string) string {
	return filepath.Join(roster.Home(root, ownerSlug), "tasks", id+".json")
}

func NewID() string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return "tsk_" + time.Now().UTC().Format("20060102150405") + hex.EncodeToString(b[:])
}

func Save(root, owner string, t Task) error {
	if err := os.MkdirAll(Dir(roster.Home(root, owner)), 0o700); err != nil {
		return err
	}
	t.UpdatedAt = time.Now().UTC()
	raw, err := json.MarshalIndent(t, "", "  ")
	if err != nil {
		return err
	}
	p := pathFor(root, owner, t.ID)
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, p)
}

func Load(root, owner, id string) (Task, error) {
	b, err := os.ReadFile(pathFor(root, owner, id))
	if err != nil {
		return Task{}, err
	}
	var t Task
	return t, json.Unmarshal(b, &t)
}

func Find(root, id string) (Task, string, error) {
	bots, err := roster.List(root, true)
	if err != nil {
		return Task{}, "", err
	}
	for _, b := range bots {
		t, err := Load(root, b.Slug, id)
		if err == nil {
			return t, b.Slug, nil
		}
	}
	return Task{}, "", fmt.Errorf("unknown task %s", id)
}

func ListFor(root, slug string) ([]Task, error) {
	bots, err := roster.List(root, true)
	if err != nil {
		return nil, err
	}
	var out []Task
	for _, b := range bots {
		entries, err := os.ReadDir(Dir(roster.Home(root, b.Slug)))
		if err != nil {
			continue
		}
		for _, e := range entries {
			if filepath.Ext(e.Name()) != ".json" {
				continue
			}
			t, err := Load(root, b.Slug, e.Name()[:len(e.Name())-5])
			if err != nil {
				continue
			}
			if t.To == slug || t.From == slug || b.Slug == slug {
				out = append(out, t)
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out, nil
}

func FindIdempotent(root, from, to, key string) (Task, bool) {
	if key == "" {
		return Task{}, false
	}
	list, err := ListFor(root, to)
	if err != nil {
		return Task{}, false
	}
	for _, t := range list {
		if t.From == from && t.To == to && t.IdempotencyKey == key {
			return t, true
		}
	}
	return Task{}, false
}

type AssignOpts struct {
	From, To, Title, Body, Priority, Key string
	Hop, ClaimTTL                        int
	ParentID                             *string
}

func Assign(root string, o AssignOpts) (Task, bool, error) {
	if o.Priority == "" {
		o.Priority = "normal"
	}
	if existing, ok := FindIdempotent(root, o.From, o.To, o.Key); ok {
		return existing, true, nil
	}
	now := time.Now().UTC()
	t := Task{
		ID:             NewID(),
		Title:          o.Title,
		From:           o.From,
		To:             o.To,
		Status:         "queued",
		Priority:       o.Priority,
		Body:           o.Body,
		ParentID:       o.ParentID,
		Hop:            o.Hop,
		IdempotencyKey: o.Key,
		ClaimTTLS:      o.ClaimTTL,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if t.ClaimTTLS <= 0 {
		t.ClaimTTLS = 900
	}
	if err := Save(root, o.To, t); err != nil {
		return Task{}, false, err
	}
	tid := t.ID
	_, err := envelope.WritePending(roster.Home(root, o.To), envelope.Envelope{
		Kind:        "task",
		From:        o.From,
		To:          o.To,
		Hop:         o.Hop,
		TaskID:      &tid,
		Body:        o.Title + "\n\n" + o.Body,
		Trace:       []string{"user", o.From},
		Attribution: fmt.Sprintf("Task from %s (@%s): %s", o.From, o.From, o.Title),
	})
	return t, false, err
}
