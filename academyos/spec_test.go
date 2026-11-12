package academyos

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"strings"
	"sync"
	"testing"
)

// specFile is the public API's OpenAPI document, published with the docs.
const specFile = "../openapi/academyos.json"

// specOperation is one operation in the OpenAPI document.
type specOperation struct {
	method string
	path   *regexp.Regexp
}

// loadSpecOperations returns the operations in specFile, with each path
// template as a pattern that matches a request path.
func loadSpecOperations(t *testing.T) []specOperation {
	t.Helper()
	b, err := os.ReadFile(specFile)
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Paths map[string]map[string]json.RawMessage `json:"paths"`
	}
	if err := json.Unmarshal(b, &doc); err != nil {
		t.Fatalf("read %s: %v", specFile, err)
	}
	param := regexp.MustCompile(`\\\{[^}]+\\\}`)
	methods := map[string]bool{"get": true, "put": true, "post": true, "patch": true, "delete": true, "head": true}
	var ops []specOperation
	for path, item := range doc.Paths {
		rx := regexp.MustCompile("^" + param.ReplaceAllString(regexp.QuoteMeta(path), `[^/]+`) + "$")
		for method := range item {
			if methods[method] {
				ops = append(ops, specOperation{method: strings.ToUpper(method), path: rx})
			}
		}
	}
	if len(ops) == 0 {
		t.Fatalf("%s has no operations", specFile)
	}
	return ops
}

// TestClientCallsAreInTheSpec checks that every request the client makes is
// an operation in the public API's OpenAPI document, so the SDK only uses
// what the API documents for API keys.
func TestClientCallsAreInTheSpec(t *testing.T) {
	ops := loadSpecOperations(t)
	var mu sync.Mutex
	seen := map[string]bool{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		seen[r.Method+" "+r.URL.EscapedPath()] = true
		mu.Unlock()
		for _, op := range ops {
			if op.method == r.Method && op.path.MatchString(r.URL.EscapedPath()) {
				w.Header().Set("Content-Type", "application/json")
				if _, err := w.Write([]byte("{}")); err != nil {
					t.Error(err)
				}
				return
			}
		}
		t.Errorf("%s %s is not in %s", r.Method, r.URL.EscapedPath(), specFile)
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(srv.Close)

	c, err := New(Options{BaseURL: srv.URL, APIKey: testKey})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	const org, course, sum = "acme", "building-web-apis", "0123abcd"
	calls := map[string]func() error{
		"Me": func() error { _, err := c.Me(ctx); return err },
		"GetPackage": func() error {
			_, err := c.GetPackage(ctx, org, course)
			return err
		},
		"PutPackage": func() error {
			_, err := c.PutPackage(ctx, org, course, &PutPackageRequest{})
			return err
		},
		"PutFile": func() error { return c.PutFile(ctx, org, sum, []byte("x")) },
		"GetFile": func() error {
			_, err := c.GetFile(ctx, org, sum)
			return err
		},
		"Publish": func() error { return c.Publish(ctx, org, course) },
		"GetPronunciations": func() error {
			_, err := c.GetPronunciations(ctx, org)
			return err
		},
		"PutPronunciations": func() error {
			_, err := c.PutPronunciations(ctx, org, Pronunciations{})
			return err
		},
	}
	for name, call := range calls {
		if err := call(); err != nil {
			t.Errorf("%s: %v", name, err)
		}
	}
	if len(seen) != len(calls) {
		t.Errorf("made %d distinct requests for %d calls: %v", len(seen), len(calls), seen)
	}
}
