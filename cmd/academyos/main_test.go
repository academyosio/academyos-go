package main

import (
	"bytes"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/academyosio/academyos-go/academyos"
)

func run(t *testing.T, args ...string) (string, error) {
	t.Helper()
	var out bytes.Buffer
	cmd := newRootCommand()
	cmd.SetOut(&out)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs(args)
	err := cmd.Execute()
	return out.String(), err
}

func TestValidate(t *testing.T) {
	out, err := run(t, "course", "validate", "../../testdata/web-apis")
	if err != nil || !strings.Contains(out, "0 errors, 0 warnings") {
		t.Fatalf("out = %q, err = %v", out, err)
	}
	dir := filepath.Join(t.TempDir(), "c")
	if err := os.CopyFS(dir, os.DirFS("../../testdata/web-apis")); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(dir, "theme", "deck.css")); err != nil {
		t.Fatal(err)
	}
	out, err = run(t, "--format", "json", "course", "validate", dir)
	if exitCode(err) != exitValidation || !strings.Contains(out, `"missing-deck-file"`) {
		t.Fatalf("out = %q, exit %d", out, exitCode(err))
	}
}

func TestExitCodes(t *testing.T) {
	cases := map[int]error{
		exitValidation: &academyos.ValidationError{},
		exitStale:      &academyos.APIError{Status: http.StatusConflict, Code: "stale_revision"},
		exitAuth:       &academyos.APIError{Status: http.StatusUnauthorized, Code: "unauthorized"},
		exitFailure:    errors.New("network down"),
	}
	for want, err := range cases {
		if got := exitCode(err); got != want {
			t.Errorf("exitCode(%v) = %d, want %d", err, got, want)
		}
	}
	if got := exitCode(&academyos.APIError{Status: 422, Code: "invalid_course"}); got != exitValidation {
		t.Errorf("invalid_course exit = %d", got)
	}
}

func TestMissingKey(t *testing.T) {
	t.Setenv("ACADEMYOS_API_KEY", "")
	if _, err := run(t, "whoami"); err == nil || !strings.Contains(err.Error(), "ACADEMYOS_API_KEY") {
		t.Fatalf("err = %v", err)
	}
}

func TestBadFormat(t *testing.T) {
	if _, err := run(t, "--format", "yaml", "version"); err == nil {
		t.Fatal("expected an error for an unknown format")
	}
}
