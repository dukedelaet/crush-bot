package cli

import (
	"fmt"

	"github.com/dukedelaet/crush-bot/internal/config"
	"github.com/dukedelaet/crush-bot/internal/crush"
	"github.com/dukedelaet/crush-bot/internal/roster"
)

func cmdKeepalive(io IO, args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(io.Err, errStyle.Render("usage: crushbot keepalive start|stop|status [slug|--all]"))
		return 2
	}
	p := config.ResolvePaths()
	cfg, err := config.Load(p)
	if err != nil {
		return fail(io, err)
	}
	bin, err := crushBin(cfg)
	if err != nil {
		return fail(io, err)
	}
	verb := args[0]
	all := false
	slug := ""
	for _, a := range args[1:] {
		if a == "--all" {
			all = true
			continue
		}
		if !hasPrefixDash(a) {
			slug = a
		}
	}
	bots, err := selectKeepaliveBots(p.Home, slug, all, verb == "start")
	if err != nil {
		return fail(io, err)
	}
	switch verb {
	case "start":
		for _, b := range bots {
			if err := crush.StartServer(bin, b, p.Home); err != nil {
				return fail(io, err)
			}
			fmt.Fprintln(io.Out, okStyle.Render("keepalive @"+b.Slug))
		}
		return 0
	case "stop":
		for _, b := range bots {
			if err := crush.StopServer(roster.Home(p.Home, b.Slug)); err != nil {
				fmt.Fprintln(io.Err, mutedStyle.Render("@"+b.Slug+": "+err.Error()))
				continue
			}
			fmt.Fprintln(io.Out, okStyle.Render("stopped @"+b.Slug))
		}
		return 0
	case "status":
		for _, b := range bots {
			home := roster.Home(p.Home, b.Slug)
			if crush.ServerLive(home) {
				fmt.Fprintln(io.Out, okStyle.Render("@"+b.Slug+" up  "+crush.HostURL(home)))
			} else {
				fmt.Fprintln(io.Out, mutedStyle.Render("@"+b.Slug+" down"))
			}
		}
		return 0
	default:
		fmt.Fprintln(io.Err, errStyle.Render("usage: crushbot keepalive start|stop|status [slug|--all]"))
		return 2
	}
}

func hasPrefixDash(s string) bool {
	return len(s) > 0 && s[0] == '-'
}

func selectKeepaliveBots(root, slug string, all, start bool) ([]roster.Bot, error) {
	if slug != "" {
		b, err := roster.Load(root, slug)
		if err != nil {
			return nil, err
		}
		return []roster.Bot{b}, nil
	}
	list, err := roster.List(root, true)
	if err != nil {
		return nil, err
	}
	if all {
		return list, nil
	}
	var out []roster.Bot
	for _, b := range list {
		if b.KeepAlive {
			out = append(out, b)
		}
	}
	if start && len(out) == 0 {
		return nil, fmt.Errorf("no bots with keepalive: true (pass a slug or --all)")
	}
	return out, nil
}
