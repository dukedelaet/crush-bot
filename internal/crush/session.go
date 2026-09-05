package crush

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// LastMeta is pinned from Crush v0.91.2 `session last --json`.
type LastMeta struct {
	ID    string `json:"id"`
	UUID  string `json:"uuid"`
	Title string `json:"title"`
}

type lastPayload struct {
	Meta LastMeta `json:"meta"`
}

func SessionLast(bin, cwd, dataDir string) (LastMeta, error) {
	cmd := exec.Command(bin, "session", "last", "--json", "--cwd", cwd, "--data-dir", dataDir)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return LastMeta{}, fmt.Errorf("session last: %w (%s)", err, strings.TrimSpace(string(out)))
	}
	var p lastPayload
	if err := json.Unmarshal(out, &p); err != nil {
		return LastMeta{}, fmt.Errorf("session last json: %w (body %q)", err, truncate(string(out), 200))
	}
	if p.Meta.UUID == "" {
		return LastMeta{}, fmt.Errorf("session last: missing meta.uuid")
	}
	return p.Meta, nil
}

func SessionRename(bin, cwd, dataDir, id, title string) error {
	cmd := exec.Command(bin, "session", "rename", id, title, "--cwd", cwd, "--data-dir", dataDir)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("session rename: %w (%s)", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func HasProviders(bin string) error {
	cmd := exec.Command(bin, "models")
	out, err := cmd.CombinedOutput()
	body := string(out)
	if err != nil {
		if strings.Contains(strings.ToLower(body), "no providers configured") {
			return fmt.Errorf("no providers configured")
		}
		return fmt.Errorf("crush models: %w (%s)", err, strings.TrimSpace(body))
	}
	if strings.Contains(strings.ToLower(body), "no providers configured") {
		return fmt.Errorf("no providers configured")
	}
	lines := 0
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.Contains(line, "(not configured)") {
			continue
		}
		lines++
	}
	if lines == 0 {
		return fmt.Errorf("no providers configured")
	}
	return nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
