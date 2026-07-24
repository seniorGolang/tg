---
name: tg-plugin
description: >-
  Authors and rebuilds tg WASM plugins (.tgp): non-interactive init/add/build,
  monorepo local file:// reinstall with :packageName, Info().Name vs dirname,
  packaging skills/. Use only when developing the plugin toolchain itself —
  not when using tgp-go generators or writing @tg contracts.
disable-model-invocation: true
---

# tg-plugin

Manual (`/tg-plugin`) for plugin authors. Run all commands from the **plugin repo root** (`plugins/`, `core/`, `go.mod`). `tg` uses `cwd` as RootDir — it does not walk up to find the repo.

## Layout

```text
plugins/<dir>/
  plugin.go      # Info().Name, Execute, Kind, ACL
  plugin.md      # tg plugin doc (embedded)
  skills/<skill>/SKILL.md   # optional; packed as <plugin>-skills.tar.gz (not inside .tgp)
```

Publish name of a skill = **directory name**, not frontmatter `name`. Keep them identical (kebab-case).

## Recipe A — new single-plugin repo

```bash
cd /path/to/new-repo
tg plugin init -n demo -c demo --deploy-type=none
# edit plugins/demo/
tg plugin build --clean
tg pkg add "file://$(pwd)/dist:demo" --force
tg plugin doc demo
```

`-n` required; defaults: module `tgp`, license `MIT`, deploy `none`. Prefer `--fail-on-missing` if prompts must not appear.

## Recipe B — add plugin to monorepo

```bash
cd /path/to/monorepo   # core/ must already exist
tg plugin add -n other -c other
tg plugin build --clean
tg pkg add "file://$(pwd)/dist:<InfoName>" --force
```

## Recipe C — rebuild / reinstall (dev loop)

```bash
tg plugin build                  # always builds ALL dirs under plugins/
tg pkg add "file://$(pwd)/dist:<InfoName>" --force
tg pkg skills install <package>  # if skills/ changed
```

**Why `--force`:** `file://` has no versions; manifest version often stays the same (git tag or `0.0.1`). Without `--force`, install may report **unchanged** (checksum match) and leave the old binary.

**Why `:InfoName`:** if `dist/manifest.yml` has **more than one** package and you pass only `file://…/dist`, CLI opens an **interactive** multiselect (hangs without TTY). Pin the package:

```bash
tg pkg add "file:///abs/path/to/dist:myDemo" --force
```

`InfoName` = `Info().Name` from WASM (often camelCase from the template), **not** the folder name under `plugins/`. Check the built `dist/manifest.yml` if unsure.

There is **no** CLI flag to build a single plugin; workaround is not supported — always build all, install one by name.

## Recipe D — refresh generated scaffold / CI

```bash
tg plugin update    # regenerates templates in the plugin repo
```

Not the same as `tg pkg update` (refreshes catalog manifests) or `tg pkg upgrade` (**not implemented**). Local reinstall = Recipe C, never `pkg upgrade`.

## Skills in a plugin

- Put domain rules and anti-patterns in `SKILL.md`, not CLI help dumps.
- Rare authoring-only skills → `disable-model-invocation: true`.
- After package install: `tg pkg skills install` (host `tg skills install` does not publish package skills).

## Never

- `tg pkg add file://dist --force` in a multi-plugin repo without `:package` → interactive select
- Expect `plugin build` to compile only one plugin
- Use dirname instead of `Info().Name` in the URI
- Use `tg pkg upgrade` for local rebuilds
- Hand-edit generated scaffold; use `tg plugin update` then rebuild
- Treat this skill as tgp domain work → package skills / skill `tg` for host ops
