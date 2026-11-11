package main

import (
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/academyosio/academyos-go/academyos"
)

func newPronunciationsCommand(g *globals) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pronunciations",
		Short: "Work with the academy's pronunciations",
		Long: "An academy's pronunciations say terms its courses share, such as a founder's name, in every course. " +
			"Narration applies a course's own pronunciations.json first, then the academy's, then the platform's.",
	}
	cmd.AddCommand(newPronunciationsPushCommand(g), newPronunciationsGetCommand(g))
	return cmd
}

func newPronunciationsPushCommand(g *globals) *cobra.Command {
	var opts academyos.PronunciationsOptions
	cmd := &cobra.Command{
		Use:   "push [file|dir]",
		Short: "Replace the academy's pronunciations with a pronunciations.json",
		Long: "push replaces the academy's pronunciations with the file, or with the pronunciations.json in the directory " +
			"(default: the current directory, usually a course repository's root), and lists the terms added, changed and removed. " +
			"Terms left out of the file are removed from the academy. An empty respelling says the term as written.",
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := "."
			if len(args) == 1 {
				path = args[0]
			}
			c, err := g.client()
			if err != nil {
				return err
			}
			opts.Org = g.org
			res, err := c.PushPronunciations(cmd.Context(), path, opts)
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			if g.json() {
				return writeJSON(out, res)
			}
			return printPronunciations(out, c.BaseURL(), res)
		},
	}
	cmd.Flags().BoolVar(&opts.DryRun, "dry-run", false, "list what would change without changing anything")
	return cmd
}

func newPronunciationsGetCommand(g *globals) *cobra.Command {
	return &cobra.Command{
		Use:   "get",
		Short: "Print the academy's pronunciations as pronunciations.json",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := g.client()
			if err != nil {
				return err
			}
			org, err := c.ResolveOrg(cmd.Context(), g.org)
			if err != nil {
				return err
			}
			p, err := c.GetPronunciations(cmd.Context(), org)
			if err != nil {
				return err
			}
			return writeJSON(cmd.OutOrStdout(), p)
		},
	}
}

func printPronunciations(w io.Writer, base string, r *academyos.PronunciationsResult) error {
	var b strings.Builder
	say := func(s string) string {
		if s == "" {
			return "(as written)"
		}
		return fmt.Sprintf("%q", s)
	}
	for _, c := range r.Added {
		fmt.Fprintf(&b, "  + %s (%s): %s\n", c.Term, c.Lang, say(c.To))
	}
	for _, c := range r.Changed {
		fmt.Fprintf(&b, "  ~ %s (%s): %s -> %s\n", c.Term, c.Lang, say(c.From), say(c.To))
	}
	for _, c := range r.Removed {
		fmt.Fprintf(&b, "  - %s (%s): %s\n", c.Term, c.Lang, say(c.From))
	}
	verb := "replaced"
	if r.DryRun {
		verb = "would replace (dry run)"
	}
	fmt.Fprintf(&b, "%s %s's pronunciations on %s: %d added, %d changed, %d removed, %d in all\n",
		verb, r.Org, base, len(r.Added), len(r.Changed), len(r.Removed), r.Count)
	_, err := io.WriteString(w, b.String())
	return err
}
