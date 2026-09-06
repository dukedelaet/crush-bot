package spawn

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/dukedelaet/crush-bot/internal/config"
	"github.com/dukedelaet/crush-bot/internal/crush"
	"github.com/dukedelaet/crush-bot/internal/protocol"
	"github.com/dukedelaet/crush-bot/internal/roster"
	"github.com/dukedelaet/crush-bot/internal/sandbox"
)

type Opts struct {
	Slug        string
	Title       string
	Description string
	Model       string
	Project     string
	CloneFrom   string
	Coder       bool
	Sandbox     string
	KeepAlive   bool
}

type Result struct {
	Bot     roster.Bot
	Warns   []string
	BootErr error
}

func Create(root string, cfg config.Config, o Opts) (Result, error) {
	var out Result
	bin, err := crush.LookPath(cfg.CrushPath)
	if err != nil {
		return out, err
	}
	if err := crush.RequireMin(bin, cfg.MinCrushVersion); err != nil {
		return out, err
	}
	if err := crush.HasProviders(bin); err != nil {
		return out, err
	}
	if o.Coder && o.Sandbox != "off" {
		if err := sandbox.Available(); err != nil {
			return out, fmt.Errorf("%w (or pass --sandbox-off)", err)
		}
	}
	bot, warns, err := roster.Spawn(root, roster.SpawnOpts{
		Slug:        o.Slug,
		Title:       o.Title,
		Description: o.Description,
		Model:       o.Model,
		Project:     o.Project,
		CloneFrom:   o.CloneFrom,
		Coder:       o.Coder,
		Sandbox:     o.Sandbox,
		KeepAlive:   o.KeepAlive,
		MaxBots:     cfg.MaxBots,
		SoulMax:     cfg.SoulMaxBytes,
	})
	if err != nil {
		return out, err
	}
	out.Warns = warns
	exe, _ := os.Executable()
	all, _ := roster.List(root, true)
	if err := protocol.Write(protocol.Options{
		Root:         root,
		Bot:          bot,
		Teammates:    all,
		Tasks:        cfg.Experimental.Tasks,
		Groups:       cfg.Experimental.Groups,
		IncludeMCP:   true,
		CrushbotPath: exe,
		SoulMax:      cfg.SoulMaxBytes,
	}); err != nil {
		return out, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), cfg.TurnLockTimeout+3*time.Minute)
	defer cancel()
	bot, err = crush.Bootstrap(ctx, crush.RunOpts{
		Bot:     bot,
		Root:    root,
		Bin:     bin,
		Timeout: cfg.TurnLockTimeout,
		Debug:   os.Getenv("CRUSHBOT_DEBUG") == "1",
		Yolo:    bot.Unattended == "yolo",
	})
	if err != nil {
		out.Bot = bot
		out.BootErr = err
		return out, nil
	}
	if bot.KeepAlive {
		if err := crush.StartServer(bin, bot, root); err != nil {
			out.Warns = append(out.Warns, "keepalive: "+err.Error())
		}
	}
	out.Bot = bot
	return out, nil
}

func FromForm(root string, cfg config.Config) (Result, error) {
	var slug, title, desc string
	var coder bool
	if err := Form(&slug, &title, &desc, &coder); err != nil {
		return Result{}, err
	}
	if slug == "" {
		return Result{}, fmt.Errorf("cancelled")
	}
	return Create(root, cfg, Opts{
		Slug:        slug,
		Title:       title,
		Description: desc,
		Coder:       coder,
	})
}
