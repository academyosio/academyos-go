package academyosgo_test

import (
	"os/exec"
	"strings"
	"testing"
)

// TestNoServerDependencies keeps the AcademyOS server out of this module:
// installing the command must not pull in its code or dependencies.
func TestNoServerDependencies(t *testing.T) {
	out, err := exec.Command("go", "list", "-deps", "./...").Output()
	if err != nil {
		t.Fatalf("go list: %v", err)
	}
	for _, pkg := range strings.Fields(string(out)) {
		if inModule(pkg, "github.com/academyosio/academyos") || strings.HasPrefix(pkg, "entgo.io/") || strings.HasPrefix(pkg, "github.com/plexusone/systemforge") {
			t.Errorf("depends on server package %s", pkg)
		}
	}
}

// inModule reports whether pkg is module m or one of its packages, so
// github.com/academyosio/academyos doesn't match academyos-go itself.
func inModule(pkg, m string) bool {
	return pkg == m || strings.HasPrefix(pkg, m+"/")
}
