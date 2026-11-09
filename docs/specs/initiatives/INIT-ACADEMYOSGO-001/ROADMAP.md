# ROADMAP — AcademyOS Go SDK and CLI

**Initiative:** `INIT-ACADEMYOSGO-001`
**Repository:** `github.com/academyosio/academyos-go`

Specs: [PRD](PRD.md), [TRD](TRD.md), [PLAN](PLAN.md). Work items carry
stable IDs (`RMI-ACADEMYOSGO-NNN`), are reviewed and executed by phase, and
commits implementing one carry the trailer `Refs: RMI-ACADEMYOSGO-<NNN>`.

## Phase 1 — Push a Course

**Theme:** A course-spec directory goes to an academy in one command.

- [x] `RMI-ACADEMYOSGO-001` Client options, the production default and URL override, the HTTPS rule, typed errors, and `Me`
- [x] `RMI-ACADEMYOSGO-002` Course packages: the files a course-spec directory needs, checksums, the package root for shared themes, and size limits
- [x] `RMI-ACADEMYOSGO-003` Push: local validation, uploading only missing files, dry runs, prune, stale-copy refusal, and the academy's IDs written to `published.json`
  - Depends on: `RMI-ACADEMYOSGO-001`, `RMI-ACADEMYOSGO-002`
- [x] `RMI-ACADEMYOSGO-004` `academyos` CLI: `whoami`, `course validate`, `course push`, `course publish`, text and JSON output, and exit codes
  - Depends on: `RMI-ACADEMYOSGO-003`
- [x] `RMI-ACADEMYOSGO-005` Dependency test: no AcademyOS server packages in the module's dependencies
  - Depends on: `RMI-ACADEMYOSGO-001`

## Phase 2 — Round Trip and Release

**Theme:** Changes flow both ways, and our own courses prove it before release.

- [ ] `RMI-ACADEMYOSGO-006` Pull with protection for local edits
  - Depends on: `RMI-ACADEMYOSGO-003`
- [ ] `RMI-ACADEMYOSGO-007` Status: local and academy changes, with course-spec lesson status
  - Depends on: `RMI-ACADEMYOSGO-003`
- [ ] `RMI-ACADEMYOSGO-008` Push the ProductBuildersHQ courses to a local academy and to production, by hand and through an agent
  - Depends on: `RMI-ACADEMYOSGO-004`, `RMI-ACADEMYOSGO-006`
- [ ] `RMI-ACADEMYOSGO-009` README with an agent guide and examples
  - Depends on: `RMI-ACADEMYOSGO-004`
- [ ] `RMI-ACADEMYOSGO-010` Public repository and v0.1.0 release
  - Depends on: `RMI-ACADEMYOSGO-008`, `RMI-ACADEMYOSGO-009`

## Phase 3 — More Ways In

**Theme:** Other tools and authors reach the same SDK.

- [ ] `RMI-ACADEMYOSGO-011` MCP server over the SDK
- [ ] `RMI-ACADEMYOSGO-012` Device sign-in, so authors don't handle keys
- [ ] `RMI-ACADEMYOSGO-013` Narration review from the command line: list, approve, and request changes per slide
- [ ] `RMI-ACADEMYOSGO-014` Academy pronunciations: push a repository's root `pronunciations.json` as the academy's (`RMI-ACADEMYOS-070`), showing the terms added, changed and removed, with a dry run, and print the academy's
