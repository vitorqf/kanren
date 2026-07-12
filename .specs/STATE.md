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

- **Phase**: Tasks (design approved, AD-001/002/003 logged).
- **Done**: Spec, design, CI/CD scaffold (green), git initialized.
- **Next**: Approve tasks.md → Execute.
- **User context**: New to Go — wants teaching sessions during implementation. Web UI must use `impeccable` skill.
