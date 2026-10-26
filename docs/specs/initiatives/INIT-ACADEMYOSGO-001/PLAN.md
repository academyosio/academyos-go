# AcademyOS Go SDK and CLI: Plan

## Approach

Build the SDK and CLI against an `httptest` server first, so the client is finished and tested before the server endpoints exist, then switch the AcademyOS team's own courses to it. The module stays unpublished until those courses go through it to a local server and to app.academyos.io.

## Steps

1. **Module and client.** Options, the base URL default and override, the HTTPS rule, typed errors, and `Me`.
2. **Course packages.** `coursepkg.Build` over a course-spec directory, with the package root for shared themes and the size limits.
3. **Push.** Local validation, the missing-files rounds, dry runs, prune, conflicts, and writing the academy's IDs to `published.json`.
4. **CLI.** `whoami`, `course validate`, `course push`, and `course publish`, with text and JSON output and the exit codes.
5. **Round trip.** `pull` and `status`.
6. **Prove it.** Push the ProductBuildersHQ courses to a local AcademyOS, then production, by hand and through an agent.
7. **Docs and release.** README with an agent guide, make the repository public, tag v0.1.0.

## Dependent Work in Other Projects

| Project | Change |
| --- | --- |
| AcademyOS | `GET /api/v1/me`; the package and file endpoints with revisions and conflicts; applying a course-spec package with the same validation; course IDs written back in responses. Replaces the current definition endpoint and the per-deck storage in the deck playback spec. Its `course` commands move out of `academyos-server` once this CLI can push |
| course-spec | Write helpers for `published.json` (RMI-COURSESPEC-013), so this module and recording tools update it the same way |
| ProductBuildersHQ courses | Port to course-spec (RMI-COURSESPEC-006) before they can be pushed with this CLI |

## Decisions

- **Public module, private until proven.** The repository's history is written as public from the first commit, and it is made public once the success criteria are met.
- **The academy comes from the key.** Keys belong to one academy, so commands don't need `--org`.
- **Files, not one document.** Uploading content-addressed files lets decks and their assets travel with the course, and repeats upload nothing.
- **The key only from the environment.** Never a flag, so it doesn't land in shell history or process lists.

## Open Questions

1. **Package root for a shared theme.** Uploading the shared theme with each course is simple; a theme shared by several courses could later be uploaded once per academy.
2. **Device sign-in.** Worth adding when authors other than us push courses.
