---
name: tg
description: >-
  Diagnoses Tool Gateway host failures: missing plugins/commands, wrong scope,
  skills not published to ~/.agents (silent skip), host vs package skills.
  Use when tgp/plugin commands are missing, pkg add/list/scope, or skills absent
  under ~/.agents — not for @tg contracts, generators, or authoring .tgp plugins
  (use package skills or /tg-plugin).
---

# tg

Platform troubleshooting only. Prefer `tg --help` for flag lists.

## Diagnose missing plugins / commands

1. `tg pkg list` — is the package installed in the **current** scope?
2. `tg pkg scope list` / `tg pkg scope use <name>` — wrong scope → commands vanish.
3. Install if needed: `tg pkg add <source>` (e.g. GitHub URL of the package repo).
4. Runtime docs: `tg plugin doc <name>` — usage of an **installed** plugin, not agent workflow.

## Skills not appearing (main trap)

Default target is `agents` → `~/.agents/skills`. If `~/.agents` is missing and `--skills-mkdir` was not set, activation **succeeds but silently skips** the target.

```bash
tg pkg skills install                 # skills from installed packages (usual fix)
tg pkg skills install --skills-mkdir  # create ~/.agents if needed
tg skills install                     # built-in host skills only (tg, tg-plugin)
```

- `tg skills install` ≠ `tg pkg skills install`. Host skills do not deliver package skills (e.g. tgp-*).
- Optional: `--skills-targets=agents,cursor` (root must exist, or use `--skills-mkdir`).

## Scopes vs skills

- Scopes isolate **installed plugins** under `~/.tg/scopes/…`.
- Skills publish to **global** `~/.agents/skills` (and other targets).
- After `scope use`, CLI plugins change; leftover skills in `~/.agents` may still mention commands that are gone in the new scope.

## `plugin doc` ≠ agent skill

| Artifact | Source | Audience |
|---|---|---|
| `tg plugin doc` | WASM / `plugin.md` inside `.tgp` | runtime usage |
| Package `SKILL.md` | separate `*-skills.tar.gz` | agent domain rules |

## Never

- Author `@tg` contracts → package skill (e.g. `/tgp-contracts`)
- Generate server/clients/swagger/kafka → matching package skill
- Scaffold / build / local `file://` reinstall of `.tgp` → `/tg-plugin`
