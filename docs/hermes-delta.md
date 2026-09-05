# Semantic delta vs Hermes Bot Mode

crushbot clones the *product idea*, not the transport.

| Hermes | crushbot |
| --- | --- |
| `SOUL.md` replaces identity slot #1 | `soul.md` overlay + PreToolUse `{"context":…}` on Crush’s coder prompt |
| `message_agent` immediately spawns `hermes -p` | Mailbox JSON + daemon `crush run` |
| Reply is a completion notification on the sender’s next turn | FYI `kind: receipt` after a successful DM wake |
| No hop/trace in `message_agent` | Hop 8 + fan-out (mailbox can ping-pong) |
| Desktop is the courier; profiles claim no daemon | Host daemon is required for the mesh |
| `@mention` in the composer | `crushbot mention` / TUI; Crush composer is unmodified |
| Kanban UI | File task queue + MCP `assign_task` |
| `Group: <name>` sessions | `bot.yaml.group_sessions` UUIDs, flag default off |

Do not expect Hermes Desktop relay, pixel pets, or in-Crush `@` middleware in v1.
