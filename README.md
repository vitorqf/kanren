# kanren

Plain-text, git-backed kanban. Every card is a markdown file. A local board and
a CLI edit the **same files**, so you own your tasks as plain text next to your
code, with git as history and sync. No server, no database, no account.

**[Quick start](#quick-start) · [Install](#install) · [A board is a folder](#a-board-is-a-folder) · [Card format](#what-a-card-looks-like) · [Commands](#commands) · [Why plain files](#why-plain-files) · [How it works](#how-it-works) · [Releasing](#releasing)**

## Quick start

[Install the binary](#install) once, then in any folder:

```sh
mkdir my-tasks && cd my-tasks
kanren serve
```

It creates the board and opens it at <http://localhost:7777>. Type in a column to
add cards, drag them between columns, click a card to edit it. Everything is
saved as plain `.md` files in `my-tasks/cards/`.

Prefer the terminal? The CLI does the same things:

```sh
kanren add "fix auth token expiry"     # new card in the first column
kanren ls                              # show the board
kanren mv 1 doing                      # move card #1
kanren ls --status doing --tag urgent  # query like a database
```

## Install

`kanren` is a single binary, like `git`. Install it once and it works from any
folder.

**Download a prebuilt binary** from the [latest release](../../releases/latest).
Pick the file for your system, make it runnable, and move it onto your `PATH`:

```sh
# example: Apple Silicon Mac (darwin-arm64) — swap for your OS/arch
curl -L -o kanren https://github.com/vitorqf/kanren/releases/latest/download/kanren-v0.1.0-darwin-arm64
chmod +x kanren
sudo mv kanren /usr/local/bin/
```

Files are named `kanren-<version>-<os>-<arch>`: `darwin-arm64` (Apple Silicon),
`darwin-amd64` (Intel Mac), `linux-amd64`, `linux-arm64`,
`windows-amd64.exe`, `windows-arm64.exe`.

**Or, with Go installed:**

```sh
go install github.com/vitorqf/kanren/cmd/kanren@latest
```

## A board is a folder

There is no central app or account. **Any folder you run `kanren` in is a
board** — the first `kanren serve` (or `kanren init`) drops a `.kanren.yml` and a
`cards/` folder right there. Two common setups:

**A standalone board** — a folder just for tasks:

```sh
mkdir ~/todo && cd ~/todo && kanren serve
```

**Inside a project** — tasks live next to the code and travel with the repo:

```sh
cd ~/code/my-app
kanren serve                              # creates ./cards/ in the repo
git add .kanren.yml cards/ && git commit -m "add board"
```

Now anyone who `git pull`s gets the cards, and moving a card shows up as a
reviewable diff. Whether the board is committed is your repo's choice; kanren
just writes files.

## What a card looks like

A card is just a markdown file with a small YAML header. Readable and editable
without kanren:

```markdown
---
id: 1
title: fix auth token expiry
status: doing
tags:
  - bug
  - urgent
assignee: vitorqf
created: 2026-07-12
---

# fix auth token expiry

Check uses `<` not `<=`.
```

`status` is the column. The columns live in `.kanren.yml`:

```yaml
columns:
  - todo
  - doing
  - done
cards_dir: cards
```

## Commands

| Command                                               | What it does                                                  |
| ----------------------------------------------------- | ------------------------------------------------------------- |
| `kanren serve [--port N]`                             | Open the local drag-drop board (auto-creates a board if none) |
| `kanren init`                                         | Create a board in the current directory                       |
| `kanren add "<title>"`                                | Add a card to the first column                                |
| `kanren ls`                                           | List cards grouped by column                                  |
| `kanren ls --status <col> --tag <t> --assignee <who>` | Filtered list (all filters combine)                           |
| `kanren ls --json`                                    | Machine-readable output for scripts                           |
| `kanren mv <id> <status>`                             | Move a card to another column                                 |
| `kanren edit <id>`                                    | Open a card in `$EDITOR`                                      |

## Why plain files

- **You own the data.** Grep it, diff it, edit it in any editor. No lock-in.
- **Git is the sync.** Branch your board, review card changes in a PR, `git blame`
  who moved what. Merges are real merges.
- **Offline by default.** It's just files on disk.
- **One binary.** The board UI ships inside the binary; no external requests.

## How it works

The CLI and the web board are thin adapters over one package that owns all file
access, so they can never disagree. Moving a card rewrites only its `status`
line, keeping git diffs minimal. The board watches the `cards/` folder and
refreshes live when files change, so a CLI edit or a `git pull` shows up without
a reload.

## Releasing

Releases are cut by pushing a git tag. The `Release` workflow then
cross-compiles binaries for linux, macOS, and Windows (amd64 + arm64) and
attaches them to a GitHub Release.

```sh
git tag v0.1.0        # use the next semver, prefixed with v
git push origin v0.1.0
```

The version is baked into the binary from the tag, so `kanren version` reports
it. Check progress with `gh run watch` or the repo's Actions tab; the release
appears under **Releases** when the job finishes.

## License

MIT.
