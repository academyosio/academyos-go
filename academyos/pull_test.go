package academyos

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	coursespec "github.com/academyosio/course-spec"
)

// pushed returns a client and a course pushed to a fake academy.
func pushed(t *testing.T) (*fakeServer, *Client, string) {
	t.Helper()
	fake, srv := newFake(t)
	c, err := New(Options{BaseURL: srv.URL, APIKey: testKey})
	if err != nil {
		t.Fatal(err)
	}
	dir := copyCourse(t)
	if _, err := c.Push(context.Background(), dir, PushOptions{}); err != nil {
		t.Fatal(err)
	}
	return fake, c, dir
}

// editInAcademy changes a package file in the fake academy, as an edit
// made there would, and raises the revision.
func editInAcademy(t *testing.T, fake *fakeServer, p string, content []byte) {
	t.Helper()
	fake.mu.Lock()
	defer fake.mu.Unlock()
	h := sha256.Sum256(content)
	sum := hex.EncodeToString(h[:])
	fake.files[sum] = content
	for i, f := range fake.manifest.Files {
		if f.Path == p {
			fake.manifest.Files[i] = File{Path: p, SHA256: sum, Size: int64(len(content))}
			fake.revision++
			return
		}
	}
	t.Fatalf("%s is not in the package", p)
}

func git(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir, "-c", "user.name=Test", "-c", "user.email=test@example.com", "-c", "commit.gpgsign=false"}, args...)...) //nolint:gosec // test git commands
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

// committed puts dir in a new git repository with everything committed.
func committed(t *testing.T, dir string) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
	git(t, dir, "init", "-q")
	git(t, dir, "add", "-A")
	git(t, dir, "commit", "-q", "-m", "course")
}

func readFile(t *testing.T, p string) []byte {
	t.Helper()
	b, err := os.ReadFile(p) //nolint:gosec // test files
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestPullIntoEmptyDirectory(t *testing.T) {
	fake, c, dir := pushed(t)
	to := filepath.Join(t.TempDir(), "copy")
	res, err := c.Pull(context.Background(), to, PullOptions{CourseID: "building-web-apis"})
	if err != nil {
		t.Fatal(err)
	}
	if res.Revision != 1 || len(res.Added) != len(fake.manifest.Files) || len(res.Updated) != 0 || res.Unchanged != 0 {
		t.Fatalf("pull = %+v", res)
	}
	for _, f := range fake.manifest.Files {
		got := readFile(t, filepath.Join(to, filepath.FromSlash(f.Path)))
		if want := readFile(t, filepath.Join(dir, filepath.FromSlash(f.Path))); !bytes.Equal(got, want) {
			t.Errorf("%s differs from the pushed file", f.Path)
		}
	}
	pub, err := coursespec.ReadPublished(filepath.Join(to, coursespec.PublishedFile))
	if err != nil {
		t.Fatal(err)
	}
	if p := pub.Platforms[PlatformName]; p.Revision != 1 || p.CourseID != "c-1" || p.BaseURL != c.BaseURL() {
		t.Errorf("published = %+v", p)
	}

	// Pulling again changes nothing.
	res, err = c.Pull(context.Background(), to, PullOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Added)+len(res.Updated) != 0 || res.Unchanged != res.Files {
		t.Errorf("second pull = %+v", res)
	}
}

func TestPullBringsAcademyEdits(t *testing.T) {
	fake, c, dir := pushed(t)
	committed(t, dir)
	edited := []byte(`{"Lamborghini": {"en-US": "lam-bor-GHEE-nee"}}` + "\n")
	editInAcademy(t, fake, "pronunciations.json", edited)

	res, err := c.Pull(context.Background(), dir, PullOptions{DryRun: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Updated) != 1 || bytes.Equal(readFile(t, filepath.Join(dir, "pronunciations.json")), edited) {
		t.Fatalf("dry run = %+v, or it wrote the file", res)
	}

	res, err = c.Pull(context.Background(), dir, PullOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Updated) != 1 || res.Updated[0] != "pronunciations.json" || res.Revision != 2 {
		t.Fatalf("pull = %+v", res)
	}
	if !bytes.Equal(readFile(t, filepath.Join(dir, "pronunciations.json")), edited) {
		t.Error("the academy's edit wasn't written")
	}
	pub, err := coursespec.ReadPublished(filepath.Join(dir, coursespec.PublishedFile))
	if err != nil {
		t.Fatal(err)
	}
	if p := pub.Platforms[PlatformName]; p.Revision != 2 || p.Lessons["resources"] != "l-1" {
		t.Errorf("published = %+v", p)
	}
}

func TestPullKeepsUncommittedEdits(t *testing.T) {
	fake, c, dir := pushed(t)
	committed(t, dir)
	mine := []byte(`{"Ferrari": {"en-US": "feh-RAH-ree"}}` + "\n")
	if err := os.WriteFile(filepath.Join(dir, "pronunciations.json"), mine, 0o600); err != nil {
		t.Fatal(err)
	}
	editInAcademy(t, fake, "pronunciations.json", []byte(`{}`+"\n"))

	_, err := c.Pull(context.Background(), dir, PullOptions{})
	var lc *LocalChangesError
	if !errors.As(err, &lc) || lc.NotInGit || len(lc.Paths) != 1 || lc.Paths[0] != "pronunciations.json" {
		t.Fatalf("err = %v", err)
	}
	if !bytes.Equal(readFile(t, filepath.Join(dir, "pronunciations.json")), mine) {
		t.Fatal("the refused pull changed the file")
	}

	if _, err := c.Pull(context.Background(), dir, PullOptions{Force: true}); err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(readFile(t, filepath.Join(dir, "pronunciations.json")), mine) {
		t.Error("--force didn't overwrite the file")
	}
}

func TestPullOutsideGit(t *testing.T) {
	fake, c, dir := pushed(t)
	editInAcademy(t, fake, "pronunciations.json", []byte(`{}`+"\n"))
	_, err := c.Pull(context.Background(), dir, PullOptions{})
	var lc *LocalChangesError
	if !errors.As(err, &lc) {
		t.Fatalf("err = %v", err)
	}
	// A temporary directory may sit inside a repository on some machines;
	// either way the file is protected.
	if !lc.NotInGit && len(lc.Paths) != 1 {
		t.Errorf("paths = %v", lc.Paths)
	}
}

func TestPackageRoot(t *testing.T) {
	base := t.TempDir()
	cases := []struct {
		dir, coursePath, want string
		ok                    bool
	}{
		{filepath.Join(base, "c"), ".", filepath.Join(base, "c"), true},
		{filepath.Join(base, "repo", "courses", "c"), "courses/c", filepath.Join(base, "repo"), true},
		{filepath.Join(base, "elsewhere"), "courses/c", "", false},
		{filepath.Join(base, "c"), "../c", "", false},
	}
	for _, tc := range cases {
		got, err := packageRoot(tc.dir, tc.coursePath)
		if (err == nil) != tc.ok || got != tc.want {
			t.Errorf("packageRoot(%s, %s) = %q, %v", tc.dir, tc.coursePath, got, err)
		}
	}
}

func TestCheckPackagePath(t *testing.T) {
	for _, p := range []string{"course.json", "sections/01/deck/index.html", "theme/a b.css"} {
		if err := checkPackagePath(p); err != nil {
			t.Errorf("%s: %v", p, err)
		}
	}
	for _, p := range []string{"", ".", "..", "../x", "a/../../x", "/etc/passwd", `a\b`, "a//b", "./a", "C:/x"} {
		if err := checkPackagePath(p); err == nil {
			t.Errorf("%q was accepted", p)
		}
	}
}
