package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/hocoder-agents/crush-bot/internal/envelope"
	"github.com/hocoder-agents/crush-bot/internal/roster"
)

var inboxFolders = []string{"pending", "processing", "archive", "failed"}

type inboxState struct {
	slug   string
	folder int
	cursor int
	envs   [4][]envelope.Envelope
}

func loadInbox(root, slug string) inboxState {
	s := inboxState{slug: slug}
	home := roster.Home(root, slug)
	for i, st := range inboxFolders {
		envs, _, _ := envelope.List(envelope.Dir(home, st))
		s.envs[i] = envs
	}
	return s
}

func (s *inboxState) current() []envelope.Envelope {
	if s.folder < 0 || s.folder >= len(inboxFolders) {
		return nil
	}
	return s.envs[s.folder]
}

func (s *inboxState) selected() (envelope.Envelope, bool) {
	cur := s.current()
	if len(cur) == 0 || s.cursor < 0 || s.cursor >= len(cur) {
		return envelope.Envelope{}, false
	}
	return cur[s.cursor], true
}

func (s *inboxState) nextFolder() {
	s.folder = (s.folder + 1) % len(inboxFolders)
	s.cursor = 0
}

func (s *inboxState) move(delta int) {
	cur := s.current()
	if len(cur) == 0 {
		s.cursor = 0
		return
	}
	s.cursor = (s.cursor + delta) % len(cur)
	if s.cursor < 0 {
		s.cursor += len(cur)
	}
}

func (s *inboxState) clamp() {
	cur := s.current()
	if s.cursor >= len(cur) {
		s.cursor = 0
	}
}

func retryFailed(root, slug, id string) error {
	home := roster.Home(root, slug)
	src := filepath.Join(envelope.FailedDir(home), id+".json")
	env, err := envelope.ReadFile(src)
	if err != nil {
		return err
	}
	env.Attempt = 0
	if _, err := envelope.WritePending(home, env); err != nil {
		return err
	}
	return os.Remove(src)
}

func renderInbox(s inboxState, width, height int) string {
	var b strings.Builder
	fmt.Fprintf(&b, "@%s  inbox\n", s.slug)
	for i, name := range inboxFolders {
		label := fmt.Sprintf("%s %d", name, len(s.envs[i]))
		if i == s.folder {
			label = selStyle.Render(label)
		} else {
			label = mutedStyle.Render(label)
		}
		if i > 0 {
			b.WriteString("  ")
		}
		b.WriteString(label)
	}
	b.WriteByte('\n')
	b.WriteByte('\n')
	cur := s.current()
	if len(cur) == 0 {
		fmt.Fprintf(&b, "%s\n", mutedStyle.Render("empty"))
	}
	for i, e := range cur {
		mark := " "
		if i == s.cursor {
			mark = "▸"
		}
		from := e.From
		if from == "" {
			from = "user"
		}
		line := fmt.Sprintf("%s %s  from=@%s  hop=%d", mark, e.Kind, from, e.Hop)
		if i == s.cursor {
			line = selStyle.Render(line)
		}
		fmt.Fprintln(&b, line)
		if i == s.cursor {
			body := strings.ReplaceAll(e.Body, "\n", " ")
			body = clipRunes(body, max(20, width-2))
			fmt.Fprintf(&b, "  %s\n", mutedStyle.Render(body))
		}
	}
	out := b.String()
	lines := strings.Split(out, "\n")
	for len(lines) < height {
		lines = append(lines, "")
	}
	if len(lines) > height {
		lines = lines[:height]
	}
	return strings.Join(lines, "\n")
}

func clipRunes(s string, n int) string {
	if n <= 0 {
		return ""
	}
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	r := []rune(s)
	if n < 2 {
		return string(r[:n])
	}
	return string(r[:n-1]) + "…"
}
