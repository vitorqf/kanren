# kanren — Project State

## Decisions

### AD-001 — All file I/O lives in `internal/store`; `internal/card` stays pure
- **Status**: active
- **Date**: 2026-07-12
- **Decision**: No package outside `internal/store` performs filesystem reads/writes. `internal/card` is a pure codec (bytes ↔ struct, no I/O). CLI (`cmd/kanren`) and web (`internal/web`) are adapters that call `store` methods only.
- **Rationale**: Single I/O owner = the structural guarantee that CLI and web board never diverge (spec WEB-04). Pure `card` = trivially testable round-trip invariant.
- **Applies to**: all future features touching cards/config.

### AD-002 — goccy/go-yaml for frontmatter serialization
- **Status**: active
- **Date**: 2026-07-12
- **Decision**: Use `github.com/goccy/go-yaml` (not archived `gopkg.in/yaml.v3`). Struct field order preserved on Marshal.
- **Rationale**: Deterministic output = clean, minimal git diffs on card edits — the core selling point. yaml.v3 is archived.

### AD-003 — No-build web frontend
- **Status**: active
- **Date**: 2026-07-12
- **Decision**: Web UI is vanilla HTML/CSS/JS + vendored SortableJS, embedded via `embed.FS`. No npm, no bundler, no framework build step.
- **Rationale**: Single-binary distribution; a reputation OSS tool must clone-and-`go build` with zero JS toolchain. Web UI polish handled via the `impeccable` skill at implementation time.

---

## Handoff

- **Phase**: Execute — Phase 1 (card) + Phase 2 (store) complete.
- **Done**: T1 codec, T2 slug/filename, T3 init/open/config, T4 add/get/save+dup-id, T5 move/list/misfiled. 30 tests green, golangci-lint clean (installed at `$GOPATH/bin`).
- **Next**: Phase 3 CLI (T6 router, T7 query+json), then Phase 4 web (T8 server, T9 drag+impeccable, T10 live reload). After T10 → Verifier.
- **Gotcha learned**: `max`/`min` are Go 1.21+ builtins; revive `redefines-builtin-id` flags shadowing them. Avoid naming vars `max`/`min`.
- **User context**: New to Go — inline execution with per-concept teaching. Web UI (T9) must use `impeccable` skill.
