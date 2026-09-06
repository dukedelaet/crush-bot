package spawn

import (
	"fmt"
	"io"
	"os"
	"strings"

	"charm.land/huh/v2"

	"github.com/dukedelaet/crush-bot/internal/roster"
)

// ErrAborted is huh's user-cancelled error.
var ErrAborted = huh.ErrUserAborted

func NormalizeSlug(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "@")
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, " ", "-")
	s = strings.ReplaceAll(s, "_", "-")
	return s
}

func Form(slug, title, desc *string, coder *bool) error {
	return FormIO(nil, nil, slug, title, desc, coder)
}

// FormAccessible is line-based (no nested Bubble Tea). Use it from the mesh
// TUI: tea.Exec's stdin is a cancelreader, which aborts Huh's TUI immediately.
func FormAccessible(in io.Reader, out io.Writer, slug, title, desc *string, coder *bool) error {
	return formIO(in, out, true, slug, title, desc, coder)
}

func FormIO(in io.Reader, out io.Writer, slug, title, desc *string, coder *bool) error {
	return formIO(in, out, false, slug, title, desc, coder)
}

func formIO(in io.Reader, out io.Writer, accessible bool, slug, title, desc *string, coder *bool) error {
	if slug == nil || title == nil || desc == nil || coder == nil {
		return fmt.Errorf("form: nil field")
	}
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Slug").
				Description("lowercase handle, e.g. researcher").
				Value(slug).
				Validate(func(s string) error {
					s = NormalizeSlug(s)
					if !roster.ValidSlug(s) {
						return fmt.Errorf("use a lowercase slug like researcher (a-z, 0-9, hyphens)")
					}
					return nil
				}),
			huh.NewInput().Title("Title").Value(title),
			huh.NewText().Title("Description").Value(desc),
			huh.NewConfirm().Title("Coder bot?").Description("bash/edit, sandboxed on Linux").Value(coder),
		),
	).WithAccessible(accessible)
	if in != nil {
		form = form.WithInput(in)
	}
	if out != nil {
		form = form.WithOutput(out)
	}
	if err := form.Run(); err != nil {
		return err
	}
	*slug = NormalizeSlug(*slug)
	if *title == "" && *slug != "" {
		*title = strings.ToUpper((*slug)[:1]) + (*slug)[1:]
	}
	return nil
}

func OpenTTY() (*os.File, error) {
	return os.OpenFile("/dev/tty", os.O_RDWR, 0)
}
