# kanren Core — Validation Report

**Verdict: PASS**
**Date**: 2026-07-12
**Diff range**: `c266b4a..HEAD` (scaffold through T10)
**Method**: Standalone fresh-eyes pass (spec-anchored coverage + discrimination sensor). Run inline rather than via sub-agent per environment constraints.

---

## Gate results

- `go test -race ./...` → **49 passed**, 0 failed (4 packages)
- `golangci-lint run ./...` → **0 issues**
- `go vet ./...` → clean

## Spec-anchored coverage

Every P1 acceptance criterion maps to at least one test asserting the spec-defined outcome.

| Requirement | Test (file:line) | Outcome asserted | Covered |
| ----------- | ---------------- | ---------------- | ------- |
| CARD-01 (filename/slug) | `card_test.go` TestSlugify / TestFilename | `0012-fix-bug.md`, fallback `card` | ✅ |
| CARD-02 (round-trip) | `card_test.go` TestRoundTrip | `Parse(Marshal(c))==c` incl. body bytes | ✅ |
| CARD-03 (malformed named error) | `card_test.go` TestParseMalformed; `store_test.go` TestIndexSkipsMalformed | error prefixed `card:`; skipped w/ warning | ✅ |
| CARD-03 edge (dashes in body) | `card_test.go` TestParseOnlyFirstBlock | only first block parsed | ✅ |
| CARD-04 (misfiled surfaced) | `query_test.go` TestMisfiledSurfaced | misfiled card returned, not reassigned | ✅ |
| CLI-01 (add) | `add_test.go` TestAddAssignsIDAndLeftmostColumn; `main_test.go` TestAddThenLs | id 1/2, leftmost column | ✅ |
| CLI-02 (move body-stable) | `query_test.go` TestMoveChangesOnlyStatusAndOrder | body byte-identical after move | ✅ |
| CLI-03 (ls grouped) | `main_test.go` TestAddThenLs | `todo (1)` grouping | ✅ |
| CLI-04 (bad input no write) | `query_test.go` TestMoveInvalidStatusNoWrite; `main_test.go` TestMvBadInput | error, file unchanged, nonzero exit | ✅ |
| QRY-01 (AND filters) | `query_test.go` TestListAndFilter; `ls_test.go` TestLsStatusFilter | single matching card | ✅ |
| QRY-02 (--json) | `ls_test.go` TestLsJSON | valid parseable array | ✅ |
| QRY-03 (empty→[]) | `ls_test.go` TestLsJSONEmptyIsArray | `[]`, exit 0 | ✅ |
| QRY-04 (unknown filter empty) | `query_test.go` TestListNoMatch; `ls_test.go` TestLsUnknownFilterNoError | empty, not error | ✅ |
| WEB-01 (board/api) | `web_test.go` TestIndexServesShell / TestListColumns / TestListCardsJSON / TestStaticAssetsServed | shell + columns + cards + assets | ✅ |
| WEB-02 (move persists) | `web_test.go` TestMoveCardPersistsLikeCLI / TestMoveInvalidStatus / TestMoveBadID | 200 + status change; 4xx/400 errors | ✅ |
| WEB-03 (live reload) | `watch_test.go` TestLiveReloadOnFileChange / TestEventsStreamOpens | SSE `reload` on file change | ✅ |
| WEB-04 (file identity) | `web_test.go` TestMoveCardPersistsLikeCLI | on-disk file == CLI mv result, body intact | ✅ |
| INIT-01/02 | `store_test.go` TestInitCreatesBoard / TestInitRefusesExisting | scaffold; refuse existing | ✅ |
| CI-01..04 | `.github/workflows/ci.yml`, `release.yml` | build/test/lint gate + release matrix | ✅ (config) |

**Manual UAT (WEB-02/04):** live server — a board move via the API (what a drag fires) changed only `status`+`order` on disk, body untouched; identical to `kanren mv`. Screenshots captured (light + dark themes).

## Discrimination sensor

Behavior-level faults injected in a scratch state, reverted after each.

| Mutation | Expected killer | Result |
| -------- | --------------- | ------ |
| Bypass Move column validation | TestMoveInvalidStatusNoWrite | ✅ killed |
| Filter.matches always true | TestListAndFilter, TestListNoMatch | ✅ killed |
| Parse uses last `---` not first | TestParseOnlyFirstBlock, TestRoundTrip/dashes | ✅ killed |
| Web moveCard no-op success | TestMoveCardPersistsLikeCLI, TestMoveInvalidStatus | ✅ killed |

**4/4 mutants killed, 0 survived.** The suite detects real behavioral regressions.

## Gaps / follow-ups (non-blocking)

- Browser drag gesture itself is covered by manual UAT, not automated e2e (no headless browser in MVP scope) — the server-side move it triggers is fully tested.
- `Save` writes to `Filename(id,title)`; renaming a card's title via hand-edit could orphan the old file. Out of scope for MVP (Move/Add keep title stable). Track if `edit`-driven title changes become common.
- Name `kanren` collides with miniKanren — product decision still open (spec Assumptions).
