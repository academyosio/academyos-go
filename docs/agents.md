# AI Agents

A coding agent can keep courses in a repository and publish them with the
`academyos` CLI. Add this to the course repository's `CLAUDE.md` or
`AGENTS.md`:

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

Every command takes `--format json`, and the [exit codes](cli.md#exit-codes)
tell an agent what to do next without reading the text.
