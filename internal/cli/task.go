package cli

import (
	"fmt"
	"strings"

	"github.com/dukedelaet/crush-bot/internal/config"
	"github.com/dukedelaet/crush-bot/internal/envelope"
	"github.com/dukedelaet/crush-bot/internal/roster"
	"github.com/dukedelaet/crush-bot/internal/task"
)

func cmdTasks(io IO, args []string) int {
	status := ""
	slug := ""
	for _, a := range args {
		if strings.HasPrefix(a, "--status=") {
			status = strings.TrimPrefix(a, "--status=")
			continue
		}
		if a == "--status" {
			continue
		}
		if !strings.HasPrefix(a, "-") && slug == "" {
			slug = a
		}
	}
	p := config.ResolvePaths()
	if slug == "" {
		bots, err := roster.List(p.Home, true)
		if err != nil {
			return fail(io, err)
		}
		for _, b := range bots {
			printTasks(io, p.Home, b.Slug, status)
		}
		return 0
	}
	printTasks(io, p.Home, slug, status)
	return 0
}

func printTasks(io IO, root, slug, status string) {
	list, err := task.ListFor(root, slug)
	if err != nil {
		fmt.Fprintln(io.Err, err.Error())
		return
	}
	fmt.Fprintln(io.Out, headStyle.Render("@"+slug))
	n := 0
	for _, t := range list {
		if status != "" && t.Status != status {
			continue
		}
		n++
		fmt.Fprintf(io.Out, "  %s  %s  %s → @%s  %s\n", t.ID, t.Status, t.Title, t.To, t.From)
	}
	if n == 0 {
		fmt.Fprintln(io.Out, mutedStyle.Render("  (none)"))
	}
}

func cmdTask(io IO, args []string) int {
	if len(args) < 2 {
		fmt.Fprintln(io.Err, errStyle.Render("usage: crushbot task show|retry|unblock <id>"))
		return 2
	}
	p := config.ResolvePaths()
	switch args[0] {
	case "show":
		t, _, err := task.Find(p.Home, args[1])
		if err != nil {
			return fail(io, err)
		}
		fmt.Fprintf(io.Out, "%s  %s\n%s\nfrom=@%s to=@%s hop=%d status=%s\n", t.ID, t.Title, t.Body, t.From, t.To, t.Hop, t.Status)
		return 0
	case "retry":
		t, owner, err := task.Find(p.Home, args[1])
		if err != nil {
			return fail(io, err)
		}
		t.Status = "queued"
		t.Reason = nil
		if err := task.Save(p.Home, owner, t); err != nil {
			return fail(io, err)
		}
		tid := t.ID
		_, _ = envelope.WritePending(roster.Home(p.Home, t.To), envelope.Envelope{
			Kind: "task", From: "user", To: t.To, Hop: 0, TaskID: &tid, Body: "operator retry: " + t.Title,
		})
		fmt.Fprintln(io.Out, okStyle.Render("retry "+t.ID))
		return 0
	case "unblock":
		t, owner, err := task.Find(p.Home, args[1])
		if err != nil {
			return fail(io, err)
		}
		t.Status = "queued"
		t.Reason = nil
		if err := task.Save(p.Home, owner, t); err != nil {
			return fail(io, err)
		}
		fmt.Fprintln(io.Out, okStyle.Render("unblocked "+t.ID))
		return 0
	default:
		fmt.Fprintln(io.Err, errStyle.Render("usage: crushbot task show|retry|unblock <id>"))
		return 2
	}
}
