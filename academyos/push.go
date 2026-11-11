package academyos

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	coursespec "github.com/academyosio/course-spec"
	"github.com/academyosio/course-spec/validate"

	"github.com/academyosio/academyos-go/coursepkg"
)

// PlatformName is the key AcademyOS uses in published.json's platforms.
const PlatformName = "academyos"

// uploadWorkers is how many files upload at once.
const uploadWorkers = 4

// ValidationError is a course that failed course-spec validation locally.
type ValidationError struct {
	Findings []validate.Finding
}

func (e *ValidationError) Error() string {
	n, _ := validate.Counts(e.Findings)
	return fmt.Sprintf("the course has %d validation errors", n)
}

// PushOptions change a push.
type PushOptions struct {
	// Org is the academy; empty means the API key's academy.
	Org string
	// DryRun reports what would change without changing anything.
	DryRun bool
	// Prune removes sections and lessons the course no longer lists,
	// except lessons learners have used.
	Prune bool
	// Force overwrites a newer copy in the academy.
	Force bool
	// Progress, when set, is told about each step.
	Progress func(msg string)
}

// PushResult reports a push.
type PushResult struct {
	Org      string
	CourseID string
	// Files and Uploaded count the package's files and those uploaded.
	Files    int
	Uploaded int
	// Warnings are local validation warnings.
	Warnings []validate.Finding
	Response *PutPackageResponse
}

// Push validates the course-spec directory dir, uploads the files the
// server lacks, applies the package, and records the academy's IDs and
// revision in the course's published.json.
func (c *Client) Push(ctx context.Context, dir string, opts PushOptions) (*PushResult, error) {
	say := func(format string, args ...any) {
		if opts.Progress != nil {
			opts.Progress(fmt.Sprintf(format, args...))
		}
	}
	findings := validate.Course(dir, validate.Options{})
	if n, _ := validate.Counts(findings); n > 0 {
		return nil, &ValidationError{Findings: findings}
	}
	pkg, err := coursepkg.Build(dir)
	if err != nil {
		return nil, err
	}
	org, err := c.ResolveOrg(ctx, opts.Org)
	if err != nil {
		return nil, err
	}
	res := &PushResult{Org: org, CourseID: pkg.CourseID, Files: len(pkg.Files), Warnings: findings}

	pubPath := filepath.Join(dir, coursespec.PublishedFile)
	pub, err := readPublished(pubPath)
	if err != nil {
		return nil, err
	}
	req := &PutPackageRequest{Manifest: manifestOf(pkg), DryRun: opts.DryRun, Prune: opts.Prune, Force: opts.Force}
	if p, ok := pub.Platforms[PlatformName]; ok && p.Revision > 0 && p.BaseURL == c.BaseURL() {
		rev := p.Revision
		req.BaseRevision = &rev
	}

	say("checking %d files with %s", len(pkg.Files), c.BaseURL())
	for round := 0; ; round++ {
		resp, err := c.PutPackage(ctx, org, pkg.CourseID, req)
		if err != nil {
			return nil, err
		}
		if len(resp.Missing) == 0 {
			res.Response = resp
			break
		}
		if round == 2 {
			return nil, fmt.Errorf("the server still lacks %d files after uploading them", len(resp.Missing))
		}
		say("uploading %d files", len(resp.Missing))
		if err := c.upload(ctx, org, pkg, resp.Missing); err != nil {
			return nil, err
		}
		res.Uploaded += len(resp.Missing)
	}

	if res.Response.Applied {
		if pub.Platforms == nil {
			pub.Platforms = map[string]coursespec.PublishedPlatform{}
		}
		pub.Platforms[PlatformName] = coursespec.PublishedPlatform{
			BaseURL:  c.BaseURL(),
			CourseID: res.Response.CourseID,
			Lessons:  res.Response.Lessons,
			Quizzes:  res.Response.Quizzes,
			Revision: res.Response.Revision,
		}
		if err := coursespec.WritePublished(pubPath, pub); err != nil {
			return nil, fmt.Errorf("record the academy's IDs: %w", err)
		}
	}
	return res, nil
}

func (c *Client) upload(ctx context.Context, org string, pkg *coursepkg.Package, sums []string) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	work := make(chan string)
	errs := make(chan error, len(sums))
	var wg sync.WaitGroup
	for range min(uploadWorkers, len(sums)) {
		wg.Go(func() {
			for sum := range work {
				errs <- c.uploadOne(ctx, org, pkg, sum)
			}
		})
	}
	for _, s := range sums {
		work <- s
	}
	close(work)
	wg.Wait()
	close(errs)
	var all []error
	for err := range errs {
		if err != nil {
			all = append(all, err)
		}
	}
	return errors.Join(all...)
}

func (c *Client) uploadOne(ctx context.Context, org string, pkg *coursepkg.Package, sum string) error {
	f, ok := pkg.File(sum)
	if !ok {
		return fmt.Errorf("the server asked for %s, which isn't in the package", sum)
	}
	data, err := os.ReadFile(f.Abs)
	if err != nil {
		return err
	}
	if err := c.PutFile(ctx, org, sum, data); err != nil {
		return fmt.Errorf("upload %s: %w", f.Path, err)
	}
	return nil
}

// ResolveOrg returns org, or the API key's academy when org is empty.
func (c *Client) ResolveOrg(ctx context.Context, org string) (string, error) {
	me, err := c.Me(ctx)
	if err != nil {
		return "", err
	}
	if me.APIKey == nil || me.APIKey.OrganizationSlug == "" {
		if org == "" {
			return "", errors.New("the API key isn't tied to an academy; pass --org")
		}
		return org, nil
	}
	if org != "" && org != me.APIKey.OrganizationSlug {
		return "", fmt.Errorf("the API key belongs to %q, not %q", me.APIKey.OrganizationSlug, org)
	}
	return me.APIKey.OrganizationSlug, nil
}

// PublishCourse publishes the course in dir, or the course with that ID.
func (c *Client) PublishCourse(ctx context.Context, dirOrID, org string) (string, string, error) {
	id := dirOrID
	if m, err := coursespec.ReadCourseManifest(filepath.Join(dirOrID, coursespec.CourseFile)); err == nil {
		id = m.ID
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", "", err
	}
	org, err := c.ResolveOrg(ctx, org)
	if err != nil {
		return "", "", err
	}
	return org, id, c.Publish(ctx, org, id)
}

func readPublished(path string) (*coursespec.Published, error) {
	p, err := coursespec.ReadPublished(path)
	switch {
	case err == nil:
		return p, nil
	case errors.Is(err, os.ErrNotExist):
		return &coursespec.Published{}, nil
	default:
		return nil, err
	}
}

func manifestOf(p *coursepkg.Package) Manifest {
	m := Manifest{CoursePath: p.CoursePath}
	for _, f := range p.Files {
		m.Files = append(m.Files, File{Path: f.Path, SHA256: f.SHA256, Size: f.Size})
	}
	return m
}
