# academyos-go

![coverage](https://img.shields.io/badge/coverage-78.5%25-yellowgreen)

The Go SDK and command line for [AcademyOS](https://academyos.io). Keep a
course as a [course-spec](https://github.com/academyosio/course-spec)
directory in git, and use `academyos` to check it, upload it with its slide
decks, translations, and quizzes, and publish it.

## Install

```bash
go install github.com/academyosio/academyos-go/cmd/academyos@latest
```

## Configure

| Variable | Meaning |
| --- | --- |
| `ACADEMYOS_API_KEY` | An API key, `aos_live_…`, created in AcademyOS by an instructor or admin. Required |
| `ACADEMYOS_API_URL` | The server. Default `https://app.academyos.io`; for a local server, `http://localhost:8080` |

The key is read only from the environment, never from a flag. It is sent
only over HTTPS, or plain HTTP to `localhost`. A key belongs to one academy,
so commands use that academy; `--org` is needed only to double-check it.

## Commands

```bash
academyos whoami                       # who the key acts for, its academy and scopes
academyos course validate <dir>        # check a course; exits 2 on errors
academyos course push <dir>            # validate, upload what the academy lacks, apply
academyos course push --dry-run <dir>  # report what would change
academyos course publish <dir>         # publish the course
academyos pronunciations push [path]   # replace the academy's pronunciations
academyos pronunciations push --dry-run [path]  # list what would change
academyos pronunciations get           # print the academy's pronunciations
academyos version
```

Every command takes `--format json`.

`push` uploads only files the academy doesn't have, applies the whole course
in one step, and matches lessons by ID, so learners keep their progress. It
writes the academy's IDs and revision to the course's `published.json`.
Commit that file: the next push sends the revision, and if the course
changed in the academy since then, the push is refused instead of
overwriting those edits.

`pronunciations push` sends a `pronunciations.json`, by default the one in
the current directory, usually the course repository's root, as the
academy's pronunciations. Narration applies a course's own
`pronunciations.json` first, then the academy's, then the platform's, so a
term shared by several courses belongs at the root. The push lists the
terms added, changed and removed, and removes from the academy any term the
file leaves out. An empty respelling says the term as written.

Exit codes:

| Code | Meaning |
| --- | --- |
| 0 | Success |
| 1 | Failure: network, server, or usage |
| 2 | The course has validation errors |
| 3 | The academy's copy is newer, or the course exists without a package: pull first, or pass `--force` |
| 4 | The API key is missing, invalid, expired, or not allowed |

## For AI Agents

Add this to the course repository's `CLAUDE.md` or `AGENTS.md`:

```markdown
## Publishing to AcademyOS

- Courses are course-spec directories. After every change, run
  `academyos course validate <dir>` and fix every error before pushing.
- Upload with `academyos course push <dir>`, then commit the updated
  `published.json`.
- The server and key come from ACADEMYOS_API_URL and ACADEMYOS_API_KEY.
  Never print, log, or commit the key.
- Exit code 3 means the course changed in the academy. Stop and ask before
  using --force.
- Never change a published lesson's ID (a slide's data-id with
  data-lesson); learner progress is keyed by it.
- Pronunciations shared by several courses go in the repository's root
  `pronunciations.json`; push them with `academyos pronunciations push`
  (try `--dry-run` first). A course's own `pronunciations.json` wins over
  the academy's.
- Use --format json when you need to read the output.
```

A typical session:

```text
$ academyos course push courses/becoming-a-product-builder
checking 42 files with https://app.academyos.io
uploading 3 files
applied becoming-a-product-builder to https://app.academyos.io in productbuildershq, revision 7
  files     42 in the package, 3 uploaded
  sections  0 created, 1 updated, 4 unchanged, 0 kept, 0 removed
  lessons   1 created, 2 updated, 19 unchanged, 0 kept, 0 removed
  quizzes   0 created, 1 updated, 4 unchanged, 0 kept, 0 removed
recorded the academy's IDs in published.json
```

## SDK

```go
c, err := academyos.New(academyos.Options{APIKey: os.Getenv("ACADEMYOS_API_KEY")})
if err != nil {
    return err
}
res, err := c.Push(ctx, "courses/building-web-apis", academyos.PushOptions{})
```

| Package | Purpose |
| --- | --- |
| `academyos` | The client: `Me`, `Push`, `PublishCourse`, and the package and file calls |
| `coursepkg` | The files a course-spec directory needs, with checksums and a package root that includes a shared theme |

## Specification

Requirements, design, plan, and roadmap are in
[`docs/specs/initiatives/INIT-ACADEMYOSGO-001`](docs/specs/initiatives/INIT-ACADEMYOSGO-001).

## License

MIT
