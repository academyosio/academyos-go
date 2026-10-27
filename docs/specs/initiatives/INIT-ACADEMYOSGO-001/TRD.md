# AcademyOS Go SDK and CLI: Technical Requirements

## Module

- Module path: `github.com/academyosio/academyos-go`.
- Packages:

  | Package | Purpose |
  | --- | --- |
  | `academyos` | The SDK: `Client`, options, typed errors, and one method per API call |
  | `coursepkg` | Turns a course-spec directory into a package: the files it needs, their checksums, and the package root |
  | `cmd/academyos` | The Cobra CLI, a thin layer over the SDK |

- Dependencies: `github.com/academyosio/course-spec`, `github.com/spf13/cobra`, and the standard library. No AcademyOS server packages; a test checks the dependency list.

## Configuration

```go
c, err := academyos.New(academyos.Options{
    BaseURL:    "",                          // default https://app.academyos.io
    APIKey:     os.Getenv("ACADEMYOS_API_KEY"),
    HTTPClient: nil,                         // default: a client with a timeout
    UserAgent:  "",                          // default academyos-go/<version>
})
```

- `BaseURL` precedence in the CLI: `--api-url`, then `ACADEMYOS_API_URL`, then `https://app.academyos.io`.
- `New` refuses a key with an `http://` URL unless the host is `localhost`, `127.0.0.1`, or `::1`.
- Keys look like `aos_live_…`. The SDK sends `Authorization: Bearer <key>` and never logs the key or prints it in errors.

## Course Packages

A push sends the course as files, not as one JSON document, so decks, the files they load, and translations travel with the outline.

`coursepkg.Build(dir)` loads the course with course-spec and lists:

- `course.json`, `pronunciations.json`, the description file, thumbnail, and certificate template;
- each released section's `section.json`, quiz, and text lessons;
- each deck, its `voiceover/transcript.json`, and every relative file it loads (styles, scripts, fonts, images, media);
- the theme directory's files.

It leaves out `published.json` (the server writes its own IDs back through the response), `build/` output, and files nothing references.

**Package root.** Paths in a package are relative to the package root: the course directory, or the nearest directory holding both the course and a shared theme outside it, such as `courses/` for `"theme": "../shared/theme"`. The package records the course directory's path within the root. No path may leave the root.

**Manifest.** The package is a list of `{path, sha256, size}` sorted by path, plus the course path. Its own checksum identifies the package version.

Limits match the server's: 20 MB per file, 200 MB per package, standard web media types only. `Build` reports a file over a limit before anything is uploaded.

## API Calls

The SDK uses these AcademyOS endpoints. The server side is specified with AcademyOS's deck playback work, which replaces its per-deck storage with this package design.

| Call | Endpoint | Purpose |
| --- | --- | --- |
| `Me` | `GET /api/v1/users/me` | The key's person, academies and roles, and the key's academy, scopes, and expiry |
| `GetPackage` | `GET /api/v1/organizations/{org}/courses/{courseId}/package` | The current manifest and its revision |
| `PutPackage` | `PUT /api/v1/organizations/{org}/courses/{courseId}/package` | Propose a manifest; returns missing checksums, or applies it |
| `PutFile` | `PUT /api/v1/organizations/{org}/files/{sha256}` | Upload one file; the server checks the hash |
| `GetFile` | `GET /api/v1/organizations/{org}/files/{sha256}` | Download one file |
| `Publish` | `POST /api/v1/organizations/{org}/courses/{courseId}/publish` | Publish the course |

`courseId` is the course-spec ID.

### Push

1. Validate locally with course-spec. Errors stop the push.
2. Build the package.
3. `PutPackage` with the manifest, the revision the local copy was last synced at, and `dryRun` and `prune` flags.
4. When the response lists missing checksums, `PutFile` each one, at most four at a time, then repeat step 3.
5. The server loads the package with course-spec, applies the same validation, and updates the course in one transaction. It answers with a summary (created, updated, unchanged, kept, and removed sections, lessons, and quizzes), the new revision, and its IDs for the course, lessons, and quizzes.
6. Write the IDs, base URL, and revision to `published.json` under `platforms.academyos`.

With `dryRun`, the server reports the summary without applying it, and step 6 is skipped.

### Revisions and Conflicts

Every applied package, and every edit made in the academy such as narration changed in review, raises the course's revision. The local copy's revision is the one in `published.json`.

- A push whose revision is older than the server's is refused with `409` and the list of files that changed in the academy. The CLI says to run `pull`, or to pass `--force` to overwrite them.
- A first push has no revision. If the course already exists in the academy, the push is refused unless `--force` is given, so a course isn't overwritten by an unrelated directory with the same ID.

### Pull

`GetPackage`, then `GetFile` for each file whose checksum differs from the local file, written in place. A local file that changed since the last sync is not overwritten; pull lists it and stops unless `--force` is given. Pull updates the revision in `published.json`.

### Status

Compares the local package with `GetPackage`: files changed locally, files changed in the academy, and both. It also includes course-spec's lesson status, so a lesson changed since its video was published is reported.

## Errors

The SDK returns `*academyos.APIError` for non-2xx responses:

```go
type APIError struct {
    Status   int
    Code     string             // such as "stale_revision", "invalid_course", "unauthorized"
    Message  string
    Findings []validate.Finding // server validation, in course-spec's format
    Changed  []string           // for stale_revision: paths changed in the academy
}
```

The server's error body is `{"code": "...", "message": "...", "findings": [...], "changed": [...]}`.

## CLI

```
academyos whoami
academyos course validate <dir> [--strict-narration]
academyos course status <dir>
academyos course push <dir> [--dry-run] [--prune] [--force]
academyos course pull <dir> [--force]
academyos course publish <dir>
academyos version
```

Global flags: `--api-url`, `--org`, `--format text|json`.

Exit codes:

| Code | Meaning |
| --- | --- |
| 0 | Success |
| 1 | Failure: network, server, or usage |
| 2 | The course has validation errors, locally or on the server |
| 3 | Stale copy: pull first, or force |
| 4 | Authentication or permission failure |

## Testing

- Unit tests against an `httptest` server that implements the endpoints above, including missing-file rounds, conflicts, and errors.
- `coursepkg` tests over course-spec's test course, including a shared theme outside the course directory.
- A test that fails if any package in `go list -deps` is from the AcademyOS server module.
