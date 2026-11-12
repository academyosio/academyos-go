# Go SDK

```bash
go get github.com/academyosio/academyos-go
```

| Package | Purpose |
| --- | --- |
| `academyos` | The client: `Me`, `Push`, `PublishCourse`, pronunciations, and the package and file calls |
| `coursepkg` | The files a course-spec directory needs, with checksums and a package root that includes a shared theme |

## Create a client

```go
c, err := academyos.New(academyos.Options{
    APIKey: os.Getenv("ACADEMYOS_API_KEY"),
})
if err != nil {
    return err
}
```

`BaseURL` defaults to `https://app.academyos.io`. The client refuses to send
a key over plain HTTP except to `localhost` and loopback addresses. A
`Client` is safe for concurrent use.

## Push and publish a course

```go
res, err := c.Push(ctx, "courses/building-web-apis", academyos.PushOptions{})
if err != nil {
    return err
}
fmt.Println(res.CourseID, res.Uploaded, "files uploaded")

if _, _, err := c.PublishCourse(ctx, "courses/building-web-apis", ""); err != nil {
    return err
}
```

`PushOptions`:

| Field | Meaning |
| --- | --- |
| `Org` | The academy; empty means the API key's academy |
| `DryRun` | Report what would change without changing anything |
| `Prune` | Remove sections and lessons the course no longer lists, except lessons learners have used |
| `Force` | Overwrite a newer copy in the academy |
| `Progress` | When set, told about each step |

## Pronunciations

```go
res, err := c.PushPronunciations(ctx, "pronunciations.json", academyos.PronunciationsOptions{DryRun: true})
```

`GetPronunciations` and `PutPronunciations` read and replace the academy's
pronunciations directly, and `DiffPronunciations` compares two sets.

## Lower-level calls

| Method | Request |
| --- | --- |
| `Me` | `GET /api/v1/users/me` |
| `GetPackage`, `PutPackage` | `GET`, `PUT /api/v1/organizations/{orgSlug}/courses/{courseSlug}/package` |
| `GetFile`, `PutFile` | `GET`, `PUT /api/v1/organizations/{orgSlug}/files/{sha256}` |
| `Publish` | `POST /api/v1/organizations/{orgSlug}/courses/{courseSlug}/publish` |
| `GetPronunciations`, `PutPronunciations` | `GET`, `PUT /api/v1/organizations/{orgSlug}/pronunciations` |

Each is described in the [API Reference](api.md). A test checks that every
request the client makes is in the reference's OpenAPI document.

## Errors

| Type | When |
| --- | --- |
| `*academyos.ValidationError` | The course failed course-spec validation locally; `Findings` lists why |
| `*academyos.APIError` | The server answered with an error: `Status`, a `Code` such as `stale_revision`, `course_exists`, `invalid_course` or `unauthorized`, and the server's `Findings` or `Changed` paths |

```go
var apiErr *academyos.APIError
if errors.As(err, &apiErr) && apiErr.Code == "stale_revision" {
    // The academy's copy changed since the last push.
}
```
