# Crush mesh model

crushbot is a Charm Crush powered bot mesh: named Crush workspaces, a host daemon, and MCP tools. Crush stays the agent runtime.

| Piece | crushbot |
| --- | --- |
| Identity | `soul.md` overlay + PreToolUse `{"context":…}` on Crush’s coder prompt |
| Inter-bot DM | Mailbox JSON + daemon `crush run` |
| Reply | FYI `kind: receipt` after a successful DM wake |
| Anti-loop | Hop 8 + fan-out (mailbox can ping-pong) |
| Courier | Host daemon is required for the mesh |
| `@mention` | `crushbot mention` / TUI; Crush composer is unmodified |
| Tasks | File task queue + MCP `assign_task` |
| Groups | `bot.yaml.group_sessions` UUIDs, flag default off |

v1 does not include a desktop relay, pixel pets, or in-Crush `@` middleware. Crush `/new` is a session inside one bot; it does not add a roster row.
