package academyos

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// PronunciationsFile is the name of a pronunciations file, at a course's
// root for the course and at a repository's root for the academy.
const PronunciationsFile = "pronunciations.json"

// Pronunciations maps a term to what is said for it in each language.
type Pronunciations map[string]map[string]string

type pronunciationsBody struct {
	Pronunciations Pronunciations `json:"pronunciations"`
}

func pronunciationsPath(org string) string {
	return "/api/v1/organizations/" + url.PathEscape(org) + "/pronunciations"
}

// GetPronunciations returns the academy's pronunciations.
func (c *Client) GetPronunciations(ctx context.Context, org string) (Pronunciations, error) {
	var body pronunciationsBody
	if err := c.doJSON(ctx, http.MethodGet, pronunciationsPath(org), nil, &body); err != nil {
		return nil, err
	}
	if body.Pronunciations == nil {
		body.Pronunciations = Pronunciations{}
	}
	return body.Pronunciations, nil
}

// PutPronunciations replaces the academy's pronunciations.
func (c *Client) PutPronunciations(ctx context.Context, org string, p Pronunciations) (Pronunciations, error) {
	var body pronunciationsBody
	if err := c.doJSON(ctx, http.MethodPut, pronunciationsPath(org), pronunciationsBody{Pronunciations: p}, &body); err != nil {
		return nil, err
	}
	return body.Pronunciations, nil
}

// PronunciationChange is one term in one language that a push changes.
type PronunciationChange struct {
	Term string `json:"term"`
	Lang string `json:"lang"`
	// From is what the academy says now; empty when added.
	From string `json:"from,omitempty"`
	// To is what it will say; empty when removed.
	To string `json:"to,omitempty"`
}

// PronunciationsResult reports a push of an academy's pronunciations.
type PronunciationsResult struct {
	Org     string                `json:"org"`
	DryRun  bool                  `json:"dry_run,omitempty"`
	Count   int                   `json:"count"`
	Added   []PronunciationChange `json:"added"`
	Changed []PronunciationChange `json:"changed"`
	Removed []PronunciationChange `json:"removed"`
}

// PronunciationsOptions change a push of an academy's pronunciations.
type PronunciationsOptions struct {
	// Org is the academy; empty means the API key's academy.
	Org string
	// DryRun reports what would change without changing anything.
	DryRun bool
}

// ReadPronunciations reads a pronunciations file, or the one in a
// directory.
func ReadPronunciations(path string) (Pronunciations, error) {
	if fi, err := os.Stat(path); err == nil && fi.IsDir() {
		path = filepath.Join(path, PronunciationsFile)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var p Pronunciations
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	if p == nil {
		return nil, fmt.Errorf("%s: expected an object of terms", path)
	}
	return p, nil
}

// PushPronunciations replaces the academy's pronunciations with the file at
// path, or the pronunciations.json in the directory at path, and reports
// what changed. The server validates the whole set and refuses it if any
// entry is invalid.
func (c *Client) PushPronunciations(ctx context.Context, path string, opts PronunciationsOptions) (*PronunciationsResult, error) {
	local, err := ReadPronunciations(path)
	if err != nil {
		return nil, err
	}
	org, err := c.ResolveOrg(ctx, opts.Org)
	if err != nil {
		return nil, err
	}
	current, err := c.GetPronunciations(ctx, org)
	if err != nil {
		return nil, err
	}
	res := DiffPronunciations(current, local)
	res.Org, res.DryRun = org, opts.DryRun
	if opts.DryRun {
		res.Count = countPronunciations(local)
		return res, nil
	}
	saved, err := c.PutPronunciations(ctx, org, local)
	if err != nil {
		return nil, err
	}
	res.Count = countPronunciations(saved)
	return res, nil
}

// DiffPronunciations lists the terms and languages that replacing from
// with to adds, changes and removes. Terms compare in any letter case and with surrounding
// space trimmed, as the server stores them.
func DiffPronunciations(from, to Pronunciations) *PronunciationsResult {
	type key struct{ term, lang string }
	flat := func(p Pronunciations) map[key]PronunciationChange {
		m := map[key]PronunciationChange{}
		for term, byLang := range p {
			t := strings.TrimSpace(term)
			for lang, say := range byLang {
				m[key{strings.ToLower(t), lang}] = PronunciationChange{Term: t, Lang: lang, To: strings.TrimSpace(say)}
			}
		}
		return m
	}
	a, b := flat(from), flat(to)
	res := &PronunciationsResult{Added: []PronunciationChange{}, Changed: []PronunciationChange{}, Removed: []PronunciationChange{}}
	for k, nb := range b {
		na, ok := a[k]
		switch {
		case !ok:
			res.Added = append(res.Added, nb)
		case na.To != nb.To || na.Term != nb.Term:
			res.Changed = append(res.Changed, PronunciationChange{Term: nb.Term, Lang: nb.Lang, From: na.To, To: nb.To})
		}
	}
	for k, na := range a {
		if _, ok := b[k]; !ok {
			res.Removed = append(res.Removed, PronunciationChange{Term: na.Term, Lang: na.Lang, From: na.To})
		}
	}
	for _, list := range [][]PronunciationChange{res.Added, res.Changed, res.Removed} {
		sort.Slice(list, func(i, j int) bool {
			if !strings.EqualFold(list[i].Term, list[j].Term) {
				return strings.ToLower(list[i].Term) < strings.ToLower(list[j].Term)
			}
			return list[i].Lang < list[j].Lang
		})
	}
	return res
}

func countPronunciations(p Pronunciations) int {
	n := 0
	for _, byLang := range p {
		n += len(byLang)
	}
	return n
}
