package spawn

import (
	"bytes"
	"strings"
	"testing"
)

func TestNormalizeSlug(t *testing.T) {
	if g := NormalizeSlug(" @Research_Buddy "); g != "research-buddy" {
		t.Fatalf("%s", g)
	}
}

func TestFormAccessibleNormalizesSlug(t *testing.T) {
	// Huh's accessible PromptString uses bufio.Scanner, which buffers the
	// rest of a pipe, so only the first field is reliable here. On a real
	// TTY each prompt reads one line.
	var slug, title, desc string
	var coder bool
	in := strings.NewReader("Scribe\n")
	var out bytes.Buffer
	if err := FormAccessible(in, &out, &slug, &title, &desc, &coder); err != nil {
		t.Fatal(err)
	}
	if slug != "scribe" {
		t.Fatalf("slug %q", slug)
	}
	if title != "Scribe" {
		t.Fatalf("title %q", title)
	}
}

func TestFormAccessibleCancelled(t *testing.T) {
	var slug, title, desc string
	var coder bool
	err := FormAccessible(strings.NewReader(""), bytes.NewBuffer(nil), &slug, &title, &desc, &coder)
	if err != nil {
		t.Fatal(err)
	}
	if slug != "" {
		t.Fatalf("slug %q", slug)
	}
}
