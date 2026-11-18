package coursepkg

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func copyCourse(t *testing.T, dst string) {
	t.Helper()
	if err := os.CopyFS(dst, os.DirFS("../testdata/web-apis")); err != nil {
		t.Fatal(err)
	}
}

func paths(p *Package) string {
	var out []string
	for _, f := range p.Files {
		out = append(out, f.Path)
	}
	return strings.Join(out, "\n")
}

func TestBuild(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "course")
	copyCourse(t, dir)
	if err := os.WriteFile(filepath.Join(dir, "published.json"), []byte(`{}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("not referenced"), 0o600); err != nil {
		t.Fatal(err)
	}
	p, err := Build(dir)
	if err != nil {
		t.Fatal(err)
	}
	want := strings.Join([]string{
		"course.json",
		"pronunciations.json",
		"sections/01-designing-endpoints/deck/index.html",
		"sections/01-designing-endpoints/deck/voiceover/transcript.json",
		"sections/01-designing-endpoints/quiz.json",
		"sections/01-designing-endpoints/section.json",
		"sections/01-designing-endpoints/text/glossary.md",
		"sections/02-final-assessment/quiz.json",
		"sections/02-final-assessment/section.json",
		"theme/deck.css",
	}, "\n")
	if got := paths(p); got != want {
		t.Errorf("files:\n%s\nwant:\n%s", got, want)
	}
	if p.CoursePath != "." || p.CourseID != "building-web-apis" {
		t.Errorf("course path %q, id %q", p.CoursePath, p.CourseID)
	}
	if f := p.Files[0]; len(f.SHA256) != 64 || f.Size == 0 {
		t.Errorf("file = %+v", f)
	}
}

func TestBuildSharedTheme(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "courses", "web-apis")
	copyCourse(t, dir)
	if err := os.Rename(filepath.Join(dir, "theme"), filepath.Join(root, "courses", "shared")); err != nil {
		t.Fatal(err)
	}
	edit := func(rel, old, new string) {
		p := filepath.Join(dir, rel)
		data, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(strings.Replace(string(data), old, new, 1)), 0o600); err != nil { //nolint:gosec // test files
			t.Fatal(err)
		}
	}
	edit("course.json", `"theme": "theme"`, `"theme": "../shared"`)
	edit("sections/01-designing-endpoints/deck/index.html", `../../../theme/deck.css`, `../../../../shared/deck.css`)
	p, err := Build(dir)
	if err != nil {
		t.Fatal(err)
	}
	if p.CoursePath != "web-apis" || !strings.Contains(paths(p), "shared/deck.css") || !strings.Contains(paths(p), "web-apis/course.json") {
		t.Errorf("course path %q, files:\n%s", p.CoursePath, paths(p))
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestBuildRecordings(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "course")
	copyCourse(t, dir)
	rec := filepath.Join(dir, "sections/01-designing-endpoints/deck/recordings")
	writeFile(t, filepath.Join(rec, "recordings.json"), `{"lessons": [{"lesson": "x", "language": "en-US",
		"audio": "en-US/x.mp3", "captions": "en-US/x.vtt", "durationMs": 1000,
		"slides": [{"id": "x", "startMs": 0, "durationMs": 1000, "narration": "sha256:0"}]}]}`)
	writeFile(t, filepath.Join(rec, "en-US/x.mp3"), "mp3")
	writeFile(t, filepath.Join(rec, "en-US/x.vtt"), "WEBVTT\n")
	writeFile(t, filepath.Join(rec, "en-US/old.mp3"), "not listed")

	p, err := Build(dir)
	if err != nil {
		t.Fatal(err)
	}
	got := paths(p)
	for _, want := range []string{"deck/recordings/recordings.json", "deck/recordings/en-US/x.mp3", "deck/recordings/en-US/x.vtt"} {
		if !strings.Contains(got, "sections/01-designing-endpoints/"+want+"\n") {
			t.Errorf("missing %s in\n%s", want, got)
		}
	}
	if strings.Contains(got, "old.mp3") {
		t.Error("a file the recordings don't list was packaged")
	}

	if err := os.Remove(filepath.Join(rec, "en-US/x.mp3")); err != nil {
		t.Fatal(err)
	}
	if _, err := Build(dir); err == nil {
		t.Error("a missing audio file wasn't reported")
	}
	writeFile(t, filepath.Join(rec, "recordings.json"), `{"lessons": [{"lesson": "x", "language": "en-US",
		"audio": "../../escape.mp3", "durationMs": 1000,
		"slides": [{"id": "x", "startMs": 0, "durationMs": 1000, "narration": "sha256:0"}]}]}`)
	if _, err := Build(dir); err == nil || !strings.Contains(err.Error(), "recordings directory") {
		t.Errorf("a path leaving recordings/ = %v", err)
	}
}
