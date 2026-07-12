# kanren — Core Specification

> Plain-text, git-backed kanban. Per-card markdown files. CLI + local web board edit the same files. Query cards like a database. OSS, reputation-only (no revenue).

## Problem Statement

Devs distrust cloud kanban lock-in and want to own their task data as plain text alongside code. Existing OSS boards need a server/DB, hide data behind app-specific formats, or lack a real board UI (todo.txt, taskell) — none give a per-card markdown store that a CLI _and_ a visual board both drive over the same files, with git as the sync/history layer.

## Goals

- [ ] A board is just a folder of per-card `.md` files — readable, greppable, diffable without the app.
- [ ] CLI and local web board mutate the **same files**; no separate DB, no divergence.
- [ ] Cards queryable like a DB: filter by status/tag/assignee, machine-readable output.
- [ ] Green CI/CD pipeline exists and gates every commit **before** feature code lands.
- [ ] Single binary, no external services, offline by default.

## Out of Scope

| Feature                                          | Reason                                               |
| ------------------------------------------------ | ---------------------------------------------------- |
| Cloud sync / hosted service                      | Reputation-only, no revenue; git _is_ the sync layer |
| Auth / multi-user accounts                       | Single-user local tool; git handles collaboration    |
| Real-time multiplayer / websockets live-collab   | Overkill for MVP; git merge is the model             |
| Mobile app                                       | Desktop/terminal dev tool                            |
| Automatic conflict _resolution_                  | Git owns merges; app must not silently rewrite       |
| Card comments / attachments / due-date reminders | Post-MVP; keep card format minimal                   |
| `// TODO:` code scanning, PR/commit card links   | Great P3+ ideas, not core loop                       |

---

## Assumptions & Open Questions

| Assumption / decision              | Chosen default                                      | Rationale                                                         | Confirmed? |
| ---------------------------------- | --------------------------------------------------- | ----------------------------------------------------------------- | ---------- |
| Language/stack                     | Go single binary, embedded web via `embed.FS`       | One binary for CLI+web, easy cross-compile, mature YAML/HTTP libs | n          |
| Status stored where                | YAML frontmatter `status:` field                    | Cleaner diffs, easier query, move = 1 field edit (user-picked)    | y          |
| Board UI form                      | Local web served on `localhost`, drag-drop          | User-picked; demos/screenshots well for reputation                | y          |
| Card filename / ID                 | `NNNN-slug.md` (zero-padded incrementing id + slug) | Stable ID for references, human-readable, sortable                | n          |
| Concurrency between CLI + open web | Last-write-wins on file; web watches fs + reloads   | Single user, no locking needed; simple                            | n          |
| Ordering within a column           | `order:` float field in frontmatter                 | Reorder without rewriting neighbors; sparse indexing              | n          |
| Column set definition              | `.kanren.yml` at board root lists columns           | Explicit, versioned with board                                    | n          |
| Concurrent edit conflict           | App never auto-merges; git surfaces conflicts       | "Automatic resolution" is out of scope                            | y          |
| **Name `kanren`**                  | Working name; collides with miniKanren logic lib    | ⚠️ Reputation/SEO risk — revisit before public launch             | **n**      |

**Open questions:** Name collision (kanren ↔ miniKanren) unresolved — flagged for decision before public release. All others defaulted above.

---

## User Stories

### P1: CI/CD pipeline first ⭐ MVP

**User Story**: As the maintainer, I want CI running build + test + lint on every push/PR before real features exist, so quality gates are in place from commit one.

**Why P1**: User mandate — CI/CD before heavy code. Prevents retrofitting gates onto a grown codebase.

**Acceptance Criteria**:

1. WHEN a commit is pushed or a PR opened THEN CI SHALL run `build`, `test`, and `lint` jobs and report status.
2. WHEN any job fails THEN CI SHALL mark the check failed (red) and block a green badge.
3. WHEN the repo has only scaffolding (no features) THEN CI SHALL still pass green on a trivial passing test.
4. WHEN a tagged release commit lands THEN a release workflow SHALL cross-compile binaries (linux/mac/win, amd64/arm64) and attach them to a GitHub Release.

**Independent Test**: Push scaffold repo → GitHub Actions shows green build/test/lint. Push a deliberately failing test → check goes red.

---

### P1: Card as per-file markdown ⭐ MVP

**User Story**: As a dev, I want each card to be one `.md` file with YAML frontmatter, so I own greppable, diffable task data.

**Why P1**: The core data model everything else reads/writes.

**Acceptance Criteria**:

1. WHEN a card is created THEN system SHALL write one file `NNNN-slug.md` with frontmatter (`id`, `title`, `status`, `tags`, `assignee`, `created`, `order`) and a markdown body.
2. WHEN a file is hand-edited to valid frontmatter THEN system SHALL read it back identically (round-trip parse == serialize for valid input).
3. WHEN a card file has malformed/missing frontmatter THEN system SHALL report the offending file+reason and SHALL NOT crash or drop other cards.
4. WHEN `status` is not one of the columns in `.kanren.yml` THEN system SHALL flag the card as misfiled rather than silently reassigning it.

**Independent Test**: `kanren add "fix bug"` creates a file; `cat` it; edit body by hand; `kanren ls` reflects the edit.

---

### P1: CLI create / list / move ⭐ MVP

**User Story**: As a dev, I want `kanren add/ls/mv/edit` to manage cards from the terminal, so I never leave my workflow.

**Why P1**: One of the two co-equal editors of the file store.

**Acceptance Criteria**:

1. WHEN `kanren add "<title>"` runs THEN system SHALL create a card in the first (leftmost) column with a new unique id and print the id+path.
2. WHEN `kanren mv <id> <status>` runs THEN system SHALL update only that card's `status` (and `order`) field, leaving the body byte-identical.
3. WHEN `kanren ls` runs THEN system SHALL list cards grouped by column in board order.
4. WHEN `kanren mv` targets a nonexistent id or invalid status THEN system SHALL exit nonzero with a clear message and change no files.

**Independent Test**: Run add → mv → ls; verify status field changed and body untouched via `git diff`.

---

### P1: Query cards like a DB ⭐ MVP

**User Story**: As a dev, I want `kanren ls --status doing --tag urgent --assignee vitorqf`, so I can slice the board like a database.

**Why P1**: A headline differentiator; drives scripting/automation.

**Acceptance Criteria**:

1. WHEN filter flags (`--status`, `--tag`, `--assignee`) are combined THEN system SHALL return cards matching **all** given filters (AND semantics).
2. WHEN `--json` is passed THEN system SHALL emit machine-readable JSON of the matching cards.
3. WHEN no cards match THEN system SHALL exit 0 with empty output (empty JSON array under `--json`).
4. WHEN an unknown tag/status is filtered THEN system SHALL return empty, not error (filter, not validation).

**Independent Test**: Seed cards; `kanren ls --status doing --tag urgent --json | jq` returns exactly the expected set.

---

### P1: Local web board over the same files ⭐ MVP

**User Story**: As a dev, I want `kanren serve` to open a localhost drag-drop board that reads and writes the exact same card files, so CLI and UI never diverge.

**Why P1**: The user-picked visual surface; proves "one store, two editors".

**Acceptance Criteria**:

1. WHEN `kanren serve` runs THEN system SHALL serve a board UI on `localhost:<port>` rendering columns from `.kanren.yml` and cards from files.
2. WHEN a card is dragged to another column THEN system SHALL persist the new `status`/`order` to that card's file (same format the CLI writes).
3. WHEN a card file changes on disk (e.g. CLI edit, `git pull`) while the board is open THEN the board SHALL reflect it without a manual full restart.
4. WHEN a card is created/edited in the web UI THEN the resulting file SHALL be indistinguishable from a CLI-created file of the same content.

**Independent Test**: `kanren serve`; drag a card in browser; `git diff` shows only that file's `status` changed; then `kanren mv` another card and watch the board update.

---

### P2: Board init & config

**User Story**: As a dev, I want `kanren init` to scaffold `.kanren.yml` + an empty board, so setup is one command.

**Acceptance Criteria**:

1. WHEN `kanren init` runs in an empty dir THEN system SHALL write `.kanren.yml` with default columns (`todo`, `doing`, `done`) and a `cards/` folder.
2. WHEN `kanren init` runs where `.kanren.yml` exists THEN system SHALL refuse and change nothing.

**Independent Test**: `kanren init` then `kanren add` works immediately.

---

### P3: Editor & body workflow

**User Story**: As a dev, I want `kanren edit <id>` to open the card in `$EDITOR`, so I can write rich card bodies.

**Acceptance Criteria**:

1. WHEN `kanren edit <id>` runs THEN system SHALL open that file in `$EDITOR` and re-validate frontmatter on save.

---

## Edge Cases

- WHEN two cards claim the same `id` THEN system SHALL report the duplicate and refuse ambiguous `mv`/`edit` by id.
- WHEN `cards/` contains a non-`.md` file THEN system SHALL ignore it.
- WHEN the board folder is empty THEN `ls`/`serve` SHALL show empty columns, not error.
- WHEN a card body contains `---` lines THEN the parser SHALL only treat the first frontmatter block as frontmatter.
- WHEN `.kanren.yml` is missing THEN commands SHALL instruct the user to run `kanren init`.
- WHEN the web server port is taken THEN `serve` SHALL fail with a clear message (and optionally try next port — logged as assumption).

---

## Requirement Traceability

| Requirement ID | Story               | Phase  | Status  |
| -------------- | ------------------- | ------ | ------- |
| CI-01          | P1: CI/CD           | Design | Pending |
| CI-02          | P1: CI/CD           | Design | Pending |
| CI-03          | P1: CI/CD           | Design | Pending |
| CI-04          | P1: CI/CD (release) | Design | Pending |
| CARD-01        | P1: Card model      | Design | Pending |
| CARD-02        | P1: Card model      | Design | Pending |
| CARD-03        | P1: Card model      | Design | Pending |
| CARD-04        | P1: Card model      | Design | Pending |
| CLI-01         | P1: CLI             | Design | Pending |
| CLI-02         | P1: CLI             | Design | Pending |
| CLI-03         | P1: CLI             | Design | Pending |
| CLI-04         | P1: CLI             | Design | Pending |
| QRY-01         | P1: Query           | Design | Pending |
| QRY-02         | P1: Query           | Design | Pending |
| QRY-03         | P1: Query           | Design | Pending |
| QRY-04         | P1: Query           | Design | Pending |
| WEB-01         | P1: Web board       | Design | Pending |
| WEB-02         | P1: Web board       | Design | Pending |
| WEB-03         | P1: Web board       | Design | Pending |
| WEB-04         | P1: Web board       | Design | Pending |
| INIT-01        | P2: Init            | -      | Pending |
| INIT-02        | P2: Init            | -      | Pending |
| EDIT-01        | P3: Editor          | -      | Pending |

**ID format:** `[CATEGORY]-[NUMBER]`
**Status values:** Pending → In Design → In Tasks → Implementing → Verified
**Coverage:** 23 total, 0 mapped to tasks (Tasks phase pending) ⚠️

---

## Success Criteria

- [ ] Fresh clone → `make ci` green locally, GitHub Actions green on push.
- [ ] `kanren add`, then a hand-edit, then `kanren ls` all agree on the same file — zero divergence.
- [ ] `kanren serve` drag = same file diff a `kanren mv` would produce.
- [ ] `kanren ls --status doing --tag urgent --json` returns exactly the right cards.
- [ ] Every card file is valid, human-readable markdown openable without the app.
