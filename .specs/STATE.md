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

- **Phase**: Execute COMPLETE — all 10 tasks done, feature validated (PASS).
- **Done**: T1–T10. 49 tests green, golangci-lint clean, validation.md written (4/4 mutants killed). Core MVP shippable: `kanren init/add/ls/mv/edit/serve` + local drag-drop board (light/dark) + live reload.
- **Next (optional follow-ups)**: resolve name collision (kanren↔miniKanren); README; consider title-rename orphan handling on `edit`; automated browser e2e; card body editing in web UI; `// TODO:` code scanning (P3+ backlog).
- **Gotchas learned**: `max`/`min` are Go 1.21+ builtins (revive `redefines-builtin-id`); errcheck flags `fmt.Fprint*`/`w.Write`/`resp.Body.Close` (excluded `fmt.Fprint*` in config, wrap the rest with `_ =`).
- **Tooling**: golangci-lint at `$(go env GOPATH)/bin/golangci-lint` (v2). CI runs it on push.
- **User context**: New to Go — inline execution with per-concept teaching.
