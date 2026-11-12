# API Reference

The [API reference](api-reference/index.html) documents the AcademyOS HTTP
API that an API key may call, for clients in any language. Download the
OpenAPI 3.1 document: [academyos.json](api-reference/academyos.json).

## Server and authentication

Requests go to `https://app.academyos.io`, with the API key as a bearer
token:

```bash
curl -H "Authorization: Bearer $ACADEMYOS_API_KEY" \
  https://app.academyos.io/api/v1/users/me
```

A key belongs to one academy, given by `{orgSlug}` in paths, and has
`courses:read` or `courses:write`. Reading needs either; anything that
changes a course or the academy's pronunciations needs `courses:write`.

## What it covers

| Area | Operations |
| --- | --- |
| Identity | Who the key acts for, its academy and scopes |
| Courses | List, create, read, update, delete and publish |
| Sections and lessons | Create, update, delete and reorder; read a lesson, its narration and its player |
| Quizzes | Create, read, update and delete a lesson's quiz |
| Versions | Create, read, update and publish course versions and their content updates |
| Course packages | Read and apply a course package, and upload or download its files by SHA-256 |
| Pronunciations | Read and replace the academy's pronunciations, review them, and suggest respellings |

Errors use the problem format (`application/problem+json`) with `title`,
`status` and `detail`.

## Where the document comes from

The document is generated from the AcademyOS server's code: its operations
are those an API key may call there, and the server's tests keep the two in
step. This repository's copy is `openapi/academyos.json`; the SDK's tests
check that every request the client makes is in it.
