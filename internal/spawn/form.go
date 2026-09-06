package spawn

import "charm.land/huh/v2"

func Form(slug, title, desc *string, coder *bool) error {
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
