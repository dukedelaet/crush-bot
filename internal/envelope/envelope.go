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

func Dir(botHome, state string) string {
	return filepath.Join(botHome, "inbox", state)
}

func PendingDir(botHome string) string { return Dir(botHome, "pending") }
func ProcessingDir(botHome string) string {
	return Dir(botHome, "processing")
}
func ArchiveDir(botHome string) string { return Dir(botHome, "archive") }
func FailedDir(botHome string) string  { return Dir(botHome, "failed") }

func ReadFile(path string) (Envelope, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Envelope{}, err
	}
	var env Envelope
	if err := json.Unmarshal(b, &env); err != nil {
		return Envelope{}, err
	}
	return env, nil
}

func List(dir string) ([]Envelope, []string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil, nil
		}
		return nil, nil, err
	}
	var envs []Envelope
	var paths []string
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".json" {
			continue
		}
		p := filepath.Join(dir, e.Name())
		env, err := ReadFile(p)
		if err != nil {
			continue
		}
		envs = append(envs, env)
		paths = append(paths, p)
	}
	return envs, paths, nil
}

func Move(src, destDir string) (string, error) {
	if err := os.MkdirAll(destDir, 0o700); err != nil {
		return "", err
	}
	dst := filepath.Join(destDir, filepath.Base(src))
	if err := os.Rename(src, dst); err != nil {
		return "", err
	}
	return dst, nil
}

func Write(dir string, env Envelope) (string, error) {
	if env.ID == "" {
		env.ID = NewID()
	}
	if env.CreatedAt.IsZero() {
		env.CreatedAt = time.Now().UTC()
	}
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
	return path, os.Rename(tmp, path)
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
