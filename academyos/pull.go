package academyos

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strings"

	coursespec "github.com/academyosio/course-spec"
)

// PullOptions change a pull.
type PullOptions struct {
	// Org is the academy; empty means the API key's academy.
	Org string
	// CourseID is the course to pull; empty means the one in the
	// directory's course.json. Give it to pull a course into an empty
	// directory.
	CourseID string
	// DryRun reports what would change without writing anything.
	DryRun bool
	// Force overwrites local files with uncommitted changes, or any
	// differing file outside a git repository.
	Force bool
	// Progress, when set, is told about each step.
	Progress func(msg string)
}

// PullResult reports a pull.
type PullResult struct {
	Org      string `json:"org"`
	CourseID string `json:"course_id"`
	Revision int    `json:"revision"`
	DryRun   bool   `json:"dry_run,omitempty"`
	// Files counts the academy's files; Unchanged those already the same
	// here.
	Files     int `json:"files"`
	Unchanged int `json:"unchanged"`
	// Added and Updated are package paths written, or that would be.
	Added   []string `json:"added"`
	Updated []string `json:"updated"`
}

// LocalChangesError is a pull refused because it would overwrite files
// changed here: uncommitted changes in git, or, outside git, any file that
// differs from the academy's.
type LocalChangesError struct {
	// Paths are package paths.
	Paths []string
	// NotInGit is true when the course isn't in a git repository, so pull
	// can't tell local edits from the academy's.
	NotInGit bool
}

func (e *LocalChangesError) Error() string {
	if e.NotInGit {
		return fmt.Sprintf("pull would overwrite %d files, and the course isn't in a git repository, so pull can't tell your edits from the academy's; pass --force to overwrite them", len(e.Paths))
	}
	return fmt.Sprintf("pull would overwrite %d files with uncommitted changes; commit or stash them, or pass --force", len(e.Paths))
}

// Pull writes the academy's copy of the course in dir, the course-spec
// directory, into dir and any shared theme beside it, and records the
// academy's revision in published.json. Files only here are left alone.
// It refuses to overwrite local edits: in a git repository, files with
// uncommitted changes; elsewhere, any differing file. Commit before
// pulling, then review what changed with git diff.
func (c *Client) Pull(ctx context.Context, dir string, opts PullOptions) (*PullResult, error) {
	say := func(format string, args ...any) {
		if opts.Progress != nil {
			opts.Progress(fmt.Sprintf(format, args...))
		}
	}
	id := opts.CourseID
	if id == "" {
		m, err := coursespec.ReadCourseManifest(filepath.Join(dir, coursespec.CourseFile))
		if err != nil {
			return nil, fmt.Errorf("read the course ID (or pass the course): %w", err)
		}
		id = m.ID
	}
	org, err := c.ResolveOrg(ctx, opts.Org)
	if err != nil {
		return nil, err
	}
	pkg, err := c.GetPackage(ctx, org, id)
	if err != nil {
		return nil, err
	}
	root, err := packageRoot(dir, pkg.Manifest.CoursePath)
	if err != nil {
		return nil, err
	}
	res := &PullResult{Org: org, CourseID: id, Revision: pkg.Revision, DryRun: opts.DryRun, Files: len(pkg.Manifest.Files),
		Added: []string{}, Updated: []string{}}

	say("comparing %d files with %s", len(pkg.Manifest.Files), c.BaseURL())
	var write []File
	for _, f := range pkg.Manifest.Files {
		if err := checkPackagePath(f.Path); err != nil {
			return nil, err
		}
		sum, err := fileSHA256(filepath.Join(root, filepath.FromSlash(f.Path)))
		switch {
		case errors.Is(err, fs.ErrNotExist):
			res.Added = append(res.Added, f.Path)
			write = append(write, f)
		case err != nil:
			return nil, err
		case sum == f.SHA256:
			res.Unchanged++
		default:
			res.Updated = append(res.Updated, f.Path)
			write = append(write, f)
		}
	}
	sort.Strings(res.Added)
	sort.Strings(res.Updated)

	if len(res.Updated) > 0 && !opts.Force {
		dirty, err := uncommitted(ctx, root, res.Updated)
		var notGit *notGitError
		switch {
		case errors.As(err, &notGit):
			return nil, &LocalChangesError{Paths: res.Updated, NotInGit: true}
		case err != nil:
			return nil, err
		case len(dirty) > 0:
			return nil, &LocalChangesError{Paths: dirty}
		}
	}
	if opts.DryRun {
		return res, nil
	}

	if len(write) > 0 {
		say("downloading %d files", len(write))
	}
	for _, f := range write {
		data, err := c.GetFile(ctx, org, f.SHA256)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", f.Path, err)
		}
		h := sha256.Sum256(data)
		if hex.EncodeToString(h[:]) != f.SHA256 {
			return nil, fmt.Errorf("%s: the academy sent content that doesn't match its checksum", f.Path)
		}
		if err := writeFileAtomic(filepath.Join(root, filepath.FromSlash(f.Path)), data); err != nil {
			return nil, err
		}
	}

	pubPath := filepath.Join(dir, coursespec.PublishedFile)
	pub, err := readPublished(pubPath)
	if err != nil {
		return nil, err
	}
	if pub.Platforms == nil {
		pub.Platforms = map[string]coursespec.PublishedPlatform{}
	}
	p := pub.Platforms[PlatformName]
	if p.BaseURL != c.BaseURL() {
		// Another server's IDs don't apply here.
		p = coursespec.PublishedPlatform{BaseURL: c.BaseURL()}
	}
	p.CourseID, p.Revision = pkg.CourseID, pkg.Revision
	pub.Platforms[PlatformName] = p
	if err := coursespec.WritePublished(pubPath, pub); err != nil {
		return nil, fmt.Errorf("record the academy's revision: %w", err)
	}
	return res, nil
}

// packageRoot returns the package root for the course directory dir:
// dir itself, or the directory coursePath levels above it.
func packageRoot(dir, coursePath string) (string, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	if coursePath == "" || coursePath == "." {
		return abs, nil
	}
	if err := checkPackagePath(coursePath); err != nil {
		return "", fmt.Errorf("course path: %w", err)
	}
	suffix := string(filepath.Separator) + filepath.FromSlash(coursePath)
	if !strings.HasSuffix(abs, suffix) {
		return "", fmt.Errorf("the academy keeps this course at %s within its package; pull into a directory with that path, such as repo/%s", coursePath, coursePath)
	}
	return strings.TrimSuffix(abs, suffix), nil
}

// checkPackagePath requires a relative, clean, forward-slash path that
// stays inside the package root.
func checkPackagePath(p string) error {
	c := path.Clean(p)
	if p == "" || strings.Contains(p, `\`) || strings.HasPrefix(p, "/") || c != p || c == "." || c == ".." ||
		strings.HasPrefix(c, "../") || hasDrive(p) {
		return fmt.Errorf("the academy listed %q, which isn't a path inside the package", p)
	}
	return nil
}

// hasDrive reports a Windows drive prefix, such as C:, on any system.
func hasDrive(p string) bool {
	return len(p) >= 2 && p[1] == ':' && (p[0]|0x20 >= 'a' && p[0]|0x20 <= 'z')
}

func fileSHA256(p string) (string, error) {
	data, err := os.ReadFile(p) //nolint:gosec // a path inside the package root, checked above
	if err != nil {
		return "", err
	}
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:]), nil
}

// writeFileAtomic writes data to p through a temporary file, so a failed
// pull leaves either the old file or the new one.
func writeFileAtomic(p string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(p), ".pull-*")
	if err != nil {
		return err
	}
	_, err = tmp.Write(data)
	if cerr := tmp.Close(); err == nil {
		err = cerr
	}
	if err == nil {
		err = os.Chmod(tmp.Name(), 0o644) //nolint:gosec // course files are meant to be read
	}
	if err == nil {
		err = os.Rename(tmp.Name(), p)
	}
	if err != nil {
		if rerr := os.Remove(tmp.Name()); rerr != nil && !errors.Is(rerr, fs.ErrNotExist) {
			return errors.Join(err, rerr)
		}
		return fmt.Errorf("write %s: %w", p, err)
	}
	return nil
}

type notGitError struct{ msg string }

func (e *notGitError) Error() string { return e.msg }

// uncommitted returns the package paths, of those given, that have
// uncommitted changes in the git repository holding root.
func uncommitted(ctx context.Context, root string, paths []string) ([]string, error) {
	top, err := gitOutput(ctx, root, "rev-parse", "--show-toplevel")
	if err != nil {
		return nil, err
	}
	topDir, err := filepath.EvalSymlinks(strings.TrimSpace(string(top)))
	if err != nil {
		return nil, err
	}
	realRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return nil, err
	}
	byRepoPath := map[string]string{}
	args := []string{"status", "--porcelain=v1", "-z", "--untracked-files=all", "--"}
	for _, p := range paths {
		rel, err := filepath.Rel(topDir, filepath.Join(realRoot, filepath.FromSlash(p)))
		if err != nil {
			return nil, err
		}
		rel = filepath.ToSlash(rel)
		byRepoPath[rel] = p
		args = append(args, rel)
	}
	out, err := gitOutput(ctx, topDir, args...)
	if err != nil {
		return nil, err
	}
	var dirty []string
	for _, entry := range bytes.Split(out, []byte{0}) {
		if len(entry) < 4 {
			continue // the source of a rename, or the trailing separator
		}
		if p, ok := byRepoPath[string(entry[3:])]; ok {
			dirty = append(dirty, p)
		}
	}
	sort.Strings(dirty)
	return dirty, nil
}

func gitOutput(ctx context.Context, dir string, args ...string) ([]byte, error) {
	// The program is fixed, and file paths only follow "--" as pathspecs.
	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", dir}, args...)...) //nolint:gosec // see above
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		var ee *exec.ExitError
		if errors.Is(err, exec.ErrNotFound) || (errors.As(err, &ee) && strings.Contains(stderr.String(), "not a git repository")) {
			return nil, &notGitError{msg: "not in a git repository"}
		}
		return nil, fmt.Errorf("git %s: %w: %s", args[0], err, strings.TrimSpace(stderr.String()))
	}
	return out, nil
}
