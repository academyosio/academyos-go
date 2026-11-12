# Command Line

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
| 3 | The academy's copy is newer, or the course exists without a package: pull first, or pass `--force` |
| 4 | The API key is missing, invalid, expired, or not allowed |
