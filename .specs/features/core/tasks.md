# kanren Core — Tasks

## Execution Protocol (MANDATORY -- do not skip)

Implement these tasks with the `tlc-spec-driven` skill: **activate it by name and follow its Execute flow and Critical Rules.** Do not search for skill files by filesystem path. The skill is the source of truth for the full flow (per-task cycle, sub-agent delegation, adequacy review, Verifier, discrimination sensor).

**If the skill cannot be activated, STOP and tell the user — do not proceed without it.**

---

**Design**: `.specs/features/core/design.md`
**Status**: Draft

---

## Test Coverage Matrix

> Generated from codebase, project guidelines, and spec — confirm before Execute. Guidelines found: none (CLAUDE.md is global tooling, no project testing rules) — strong defaults applied. Scaffold sample: `internal/card/card_test.go` (Go stdlib `testing`, co-located `_test.go`).

| Code Layer | Required Test Type | Coverage Expectation | Location Pattern | Run Command |
| ---------- | ------------------ | -------------------- | ---------------- | ----------- |
| `internal/card` (pure codec) | unit | All branches; 1:1 to CARD-* ACs; every listed edge case (malformed, `---` in body, round-trip) | `internal/card/*_test.go` | `go test -race ./internal/card/` |
| `internal/store` (I/O + logic) | unit | All branches; 1:1 to CLI-*/QRY-*/CARD-04/INIT-* ACs; dup id, misfiled, empty dir edges | `internal/store/*_test.go` | `go test -race ./internal/store/` |
| `internal/web` (HTTP handlers) | integration | Every endpoint: happy + error path via `net/http/httptest` (WEB-01/02/03/04) | `internal/web/*_test.go` | `go test -race ./internal/web/` |
| `cmd/kanren` (CLI router) | unit | Arg parsing + exit codes for happy + bad-input (CLI-04) | `cmd/kanren/*_test.go` | `go test -race ./cmd/kanren/` |
| CI/config/embedded assets | none | build gate only | — | `make ci` |

Browser drag-drop behavior (SortableJS) = manual UAT with `impeccable`, not automated e2e (no headless browser in MVP scope).

## Parallelism Assessment

> Confirm before Execute.

| Test Type | Parallel-Safe? | Isolation Model | Evidence |
| --------- | -------------- | --------------- | -------- |
| card unit | Yes | Pure funcs, no shared state, in-memory bytes | `card` has zero I/O (AD-001) |
| store unit | Yes | Each test uses `t.TempDir()` (per-test isolated dir) | Go idiom; no shared global store |
| web integration | Yes | `httptest.Server` per test over a `t.TempDir()` store | No shared port/state |
| cmd unit | Yes | Per-test `t.TempDir()` + captured stdout | No shared state |

All layers parallel-safe via `t.TempDir()` isolation → `[P]` allowed where code deps permit.

## Gate Check Commands

> Confirm before Execute.

| Gate Level | When to Use | Command |
| ---------- | ----------- | ------- |
| Quick | After a task with unit tests in one package | `go test -race ./internal/<pkg>/` |
| Full | After web/integration tasks | `go test -race ./...` |
| Build | Phase completion / config-only tasks | `make ci` (build + test + vet + lint) |

---

## Execution Plan

### Phase 1: Card codec (Foundation, Sequential)
```
T1 → T2
```

### Phase 2: Store (Sequential — shared package, incremental)
```
T2 → T3 → T4 → T5
```

### Phase 3: CLI (depends on full store)
```
T5 → T6 → T7
```

### Phase 4: Web board (depends on full store)
```
T5 → T8 → T9 → T10
```

Phases 3 and 4 both depend only on T5 → could interleave, but kept sequential (single-dev, avoids context thrash). 4 phases → sub-agent offer applies at Execute.

---

## Task Breakdown

### T1: Card parse/serialize round-trip codec
**What**: Implement `Parse([]byte) (Card, error)` and `(Card) Marshal() ([]byte, error)` — split first `---`…`---` frontmatter, yaml via goccy, body = remainder.
**Where**: `internal/card/card.go` (extend), `internal/card/card_test.go`
**Depends on**: None
**Reuses**: existing `Card` struct + yaml tags
**Requirement**: CARD-01, CARD-02, CARD-03
**Tools**: MCP: `context7` (goccy API); Skill: NONE
**Done when**:
- [ ] `Parse(Marshal(c)) == c` for valid cards (round-trip, table-driven)
- [ ] Malformed frontmatter → error naming the reason, no panic (CARD-03)
- [ ] Body containing `---` lines: only first block parsed as frontmatter (edge case)
- [ ] `go mod tidy` adds goccy/go-yaml
- [ ] Quick gate passes; test count ≥ 6
**Tests**: unit
**Gate**: quick
**Commit**: `feat(card): frontmatter parse/serialize round-trip codec`

### T2: Slugify + filename helper
**What**: `Slugify(title) string` (lowercase, spaces→`-`, strip unsafe) + `Filename(id, title) string` → `NNNN-slug.md`.
**Where**: `internal/card/card.go`, `internal/card/card_test.go`
**Depends on**: T1
**Requirement**: CARD-01
**Tools**: MCP: NONE; Skill: NONE
**Done when**:
- [ ] Slug is filename-safe (unicode, punctuation, empty title → fallback)
- [ ] `Filename(12,"Fix bug")` == `0012-fix-bug.md`
- [ ] Quick gate passes; test count ≥ 4
**Tests**: unit
**Gate**: quick
**Commit**: `feat(card): slugify and card filename helpers`

### T3: Store open/init + config
**What**: `Init(dir)` scaffolds `.kanren.yml`+`cards/`; `Open(dir)` loads config+indexes cards, errors if config missing; `Columns()`.
**Where**: `internal/store/store.go`, `internal/store/store_test.go`
**Depends on**: T2
**Requirement**: INIT-01, INIT-02, CARD-03 (skip-malformed-with-report)
**Tools**: MCP: NONE; Skill: NONE
**Done when**:
- [ ] `Init` on empty dir writes config + `cards/`; refuses if config exists (INIT-02)
- [ ] `Open` missing config → error telling user to run `init`
- [ ] Malformed card file → collected error, other cards still indexed (CARD-03)
- [ ] Non-`.md` files ignored; empty dir OK (edges)
- [ ] Quick gate `./internal/store/`; test count ≥ 6
**Tests**: unit
**Gate**: quick
**Commit**: `feat(store): board init, open, and config loading`

### T4: Store Add + Get + Save
**What**: `Add(title)` → next id, leftmost column, `created`, write file; `Get(id)`; `Save(Card)`. Duplicate-id detection.
**Where**: `internal/store/store.go`, test
**Depends on**: T3
**Requirement**: CLI-01, edge: duplicate id
**Tools**: MCP: NONE; Skill: NONE
**Done when**:
- [ ] `Add` assigns unique incrementing id, leftmost column, returns id+path (CLI-01)
- [ ] Duplicate id on disk → `Get`/id-ops refuse with both filenames (edge)
- [ ] Written file round-trips through `card.Parse` identically
- [ ] Quick gate; test count ≥ 5
**Tests**: unit
**Gate**: quick
**Commit**: `feat(store): add, get, save cards with id allocation`

### T5: Store Move + List(filter)
**What**: `Move(id,status)` edits only `status`/`order`, body byte-identical, validates status∈columns else misfiled; `List(Filter)` AND-filter status/tag/assignee.
**Where**: `internal/store/store.go`, test
**Depends on**: T4
**Requirement**: CLI-02, CLI-03, CLI-04, QRY-01, QRY-03, QRY-04, CARD-04
**Tools**: MCP: NONE; Skill: NONE
**Done when**:
- [ ] `Move` changes only status/order — golden assert body bytes unchanged (CLI-02)
- [ ] Bad id/status → error, zero file writes (CLI-04)
- [ ] status∉columns flagged misfiled, not silently moved (CARD-04)
- [ ] `List` AND-combines filters; no match → empty, no error (QRY-01/03/04)
- [ ] Quick gate; test count ≥ 8
**Tests**: unit
**Gate**: quick
**Commit**: `feat(store): move cards and DB-style filtered list`

### T6: CLI router + init/add/ls/mv/edit
**What**: Replace stub `main.go`; subcommand dispatch → store; `ls` grouped by column; `edit` opens `$EDITOR`.
**Where**: `cmd/kanren/main.go`, `cmd/kanren/*.go`, `cmd/kanren/main_test.go`
**Depends on**: T5
**Requirement**: CLI-01/02/03/04, INIT-01, EDIT-01
**Tools**: MCP: NONE; Skill: NONE
**Done when**:
- [ ] `init/add/ls/mv/edit` dispatch to store; unknown cmd → nonzero + usage
- [ ] `mv` bad input → nonzero exit, clear message (CLI-04)
- [ ] `ls` groups by column in board order (CLI-03)
- [ ] Quick gate `./cmd/kanren/`; test count ≥ 5
**Tests**: unit
**Gate**: quick
**Commit**: `feat(cli): init, add, ls, mv, edit commands`

### T7: CLI query flags + `--json`
**What**: `ls --status/--tag/--assignee/--json` wired to `store.List` + JSON encoder.
**Where**: `cmd/kanren/ls.go`, test
**Depends on**: T6
**Requirement**: QRY-01, QRY-02, QRY-03, QRY-04
**Tools**: MCP: NONE; Skill: NONE
**Done when**:
- [ ] Combined filters = AND (QRY-01)
- [ ] `--json` emits valid parseable JSON array; empty match → `[]` exit 0 (QRY-02/03)
- [ ] Unknown tag/status → empty, not error (QRY-04)
- [ ] Quick gate; test count ≥ 5
**Tests**: unit
**Gate**: quick
**Commit**: `feat(cli): query filters and --json output`

### T8: Web server + board render + JSON API
**What**: `Serve(store,port)`; `GET /` board HTML (columns+cards, embed.FS assets), `GET /api/cards`, `POST /api/cards/{id}/move`.
**Where**: `internal/web/web.go`, `internal/web/assets/*`, `internal/web/web_test.go`
**Depends on**: T5
**Requirement**: WEB-01, WEB-02, WEB-04
**Tools**: MCP: `context7` (net/http, embed); Skill: `impeccable` (UI design)
**Done when**:
- [ ] `GET /` renders columns from `store.Columns()` + cards (WEB-01)
- [ ] `POST /move` calls `store.Move`; resulting file identical to CLI mv (WEB-02/04) — assert via store round-trip
- [ ] Port taken → clear error (edge)
- [ ] Integration gate via `httptest`; test count ≥ 5
**Tests**: integration
**Gate**: full
**Commit**: `feat(web): board server, render, and JSON API`

### T9: Drag-drop UI (SortableJS) + impeccable polish
**What**: Vendored SortableJS; drag card→column POSTs move; apply `impeccable` for layout/type/spacing/color/dark-mode/empty-states.
**Where**: `internal/web/assets/{index.html,app.js,style.css,sortable.min.js}`
**Depends on**: T8
**Requirement**: WEB-02
**Tools**: MCP: NONE; Skill: `impeccable` (MANDATORY per user)
**Done when**:
- [ ] Drag card between columns fires `POST /move`, persists file
- [ ] `impeccable` pass applied: visual hierarchy, spacing scale, light+dark theme, empty-column state
- [ ] Assets embedded via `embed.FS`; no external network requests
- [ ] Manual UAT: drag = same `git diff` a `kanren mv` produces
**Tests**: none (browser behavior = manual UAT; server side covered T8)
**Gate**: build
**Commit**: `feat(web): drag-drop board UI with impeccable polish`

### T10: Live reload (fsnotify + SSE)
**What**: Watch `cards/` via fsnotify; `GET /events` SSE; browser reloads board on file change (CLI edit / git pull).
**Where**: `internal/web/watch.go`, `internal/web/assets/app.js`, test
**Depends on**: T9
**Requirement**: WEB-03
**Tools**: MCP: `context7` (fsnotify); Skill: NONE
**Done when**:
- [ ] External file change → SSE event → board updates without restart (WEB-03)
- [ ] Watcher unavailable → 2s poll fallback (design mitigation)
- [ ] Integration test: write file, assert SSE emits; test count ≥ 2
- [ ] Full gate `go test -race ./...`
**Tests**: integration
**Gate**: full
**Commit**: `feat(web): live board reload via fsnotify and SSE`

---

## Task Granularity Check

| Task | Scope | Status |
| ---- | ----- | ------ |
| T1 codec | 2 funcs, 1 file | ✅ |
| T2 slug/filename | 2 funcs | ✅ |
| T3 init/open/config | 1 concern (lifecycle) | ✅ |
| T4 add/get/save | 1 concern (create) | ✅ |
| T5 move/list | 1 concern (mutate+query) | ✅ |
| T6 CLI router | router + dispatch, cohesive | ⚠️ OK (cohesive, one file) |
| T7 query flags | flag wiring | ✅ |
| T8 web server+api | server+handlers, cohesive | ⚠️ OK |
| T9 drag UI | frontend assets | ✅ |
| T10 live reload | 1 feature | ✅ |

## Diagram-Definition Cross-Check

| Task | Depends On (body) | Diagram | Status |
| ---- | ----------------- | ------- | ------ |
| T1 | None | Phase1 start | ✅ |
| T2 | T1 | T1→T2 | ✅ |
| T3 | T2 | T2→T3 | ✅ |
| T4 | T3 | T3→T4 | ✅ |
| T5 | T4 | T4→T5 | ✅ |
| T6 | T5 | T5→T6 | ✅ |
| T7 | T6 | T6→T7 | ✅ |
| T8 | T5 | T5→T8 | ✅ |
| T9 | T8 | T8→T9 | ✅ |
| T10 | T9 | T9→T10 | ✅ |

## Test Co-location Validation

| Task | Layer | Matrix Requires | Task Says | Status |
| ---- | ----- | --------------- | --------- | ------ |
| T1 | card | unit | unit | ✅ |
| T2 | card | unit | unit | ✅ |
| T3 | store | unit | unit | ✅ |
| T4 | store | unit | unit | ✅ |
| T5 | store | unit | unit | ✅ |
| T6 | cmd | unit | unit | ✅ |
| T7 | cmd | unit | unit | ✅ |
| T8 | web | integration | integration | ✅ |
| T9 | web assets (browser) | none (manual UAT) | none | ✅ |
| T10 | web | integration | integration | ✅ |

All checks pass.
