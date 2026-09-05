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

type showPayload struct {
	Messages []showMsg `json:"messages"`
}

type showMsg struct {
	Role  string     `json:"role"`
	Parts []showPart `json:"parts"`
}

type showPart struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// LastAssistant is pinned from Crush v0.91.2 `session show --json` (last assistant text parts).
func LastAssistant(bin, cwd, dataDir, sessionID string, maxChars int) (string, error) {
	args := []string{"session", "show"}
	if sessionID != "" {
		args = append(args, sessionID)
	} else {
		args = []string{"session", "last"}
	}
	args = append(args, "--json", "--cwd", cwd, "--data-dir", dataDir)
	out, err := exec.Command(bin, args...).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("session show: %w (%s)", err, strings.TrimSpace(string(out)))
	}
	var p showPayload
	if err := json.Unmarshal(out, &p); err != nil {
		return "", fmt.Errorf("session show json: %w", err)
	}
	var text string
	for i := len(p.Messages) - 1; i >= 0; i-- {
		if p.Messages[i].Role != "assistant" {
			continue
		}
		var b strings.Builder
		for _, part := range p.Messages[i].Parts {
			if part.Type == "text" {
				b.WriteString(part.Text)
			}
		}
		text = strings.TrimSpace(b.String())
		if text != "" {
			break
		}
	}
	if maxChars > 0 && len(text) > maxChars {
		// character cap (design: 4096 chars, not KiB)
		r := []rune(text)
		if len(r) > maxChars {
			text = string(r[:maxChars])
		}
	}
	return text, nil
}
