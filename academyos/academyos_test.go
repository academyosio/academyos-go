package academyos

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	coursespec "github.com/academyosio/course-spec"
)

func copyCourse(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "course")
	if err := os.CopyFS(dir, os.DirFS("../testdata/web-apis")); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestNew(t *testing.T) {
	cases := []struct {
		url string
		ok  bool
	}{
		{"", true},
		{"https://example.com", true},
		{"http://localhost:8080", true},
		{"http://app.localhost:8081", true},
		{"http://localhost.example.com", false},
		{"http://127.0.0.1:8080", true},
		{"http://[::1]:8080", true},
		{"http://example.com", false},
		{"ftp://example.com", false},
		{"not a url", false},
	}
	for _, c := range cases {
		_, err := New(Options{BaseURL: c.url, APIKey: testKey})
		if (err == nil) != c.ok {
			t.Errorf("%q: err = %v", c.url, err)
		}
	}
	if _, err := New(Options{}); err == nil {
		t.Error("expected an error without a key")
	}
	c, err := New(Options{APIKey: testKey})
	if err != nil || c.BaseURL() != DefaultBaseURL {
		t.Errorf("default base URL = %v, %v", c, err)
	}
}

func TestPush(t *testing.T) {
	fake, srv := newFake(t)
	c, err := New(Options{BaseURL: srv.URL, APIKey: testKey})
	if err != nil {
		t.Fatal(err)
	}
	dir := copyCourse(t)
	ctx := context.Background()

	res, err := c.Push(ctx, dir, PushOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Org != "acme" || res.Uploaded != 10 || !res.Response.Applied || res.Response.Revision != 1 {
		t.Fatalf("push = %+v %+v", res, res.Response)
	}
	pub, err := coursespec.ReadPublished(filepath.Join(dir, "published.json"))
	if err != nil {
		t.Fatal(err)
	}
	p := pub.Platforms[PlatformName]
	if p.CourseID != "c-1" || p.Revision != 1 || p.Lessons["resources"] != "l-1" || p.BaseURL != srv.URL {
		t.Errorf("published = %+v", p)
	}

	// The second push uploads nothing and sends the recorded revision.
	res, err = c.Push(ctx, dir, PushOptions{})
	if err != nil {
		t.Fatal(err)
	}
	last := fake.puts[len(fake.puts)-1]
	if res.Uploaded != 0 || last.BaseRevision == nil || *last.BaseRevision != 1 {
		t.Errorf("second push uploaded %d, base %v", res.Uploaded, last.BaseRevision)
	}
}

func TestPushDryRunWritesNothing(t *testing.T) {
	_, srv := newFake(t)
	c, err := New(Options{BaseURL: srv.URL, APIKey: testKey})
	if err != nil {
		t.Fatal(err)
	}
	dir := copyCourse(t)
	res, err := c.Push(context.Background(), dir, PushOptions{DryRun: true})
	if err != nil || !res.Response.DryRun || res.Response.Applied {
		t.Fatalf("dry run = %+v, %v", res, err)
	}
	if _, err := os.Stat(filepath.Join(dir, "published.json")); !errors.Is(err, os.ErrNotExist) {
		t.Error("a dry run wrote published.json")
	}
}

func TestPushValidationError(t *testing.T) {
	_, srv := newFake(t)
	c, err := New(Options{BaseURL: srv.URL, APIKey: testKey})
	if err != nil {
		t.Fatal(err)
	}
	dir := copyCourse(t)
	if err := os.Remove(filepath.Join(dir, "theme", "deck.css")); err != nil {
		t.Fatal(err)
	}
	_, err = c.Push(context.Background(), dir, PushOptions{})
	var ve *ValidationError
	if !errors.As(err, &ve) || len(ve.Findings) == 0 {
		t.Fatalf("err = %v", err)
	}
}

func TestPushConflict(t *testing.T) {
	fake, srv := newFake(t)
	fake.conflict = "stale_revision"
	c, err := New(Options{BaseURL: srv.URL, APIKey: testKey})
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.Push(context.Background(), copyCourse(t), PushOptions{})
	var ae *APIError
	if !errors.As(err, &ae) || ae.Code != "stale_revision" || ae.Status != 409 || len(ae.Changed) != 1 {
		t.Fatalf("err = %v", err)
	}
}

func TestAuthError(t *testing.T) {
	_, srv := newFake(t)
	c, err := New(Options{BaseURL: srv.URL, APIKey: "aos_live_wrong"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.Me(context.Background())
	var ae *APIError
	if !errors.As(err, &ae) || ae.Code != "unauthorized" || !strings.Contains(ae.Error(), "revoked") {
		t.Fatalf("err = %v", err)
	}
}

func TestOrgMismatch(t *testing.T) {
	_, srv := newFake(t)
	c, err := New(Options{BaseURL: srv.URL, APIKey: testKey})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.Push(context.Background(), copyCourse(t), PushOptions{Org: "other"}); err == nil || !strings.Contains(err.Error(), "belongs to") {
		t.Fatalf("err = %v", err)
	}
}

func TestPublishCourse(t *testing.T) {
	_, srv := newFake(t)
	c, err := New(Options{BaseURL: srv.URL, APIKey: testKey})
	if err != nil {
		t.Fatal(err)
	}
	org, id, err := c.PublishCourse(context.Background(), copyCourse(t), "")
	if err != nil || org != "acme" || id != "building-web-apis" {
		t.Fatalf("publish = %s %s %v", org, id, err)
	}
}
