package cli

import (
	"os"

	"charm.land/huh/v2"
	"golang.org/x/term"
)

func interactive() bool {
	return term.IsTerminal(int(os.Stdin.Fd())) && term.IsTerminal(int(os.Stdout.Fd()))
}

func confirmForm(prompt string) (bool, error) {
	ok := false
	err := huh.NewForm(huh.NewGroup(
		huh.NewConfirm().Title(prompt).Value(&ok),
	)).Run()
	return ok, err
}
