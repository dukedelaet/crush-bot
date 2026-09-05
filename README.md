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

## Usage (scaffold)

```bash
crushbot            # Bubble Tea mesh (placeholder until the roster lands)
crushbot --help
crushbot init       # create $CRUSHBOT_HOME (~/.local/share/crushbot)
```

Override the data dir with `CRUSHBOT_HOME`. Config lives at `$XDG_CONFIG_HOME/crushbot/config.yaml`.

Design: [DESIGN.md](./DESIGN.md).
