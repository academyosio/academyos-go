// Package coursepkg turns a course-spec directory into a package: the files
// the course needs, their checksums, and the package root.
package coursepkg

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	coursespec "github.com/academyosio/course-spec"
)

// Limits match the server's.
const (
	MaxFileSize    = 20 << 20
	MaxPackageSize = 200 << 20
)

// File is one file in a package.
type File struct {
	// Path is relative to the package root, with forward slashes.
	Path   string
	SHA256 string
	Size   int64
	// Abs is the file's absolute path.
	Abs string
}

// Package is a course directory's files, rooted so every file is inside.
type Package struct {
	// Root is the package root: the course directory, or the nearest
	// directory holding both it and a shared theme.
	Root string
	// CoursePath is the course directory relative to Root, "." when equal.
	CoursePath string
	// CourseID is the course's ID from course.json.
	CourseID string
	Files    []File
}

// File returns the package file with the given checksum.
func (p *Package) File(sum string) (File, bool) {
	for _, f := range p.Files {
		if f.SHA256 == sum {
			return f, true
		}
	}
	return File{}, false
}

// Build loads the course at dir and lists the files it needs:
// course.json, pronunciations, the description, thumbnail and certificate
// template, each released section's manifest, quiz and text lessons, each
// deck with its transcript and the files it loads, and the theme. It
// leaves out published.json and build output.
func Build(dir string) (*Package, error) {
	courseDir, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	c, err := coursespec.Load(courseDir)
	if err != nil {
		return nil, err
	}
	set := map[string]bool{}
	add := func(p string) { set[filepath.Clean(p)] = true }
	addIfExists := func(p string) error {
		_, err := os.Stat(p)
		switch {
		case err == nil:
			add(p)
			return nil
		case errors.Is(err, fs.ErrNotExist):
			return nil
		default:
			return err
		}
	}

	m := c.Manifest
	add(filepath.Join(courseDir, coursespec.CourseFile))
	if err := addIfExists(filepath.Join(courseDir, coursespec.PronunciationsFile)); err != nil {
		return nil, err
	}
	for _, rel := range []string{m.DescriptionFile, m.Thumbnail} {
		if rel != "" {
			add(filepath.Join(courseDir, rel))
		}
	}
	if m.Certificate != nil && m.Certificate.Template != "" {
		add(filepath.Join(courseDir, m.Certificate.Template))
	}
	theme := m.Theme
	if theme == "" {
		theme = coursespec.DefaultTheme
	}
	if err := addTree(filepath.Join(courseDir, theme), add); err != nil {
		return nil, err
	}

	for _, s := range c.Sections {
		if s.Manifest == nil || !s.Ref.IsReleased() {
			continue
		}
		add(filepath.Join(s.Dir, coursespec.SectionFile))
		qp, err := coursespec.QuizPath(s.Dir, s.Manifest)
		if err != nil {
			return nil, err
		}
		if qp != "" {
			add(qp)
		}
		for _, l := range s.Lessons {
			if l.Type == coursespec.LessonText && l.Content != "" {
				add(filepath.Join(s.Dir, filepath.FromSlash(l.Content)))
			}
		}
		if s.Deck == nil {
			continue
		}
		deckPath := filepath.Join(s.Dir, filepath.FromSlash(s.Manifest.Source.Path))
		add(deckPath)
		if err := addIfExists(filepath.Join(filepath.Dir(deckPath), filepath.FromSlash(coursespec.TranscriptFile))); err != nil {
			return nil, err
		}
		for _, r := range s.Deck.Refs {
			if r.Local() {
				add(filepath.Join(filepath.Dir(deckPath), filepath.FromSlash(r.Path())))
			}
		}
	}

	paths := make([]string, 0, len(set))
	for p := range set {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	root := courseDir
	for _, p := range paths {
		root = commonDir(root, filepath.Dir(p))
	}
	if repo := repoRoot(courseDir); repo != "" && !inside(root, repo) {
		return nil, fmt.Errorf("the course loads files outside its repository (package root %s)", root)
	}

	pkg := &Package{Root: root, CourseID: m.ID}
	rel, err := filepath.Rel(root, courseDir)
	if err != nil {
		return nil, err
	}
	pkg.CoursePath = filepath.ToSlash(rel)
	var total int64
	for _, p := range paths {
		f, err := hashFile(root, p)
		if err != nil {
			return nil, err
		}
		if f.Size > MaxFileSize {
			return nil, fmt.Errorf("%s is %d MB; files are limited to %d MB", f.Path, f.Size>>20, MaxFileSize>>20)
		}
		total += f.Size
		pkg.Files = append(pkg.Files, f)
	}
	if total > MaxPackageSize {
		return nil, fmt.Errorf("the package is %d MB; packages are limited to %d MB", total>>20, MaxPackageSize>>20)
	}
	sort.Slice(pkg.Files, func(i, j int) bool { return pkg.Files[i].Path < pkg.Files[j].Path })
	return pkg, nil
}

// addTree adds every regular file under dir, skipping hidden files; a
// missing directory adds nothing.
func addTree(dir string, add func(string)) error {
	if _, err := os.Stat(dir); errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	return filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if strings.HasPrefix(d.Name(), ".") && p != dir {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.Type().IsRegular() {
			add(p)
		}
		return nil
	})
}

func hashFile(root, p string) (File, error) {
	data, err := os.ReadFile(p) //nolint:gosec // paths come from the course's own references
	if err != nil {
		return File{}, err
	}
	rel, err := filepath.Rel(root, p)
	if err != nil {
		return File{}, err
	}
	sum := sha256.Sum256(data)
	return File{Path: filepath.ToSlash(rel), SHA256: hex.EncodeToString(sum[:]), Size: int64(len(data)), Abs: p}, nil
}

// commonDir returns the deepest directory containing both a and b.
func commonDir(a, b string) string {
	for !inside(b, a) {
		parent := filepath.Dir(a)
		if parent == a {
			return a
		}
		a = parent
	}
	return a
}

func inside(p, dir string) bool {
	r, err := filepath.Rel(dir, p)
	return err == nil && r != ".." && !strings.HasPrefix(r, ".."+string(filepath.Separator))
}

func repoRoot(dir string) string {
	for d := dir; ; d = filepath.Dir(d) {
		if _, err := os.Stat(filepath.Join(d, ".git")); err == nil {
			return d
		}
		if filepath.Dir(d) == d {
			return ""
		}
	}
}
