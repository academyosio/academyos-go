# Command Line

```bash
academyos whoami                       # who the key acts for, its academy and scopes
academyos course validate <dir>        # check a course; exits 2 on errors
academyos course push <dir>            # validate, upload what the academy lacks, apply
academyos course push --dry-run <dir>  # report what would change
academyos course pull <dir>            # write the academy's copy of the course here
academyos course pull --course <id> <dir>  # pull a course into an empty directory
academyos course publish <dir>         # publish the course
academyos pronunciations push [path]   # replace the academy's pronunciations
academyos pronunciations push --dry-run [path]  # list what would change
academyos pronunciations get           # print the academy's pronunciations
academyos version
```

Every command takes `--format json`.

## Courses

`push` uploads only files the academy doesn't have, applies the whole course
in one step, and matches lessons by ID, so learners keep their progress. It
writes the academy's IDs and revision to the course's `published.json`.
Commit that file: the next push sends the revision, and if the course
changed in the academy since then, the push is refused instead of
overwriting those edits.

A typical push:

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

### Pulling a course

`pull` brings changes made in the academy, such as pronunciations or
narration edited there, back into the course directory, so they can be
committed and reviewed like any other change. It writes the files that
differ from the academy's copy, leaves files that exist only here alone,
and records the academy's revision in `published.json`, so the next push
isn't refused as stale.

```bash
git commit -am "Before pulling"   # pull refuses to overwrite uncommitted changes
academyos course pull --dry-run courses/becoming-a-product-builder
academyos course pull courses/becoming-a-product-builder
git diff                          # review what came down
```

| Flag | Meaning |
| --- | --- |
| `--course <id>` | The course to pull, for an empty directory; otherwise the ID in `course.json` |
| `--dry-run` | List the files that would be added or updated, writing nothing |
| `--force` | Overwrite local changes |

Pull protects your edits. In a git repository, it refuses to overwrite a
file with uncommitted changes, lists them, and exits with code 3; commit or
stash them, or pass `--force`. Outside git, it can't tell your edits from the
academy's, so any file that differs needs `--force`.

The directory is the course's own. A course kept at `courses/<id>` in its
repository is pulled into `repo/courses/<id>`, where shared files such as a
theme beside it are written too.

## Pronunciations

`pronunciations push` sends a `pronunciations.json`, by default the one in
the current directory, usually the course repository's root, as the
academy's pronunciations. Narration applies a course's own
`pronunciations.json` first, then the academy's, then the platform's, so a
term shared by several courses belongs at the root. The push lists the
terms added, changed and removed, and removes from the academy any term the
file leaves out. An empty respelling says the term as written.

## Exit codes

| Code | Meaning |
| --- | --- |
| 0 | Success |
| 1 | Failure: network, server, or usage |
| 2 | The course has validation errors |
| 3 | The academy's copy is newer, or the course exists without a package: pull first, or pass `--force`. For `pull`, files changed here would be overwritten |
| 4 | The API key is missing, invalid, expired, or not allowed |
