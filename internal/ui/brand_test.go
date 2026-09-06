package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/hocoder-agents/crush-bot/internal/version"
)

func TestBrandLine(t *testing.T) {
	plain := ansi.Strip(brandLine())
	if !strings.Contains(plain, "crushbot") {
		t.Fatalf("missing name: %q", plain)
	}
	if !strings.Contains(plain, version.Version) {
		t.Fatalf("missing version: %q", plain)
	}
	if !strings.Contains(plain, "💗") {
		t.Fatalf("missing heart: %q", plain)
	}
}
