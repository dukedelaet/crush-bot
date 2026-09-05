# crushbot

Terminal clone of [Hermes Agent Bot Mode](https://hermes-agent.nousresearch.com/docs/user-guide/bot-mode), using [Crush](https://github.com/charmbracelet/crush) as the agent runtime.

This is a **Charm app** (Bubble Tea, Lip Gloss, Huh). It does not use cobra.

## Requirements

- Go 1.24+
- Crush ≥ 0.91.2 on `$PATH` (needed once spawn/say land; not required for `init`)

## Build

```bash
go build -o crushbot ./cmd/crushbot
```

## Usage

```bash
crushbot init
crushbot spawn researcher --title Researcher
crushbot list
crushbot say researcher "Who are you?"
crushbot chat researcher          # Crush TUI under turn.lock
crushbot doctor
crushbot                          # Bubble Tea mesh
```

```bash
./crushbot daemon start   # wakes bots that have pending DMs
./crushbot inbox
./crushbot group enable
./crushbot group create review researcher coder
./crushbot group chat review --plain
```

`--coder` bots are wrapped in `bwrap` on Linux (pass `--sandbox-off` to override).

Requires Crush ≥ 0.91.2 with a configured provider for `spawn` / `say` / `chat`. The daemon is required for mesh delivery (`message_bot` → inbox → `crush run`).

Override the data dir with `CRUSHBOT_HOME`. Config lives at `$XDG_CONFIG_HOME/crushbot/config.yaml`.

Design: [DESIGN.md](./DESIGN.md).
