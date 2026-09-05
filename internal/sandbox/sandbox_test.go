package sandbox

import (
	"os"
	"strings"
	"testing"

	"github.com/dukedelaet/crush-bot/internal/roster"
)

func TestRequired(t *testing.T) {
	if Required(roster.Bot{}) {
		t.Fatal("default bot must not require sandbox")
	}
	if !Required(roster.Bot{Tools: roster.Tools{Bash: true}}) {
		t.Fatal("bash bot requires sandbox")
	}
	if Required(roster.Bot{Tools: roster.Tools{Bash: true}, Sandbox: "off"}) {
		t.Fatal("sandbox:off")
	}
}

func TestBwrapArgsNoHomeBind(t *testing.T) {
	root := t.TempDir()
	bot := roster.Bot{Slug: "coder", Tools: roster.Tools{Bash: true, Edit: true}}
	args := BwrapArgs("/usr/bin/crush", []string{"run", "hi"}, bot, root)
	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "--die-with-parent") {
		t.Fatalf("%s", joined)
	}
	if !strings.Contains(joined, "--bind") {
		t.Fatal("missing rw bind")
	}
	// operator HOME must not appear as a bind source besides sandbox-home under bot
	for i, a := range args {
		if a == "--bind" || a == "--ro-bind" {
			src := args[i+1]
			if src == osHome(t) {
				t.Fatalf("bound operator HOME %s", src)
			}
		}
	}
}

func osHome(t *testing.T) string {
	t.Helper()
	h, err := os.UserHomeDir()
	if err != nil || h == "" {
		return "/nonexistent-home"
	}
	return h
}
