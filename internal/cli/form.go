package cli

import (
	"os"

	"charm.land/huh/v2"
	"golang.org/x/term"
)

func interactive() bool {
	return term.IsTerminal(int(os.Stdin.Fd())) && term.IsTerminal(int(os.Stdout.Fd()))
}

func spawnForm(slug, title, desc *string, coder *bool) error {
	groups := []*huh.Group{}
	if slug != nil && *slug == "" {
		groups = append(groups, huh.NewGroup(
			huh.NewInput().Title("Slug").Description("^[a-z][a-z0-9-]{0,62}$").Value(slug),
		))
	}
	groups = append(groups, huh.NewGroup(
		huh.NewInput().Title("Title").Value(title),
		huh.NewText().Title("Description").Value(desc),
		huh.NewConfirm().Title("Coder bot?").Description("Enables bash/edit (sandboxed on Linux)").Value(coder),
	))
	return huh.NewForm(groups...).Run()
}

func confirmForm(prompt string) (bool, error) {
	ok := false
	err := huh.NewForm(huh.NewGroup(
		huh.NewConfirm().Title(prompt).Value(&ok),
	)).Run()
	return ok, err
}
