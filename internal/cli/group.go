package cli

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/dukedelaet/crush-bot/internal/config"
	"github.com/dukedelaet/crush-bot/internal/group"
	"github.com/dukedelaet/crush-bot/internal/protocol"
	"github.com/dukedelaet/crush-bot/internal/roster"
)

func cmdGroup(io IO, args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(io.Err, errStyle.Render("usage: crushbot group enable|create|list|chat|disband"))
		return 2
	}
	p := config.ResolvePaths()
	cfg, err := config.Load(p)
	if err != nil {
		return fail(io, err)
	}
	switch args[0] {
	case "enable":
		cfg.Experimental.Groups = true
		if err := config.Save(p, cfg); err != nil {
			return fail(io, err)
		}
		_ = config.EnsureHome(p)
		_ = config.Save(config.Paths{ConfigDir: p.Home, ConfigFile: p.Home + "/config.yaml"}, cfg)
		bots, _ := roster.List(p.Home, true)
		for _, b := range bots {
			_ = protocol.Write(protocol.Options{Root: p.Home, Bot: b, Teammates: bots, Tasks: cfg.Experimental.Tasks, Groups: true, IncludeMCP: true, CrushbotPath: "crushbot"})
		}
		fmt.Fprintln(io.Out, okStyle.Render("experimental.groups enabled"))
		return 0
	case "create":
		if !cfg.Experimental.Groups {
			fmt.Fprintln(io.Err, errStyle.Render("enable experimental.groups first: crushbot group enable"))
			return 1
		}
		if len(args) < 4 {
			fmt.Fprintln(io.Err, errStyle.Render("usage: crushbot group create <name> <member> <member> [members...]"))
			return 2
		}
		g, err := group.Create(p.Home, args[1], args[2:])
		if err != nil {
			return fail(io, err)
		}
		fmt.Fprintln(io.Out, okStyle.Render("created group "+g.ID+" members "+strings.Join(g.Members, ",")))
		return 0
	case "list":
		if !cfg.Experimental.Groups {
			fmt.Fprintln(io.Err, errStyle.Render("enable experimental.groups first: crushbot group enable"))
			return 1
		}
		gs, err := group.List(p.Home)
		if err != nil {
			return fail(io, err)
		}
		if len(gs) == 0 {
			fmt.Fprintln(io.Out, mutedStyle.Render("no groups"))
			return 0
		}
		for _, g := range gs {
			fmt.Fprintf(io.Out, "%s  %s  %s\n", g.ID, g.Name, strings.Join(g.Members, ","))
		}
		return 0
	case "disband":
		if len(args) != 2 {
			fmt.Fprintln(io.Err, errStyle.Render("usage: crushbot group disband <id>"))
			return 2
		}
		if err := group.Delete(p.Home, args[1]); err != nil {
			return fail(io, err)
		}
		fmt.Fprintln(io.Out, okStyle.Render("disbanded "+args[1]))
		return 0
	case "chat":
		plain := false
		id := ""
		for _, a := range args[1:] {
			if a == "--plain" {
				plain = true
				continue
			}
			if !strings.HasPrefix(a, "-") {
				id = a
			}
		}
		if id == "" {
			fmt.Fprintln(io.Err, errStyle.Render("usage: crushbot group chat <id> [--plain]"))
			return 2
		}
		if !cfg.Experimental.Groups {
			fmt.Fprintln(io.Err, errStyle.Render("enable experimental.groups first: crushbot group enable"))
			return 1
		}
		g, err := group.Load(p.Home, id)
		if err != nil {
			return fail(io, err)
		}
		if !plain && isTTY(os.Stdout) {
			fmt.Fprintln(io.Out, mutedStyle.Render("plain prompt loop (host TUI). Ctrl-D to exit. Room continues in daemon if running."))
		}
		bin, err := crushBin(cfg)
		if err != nil {
			return fail(io, err)
		}
		sc := bufio.NewScanner(io.In)
		if io.In == nil {
			sc = bufio.NewScanner(os.Stdin)
		}
		fmt.Fprintln(io.Out, mutedStyle.Render("group @"+g.ID+" — type a line, empty to skip"))
		for sc.Scan() {
			line := sc.Text()
			if strings.TrimSpace(line) == "" {
				continue
			}
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
			err := group.RunUntilSettle(ctx, cfg, bin, p.Home, g, line)
			cancel()
			if err != nil {
				fmt.Fprintln(io.Err, errStyle.Render(err.Error()))
			}
			lines, _ := group.ReadTranscript(p.Home, g.ID)
			for _, l := range lines {
				fmt.Fprintf(io.Out, "[%s] %s: %s\n", l.Kind, l.From, l.Body)
			}
		}
		return 0
	default:
		fmt.Fprintln(io.Err, errStyle.Render("usage: crushbot group enable|create|list|chat|disband"))
		return 2
	}
}
