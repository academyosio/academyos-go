# Getting Started

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

## Check the key

```bash
academyos whoami
```

`whoami` prints who the key acts for, its academy and its scopes. A key with
`courses:read` can read courses; `courses:write` can also change and publish
them.

## Push a course

```bash
academyos course validate courses/building-web-apis
academyos course push --dry-run courses/building-web-apis
academyos course push courses/building-web-apis
academyos course publish courses/building-web-apis
```

Then commit the course's updated `published.json`. See
[Command Line](cli.md) for every command and exit code.

## Pull changes made in the academy

When pronunciations or narration are edited in AcademyOS, bring them back
into the repository:

```bash
git status                                    # commit or stash your own edits first
academyos course pull courses/building-web-apis
git diff                                      # review, then commit
```

`pull` needs a key with `courses:read`. It refuses to overwrite uncommitted
changes; see [Pulling a course](cli.md#pulling-a-course).
