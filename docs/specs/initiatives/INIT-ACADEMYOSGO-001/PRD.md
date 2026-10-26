# AcademyOS Go SDK and CLI: Product Requirements

## Summary

`academyos-go` is a public Go module with an SDK for the AcademyOS API and the `academyos` command built on it. An author, or an AI agent working for one, keeps a course as a [course-spec](https://github.com/academyosio/course-spec) directory in git and uses `academyos` to check it, upload it with its slide decks, translations, and quizzes, publish it, and see what differs from the academy. It talks to `https://app.academyos.io` unless told to use another server, such as one running locally.

## Problem

Course authoring through the API works today, but not as a tool others can install:

- The upload command lives in the AcademyOS server binary, so an author needs the server's code and its dependencies to push a course.
- Its HTTP client is internal to the server, so no other Go program, such as an MCP server or a CI job, can reuse it.
- The upload sends one JSON document with the outline and quizzes. Slide decks, translations, pronunciations, and the files decks load can't be sent at all.
- The course format it reads is AcademyOS's own. The open course-spec format replaces it.

## Goals

1. **Install with one command.** `go install github.com/academyosio/academyos-go/cmd/academyos@latest`, with no server code in the module.
2. **The whole course in one push.** Outline, decks, the files decks load, translations, pronunciations, and quizzes, uploaded together and applied in one step on the server.
3. **Check before upload.** The course is validated locally with course-spec, with the same rules the server applies, so nothing is uploaded from a broken directory.
4. **Safe to repeat and safe for agents.** Pushing an unchanged course changes nothing. A push from a stale copy is refused instead of overwriting edits made in the academy. Every command has JSON output and distinct exit codes.
5. **Production by default, local on request.** The default server is `https://app.academyos.io`; a flag or environment variable points at another, such as `http://localhost:8080`.
6. **One SDK for every adapter.** The CLI is a thin layer over the SDK, so an MCP server and other tools later share the same calls.

## Non-Goals

- Running or operating a server. Migrations, invites, and administration stay in `academyos-server`.
- Signing in through a browser. The first release uses API keys created in AcademyOS; a device sign-in flow can follow.
- An MCP server. It comes after the CLI has proven the SDK.
- Editing courses outside course-spec. The SDK reads and writes course-spec directories.

## Users

| User | Needs |
| --- | --- |
| **Course author** | Install, push, publish, and see what changed, without learning the API |
| **AI agent** | Exact commands, JSON output, actionable errors, and refusals instead of silent overwrites |
| **CI pipeline** | A non-interactive push on merge, configured by environment variables |
| **AcademyOS team** | Push our own academy's courses with the tool we give authors, to prove it before publishing the module |

## Requirements

### Configuration

- The server is `https://app.academyos.io` unless `--api-url` or `ACADEMYOS_API_URL` names another.
- The API key comes from `ACADEMYOS_API_KEY`, never from a command-line argument, so it stays out of shell history and process lists.
- Keys are sent only over HTTPS, except to `localhost` and loopback addresses for development.
- An API key belongs to one person in one academy, so the academy comes from the key. `--org` is optional and must match when given.

### Commands

- `academyos whoami` shows the key's person, academy, scopes, and expiry.
- `academyos course validate <dir>` checks a course with course-spec.
- `academyos course push <dir>` validates, uploads only files the server lacks, and applies the course. `--dry-run` reports what would change without changing anything. `--prune` removes sections and lessons no longer in the directory, except lessons learners have started.
- `academyos course status <dir>` compares the directory with the academy: what is new, changed, or removed on each side.
- `academyos course pull <dir>` writes changes made in the academy, such as edited narration, back to the directory, refusing to overwrite local edits unless forced.
- `academyos course publish <dir>` publishes the course.
- After a push, the academy's IDs for the course, lessons, and quizzes are written to the course's `published.json`, so a fresh clone knows what is live.

### Output

- Text by default; `--format json` for tools and agents.
- Exit codes distinguish success, a general failure, validation errors, a stale copy, and an authentication failure.
- Validation errors from the server use the same finding format as course-spec, with file, location, message, and fix.

## Success Criteria

1. A ProductBuildersHQ course, kept in course-spec format, is pushed with its decks and quizzes to a local AcademyOS and then to app.academyos.io, by a person and by an AI agent following only the README.
2. Pushing it again unchanged reports nothing to do.
3. Narration edited in the academy comes back with `pull`, and a push from the older copy is refused with a message that says to pull.
4. `go list -deps ./...` in this module contains no AcademyOS server packages.
5. After that, the repository is made public and v0.1.0 is tagged.
