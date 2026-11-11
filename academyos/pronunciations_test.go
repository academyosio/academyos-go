package academyos

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestDiffPronunciations(t *testing.T) {
	from := Pronunciations{
		"Lamborghini": {"en-US": "lam-bor-ghee-nee"},
		"Enzo":        {"en-US": "en-tso"},
		"Miura":       {"en-US": "mee-oo-rah", "it-IT": "miura"},
	}
	to := Pronunciations{
		" lamborghini ": {"en-US": "lam-bor-ghee-nee"}, // case differs: changed
		"Miura":         {"en-US": "meeoora", "it-IT": "miura"},
		"Countach":      {"en-US": "koontash"},
	}
	got := DiffPronunciations(from, to)
	want := &PronunciationsResult{
		Added: []PronunciationChange{{Term: "Countach", Lang: "en-US", To: "koontash"}},
		Changed: []PronunciationChange{
			{Term: "lamborghini", Lang: "en-US", From: "lam-bor-ghee-nee", To: "lam-bor-ghee-nee"},
			{Term: "Miura", Lang: "en-US", From: "mee-oo-rah", To: "meeoora"},
		},
		Removed: []PronunciationChange{{Term: "Enzo", Lang: "en-US", From: "en-tso"}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("diff = %+v\nwant  %+v", got, want)
	}
}

func TestPushPronunciations(t *testing.T) {
	fake, srv := newFake(t)
	fake.pronunciations = Pronunciations{"Enzo": {"en-US": "en-tso"}}
	c, err := New(Options{BaseURL: srv.URL, APIKey: testKey})
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, PronunciationsFile),
		[]byte(`{"Lamborghini": {"en-US": "lamborgheeny"}, "Enzo": {"en-US": ""}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()

	// A dry run reports the changes and replaces nothing.
	res, err := c.PushPronunciations(ctx, dir, PronunciationsOptions{DryRun: true})
	if err != nil {
		t.Fatal(err)
	}
	if res.Org != "acme" || !res.DryRun || len(res.Added) != 1 || len(res.Changed) != 1 || res.Count != 2 || fake.pronunciationPuts != 0 {
		t.Fatalf("dry run = %+v, puts %d", res, fake.pronunciationPuts)
	}

	res, err = c.PushPronunciations(ctx, filepath.Join(dir, PronunciationsFile), PronunciationsOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if res.DryRun || res.Count != 2 || fake.pronunciationPuts != 1 || fake.pronunciations["Lamborghini"]["en-US"] != "lamborgheeny" {
		t.Errorf("push = %+v, academy now %v", res, fake.pronunciations)
	}

	if _, err := c.PushPronunciations(ctx, filepath.Join(dir, "missing.json"), PronunciationsOptions{}); err == nil {
		t.Error("expected an error for a missing file")
	}
}
