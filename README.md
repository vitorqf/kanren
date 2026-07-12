# kanren

Plain-text, git-backed kanban. Every card is a markdown file. A local board and
a CLI edit the **same files**, so you own your tasks as plain text next to your
code, with git as history and sync. No server, no database, no account.

## Quick start

One command. It creates a board if there isn't one and opens it in your browser.

```sh
kanren serve
```

Then go to <http://localhost:7777>, type in a column to add cards, and drag them
between columns. That's it. Everything you do is saved as plain `.md` files in a
`cards/` folder you can commit to git.

Prefer the terminal? The CLI does the same things:

```sh
kanren add "fix auth token expiry"     # new card in the first column
kanren ls                              # show the board
kanren mv 1 doing                      # move card #1
kanren ls --status doing --tag urgent  # query like a database
```

## Install

With Go installed:

```sh
go install github.com/vitor/kanren/cmd/kanren@latest
```

Or download a prebuilt binary from the [releases page](../../releases) and put it
on your `PATH`.

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
assignee: vitor
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

| Command | What it does |
| --- | --- |
| `kanren serve [--port N]` | Open the local drag-drop board (auto-creates a board if none) |
| `kanren init` | Create a board in the current directory |
| `kanren add "<title>"` | Add a card to the first column |
| `kanren ls` | List cards grouped by column |
| `kanren ls --status <col> --tag <t> --assignee <who>` | Filtered list (all filters combine) |
| `kanren ls --json` | Machine-readable output for scripts |
| `kanren mv <id> <status>` | Move a card to another column |
| `kanren edit <id>` | Open a card in `$EDITOR` |

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

## License

MIT.
