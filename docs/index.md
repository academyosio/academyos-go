# AcademyOS Go SDK

The Go SDK and command line for [AcademyOS](https://academyos.io). Keep a
course as a [course-spec](https://github.com/academyosio/course-spec)
directory in git, and use `academyos` to check it, upload it with its slide
decks, translations and quizzes, and publish it.

```bash
go install github.com/academyosio/academyos-go/cmd/academyos@latest
export ACADEMYOS_API_KEY=aos_live_…
academyos course push courses/building-web-apis
```

## Where to start

| You want to | Read |
| --- | --- |
| Install the CLI and connect it to an academy | [Getting Started](getting-started.md) |
| Validate, push and publish courses | [Command Line](cli.md) |
| Call AcademyOS from Go | [Go SDK](sdk.md) |
| Let a coding agent publish courses | [AI Agents](agents.md) |
| Call the HTTP API from another language | [API Reference](api.md) |

## What an API key can do

An API key belongs to one academy. With it you can create, update and
publish that academy's courses, upload course packages and their files, and
manage the academy's pronunciations. Enrollments, learner progress, analytics,
billing and key management aren't available to API keys.
