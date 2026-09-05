# soul.md

Every bot has a required `soul.md` in its home directory. It is the source of truth for identity, tone, and disagreement. crushbot seeds it once on `spawn` and **never overwrites** an existing file.

## What belongs here

- Who the bot is
- How it writes
- What it refuses (sycophancy, hype, stretching into another bot’s job)

## What does not

Mesh protocol, teammate lists, tool allow-lists, and repo conventions live in generated `protocol.md` / `CRUSH.md`. Edit `soul.md`; do not paste protocol into it.

## How Crush sees it

Crush still loads its built-in coder prompt. crushbot overlays:

1. `option global-context-path` → `soul.md` then `protocol.md`
2. cwd `CRUSH.md` starting with a hard identity line
3. A PreToolUse hook that re-asserts identity before every tool call

After a coding turn, `crushbot say <slug> "who are you?"` should still answer as that bot.

## Editing

```bash
crushbot soul researcher           # print
crushbot soul researcher --edit    # $EDITOR; never clobbers on spawn
```

Injection-looking phrases (`ignore previous instructions`, …) produce a **warning** on spawn/save. They do not block.
