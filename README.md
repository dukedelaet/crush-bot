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

```bash
crushbot spawn coder --coder --keepalive   # warm crush server (unix socket)
crushbot keepalive start --all
crushbot daemon install                    # systemd --user unit
crushbot group chat review                 # Bubble Tea room ( --plain for scripts )
```

Requires Crush ≥ 0.91.2 with a configured provider for `spawn` / `say` / `chat`. The daemon is required for mesh delivery (`message_bot` → inbox → `crush run`).

Override the data dir with `CRUSHBOT_HOME`. Config lives at `$XDG_CONFIG_HOME/crushbot/config.yaml`.

## Docs

- [soul.md](docs/soul.md) — identity files
- [mesh.md](docs/mesh.md) — DMs, tasks, groups, daemon
- [hermes-delta.md](docs/hermes-delta.md) — what is not a wire clone
- [DESIGN.md](./DESIGN.md) — full spec

`crushbot doctor --check` exits 2 if `needs_you.jsonl` is non-empty or any inbox `failed/` has files.

```bash
scripts/e2e-mesh.sh   # fake Crush, no provider
```
