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
		if strings.HasPrefix(pkg, "github.com/grokify/academyos") || strings.HasPrefix(pkg, "entgo.io/") || strings.HasPrefix(pkg, "github.com/plexusone/systemforge") {
			t.Errorf("depends on server package %s", pkg)
		}
	}
}
