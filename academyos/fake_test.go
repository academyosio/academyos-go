package academyos

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

// fakeServer implements the package routes in memory.
type fakeServer struct {
	t        *testing.T
	mu       sync.Mutex
	files    map[string][]byte
	revision int
	puts     []PutPackageRequest
	// conflict, when set, answers every package PUT with this code.
	conflict string
}

func newFake(t *testing.T) (*fakeServer, *httptest.Server) {
	f := &fakeServer{t: t, files: map[string][]byte{}}
	srv := httptest.NewServer(f)
	t.Cleanup(srv.Close)
	return f, srv
}

func (f *fakeServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if r.Header.Get("Authorization") != "Bearer "+testKey {
		w.WriteHeader(http.StatusUnauthorized)
		writeJSONBody(f.t, w, map[string]any{"title": "Unauthorized", "detail": "invalid, expired or revoked API key"})
		return
	}
	switch {
	case r.URL.Path == "/api/v1/users/me":
		writeJSONBody(f.t, w, Me{Name: "Test", Email: "t@example.com", APIKey: &APIKey{OrganizationSlug: "acme", Scopes: []string{"courses:write"}}})
	case strings.HasPrefix(r.URL.Path, "/api/v1/organizations/acme/files/"):
		sum := strings.TrimPrefix(r.URL.Path, "/api/v1/organizations/acme/files/")
		data, err := io.ReadAll(r.Body)
		if err != nil {
			f.t.Error(err)
		}
		h := sha256.Sum256(data)
		if hex.EncodeToString(h[:]) != sum {
			w.WriteHeader(http.StatusUnprocessableEntity)
			return
		}
		f.files[sum] = data
		w.WriteHeader(http.StatusNoContent)
	case r.URL.Path == "/api/v1/organizations/acme/courses/building-web-apis/package" && r.Method == http.MethodPut:
		var req PutPackageRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			f.t.Error(err)
		}
		f.puts = append(f.puts, req)
		if f.conflict != "" {
			w.WriteHeader(http.StatusConflict)
			writeJSONBody(f.t, w, map[string]any{"code": f.conflict, "message": "pull first", "changed": []string{"course.json"}})
			return
		}
		var missing []string
		for _, file := range req.Manifest.Files {
			if _, ok := f.files[file.SHA256]; !ok {
				missing = append(missing, file.SHA256)
			}
		}
		if len(missing) > 0 {
			writeJSONBody(f.t, w, PutPackageResponse{Missing: missing})
			return
		}
		if !req.DryRun {
			f.revision++
		}
		writeJSONBody(f.t, w, PutPackageResponse{
			Applied: !req.DryRun, DryRun: req.DryRun, CourseID: "c-1", Revision: f.revision,
			Result:  &ApplyResult{Lessons: Counts{Created: 3}},
			Lessons: map[string]string{"resources": "l-1"}, Quizzes: map[string]string{"designing-endpoints-quiz": "q-1"},
		})
	case strings.HasSuffix(r.URL.Path, "/publish"):
		w.WriteHeader(http.StatusNoContent)
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func writeJSONBody(t *testing.T, w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		t.Error(err)
	}
}

// testKey is a placeholder key the fake server accepts.
const testKey = "aos_live_test" //nolint:gosec // not a credential
