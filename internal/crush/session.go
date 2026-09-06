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
	Name string `json:"name,omitempty"`
}

// Line is one transcript turn from session show --json.
type Line struct {
	Role string
	Text string
}

// LastAssistant is pinned from Crush v0.91.2 `session show --json` (last assistant text parts).
func LastAssistant(bin, cwd, dataDir, sessionID string, maxChars int) (string, error) {
	out, err := sessionShowJSON(bin, cwd, dataDir, sessionID)
	if err != nil {
		return "", err
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

func SessionTranscript(bin, cwd, dataDir, sessionID string, maxMsgs int) ([]Line, error) {
	raw, err := sessionShowJSON(bin, cwd, dataDir, sessionID)
	if err != nil {
		return nil, err
	}
	return parseTranscript(raw, maxMsgs)
}

func sessionShowJSON(bin, cwd, dataDir, sessionID string) ([]byte, error) {
	args := []string{"session", "show"}
	if sessionID != "" {
		args = append(args, sessionID)
	} else {
		args = []string{"session", "last"}
	}
	args = append(args, "--json", "--cwd", cwd, "--data-dir", dataDir)
	out, err := exec.Command(bin, args...).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("session show: %w (%s)", err, strings.TrimSpace(string(out)))
	}
	return out, nil
}

func parseTranscript(raw []byte, maxMsgs int) ([]Line, error) {
	var p showPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, fmt.Errorf("session show json: %w", err)
	}
	var lines []Line
	for _, m := range p.Messages {
		var b strings.Builder
		for _, part := range m.Parts {
			switch part.Type {
			case "text":
				if part.Text != "" {
					if b.Len() > 0 {
						b.WriteByte('\n')
					}
					b.WriteString(part.Text)
				}
			case "tool", "tool_use", "tool_call":
				name := part.Name
				if name == "" {
					name = "tool"
				}
				if b.Len() > 0 {
					b.WriteByte('\n')
				}
				b.WriteString("[" + name + "]")
			}
		}
		text := strings.TrimSpace(b.String())
		if text == "" {
			continue
		}
		role := m.Role
		if role == "" {
			role = "other"
		}
		lines = append(lines, Line{Role: role, Text: text})
	}
	if maxMsgs > 0 && len(lines) > maxMsgs {
		lines = lines[len(lines)-maxMsgs:]
	}
	return lines, nil
}
