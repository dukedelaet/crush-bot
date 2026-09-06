package cli

import (
	"fmt"
	"strings"

	"github.com/hocoder-agents/crush-bot/internal/config"
	"github.com/hocoder-agents/crush-bot/internal/envelope"
	"github.com/hocoder-agents/crush-bot/internal/roster"
)

func cmdMention(io IO, args []string) int {
	if len(args) < 3 {
		fmt.Fprintln(io.Err, errStyle.Render("usage: crushbot mention <bot> <target> <text>"))
		return 2
	}
	from, to := args[0], strings.TrimPrefix(args[1], "@")
	text := strings.Join(args[2:], " ")
	p := config.ResolvePaths()
	if !roster.Exists(p.Home, from) {
		return fail(io, fmt.Errorf("unknown bot %s", from))
	}
	if !roster.Exists(p.Home, to) {
		return fail(io, fmt.Errorf("unknown_bot"))
	}
	body := fmt.Sprintf("The operator asks you to message @%s. Compose your own wording. Substance: %s", to, text)
	id, err := envelope.WritePending(roster.Home(p.Home, from), envelope.Envelope{
		Kind: "mention_directive", From: "user", To: from, Hop: 0,
		Trace: []string{"user"}, Body: body,
		Attribution: "Operator mention directive:",
	})
	if err != nil {
		return fail(io, err)
	}
	fmt.Fprintln(io.Out, okStyle.Render("queued "+id+" → @"+from))
	return 0
}

func cmdBroadcast(io IO, args []string) int {
	text := strings.Join(args, " ")
	if strings.TrimSpace(text) == "" {
		fmt.Fprintln(io.Err, errStyle.Render("usage: crushbot broadcast <text>"))
		return 2
	}
	p := config.ResolvePaths()
	bots, err := roster.List(p.Home, false)
	if err != nil {
		return fail(io, err)
	}
	n := 0
	for _, b := range bots {
		_, err := envelope.WritePending(roster.Home(p.Home, b.Slug), envelope.Envelope{
			Kind: "broadcast", From: "user", To: b.Slug, Hop: 0,
			Trace: []string{"user"}, Body: text,
			Attribution: "Broadcast from operator:",
		})
		if err != nil {
			return fail(io, err)
		}
		n++
	}
	fmt.Fprintln(io.Out, okStyle.Render(fmt.Sprintf("queued broadcast to %d bots", n)))
	return 0
}
