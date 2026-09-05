package envelope

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type Envelope struct {
	ID          string    `json:"id"`
	Kind        string    `json:"kind"`
	From        string    `json:"from"`
	To          string    `json:"to"`
	GroupID     *string   `json:"group_id"`
	Hop         int       `json:"hop"`
	ParentID    *string   `json:"parent_id"`
	TaskID      *string   `json:"task_id"`
	Attempt     int       `json:"attempt"`
	CreatedAt   time.Time `json:"created_at"`
	Attribution string    `json:"attribution"`
	Body        string    `json:"body"`
	Trace       []string  `json:"trace"`
}

func PendingDir(botHome string) string {
	return filepath.Join(botHome, "inbox", "pending")
}

func NewID() string {
	var b [10]byte
	_, _ = rand.Read(b[:])
	return fmt.Sprintf("%s%02x", time.Now().UTC().Format("20060102150405"), b)
}

func WritePending(botHome string, env Envelope) (string, error) {
	if env.ID == "" {
		env.ID = NewID()
	}
	if env.CreatedAt.IsZero() {
		env.CreatedAt = time.Now().UTC()
	}
	if env.Trace == nil {
		env.Trace = []string{}
	}
	dir := PendingDir(botHome)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	path := filepath.Join(dir, env.ID+".json")
	raw, err := json.MarshalIndent(env, "", "  ")
	if err != nil {
		return "", err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o600); err != nil {
		return "", err
	}
	if err := os.Rename(tmp, path); err != nil {
		return "", err
	}
	return env.ID, nil
}
