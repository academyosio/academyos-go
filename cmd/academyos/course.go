package main

import (
	"errors"
	"fmt"
	"io"

	"github.com/academyosio/course-spec/validate"
	"github.com/spf13/cobra"

	"github.com/academyosio/academyos-go/academyos"
)

func newCourseCommand(g *globals) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "course",
		Short: "Work with course-spec courses",
	}
	cmd.AddCommand(newValidateCommand(g), newPushCommand(g), newPullCommand(g), newPublishCommand(g))
	return cmd
}

func newValidateCommand(g *globals) *cobra.Command {
	var opts validate.Options
	cmd := &cobra.Command{
		Use:   "validate <dir>",
		Short: "Check a course with course-spec; exits 2 on errors",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			fs := validate.Course(args[0], opts)
			if fs == nil {
				fs = []validate.Finding{}
			}
			errs, warns := validate.Counts(fs)
			out := cmd.OutOrStdout()
			if g.json() {
				if err := writeJSON(out, map[string]any{"errors": errs, "warnings": warns, "findings": fs}); err != nil {
					return err
				}
			} else {
				printFindings(out, fs)
				fmt.Fprintf(out, "%d errors, %d warnings\n", errs, warns)
			}
			if errs > 0 {
				return &academyos.ValidationError{Findings: fs}
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&opts.StrictNarration, "strict-narration", false, "warn about characters TTS reads badly")
	return cmd
}

func newPushCommand(g *globals) *cobra.Command {
	var opts academyos.PushOptions
	cmd := &cobra.Command{
		Use:   "push <dir>",
		Short: "Validate a course and upload it with its decks, then apply it",
		Long: "push validates the course, uploads only the files the academy lacks, and applies the course in one step. " +
			"It records the academy's IDs and revision in the course's published.json; commit that file. " +
			"A push from a copy older than the academy's is refused: pull first, or pass --force.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := g.client()
			if err != nil {
				return err
			}
			opts.Org = g.org
			errOut := cmd.ErrOrStderr()
			if !g.json() {
				opts.Progress = func(msg string) { fmt.Fprintln(errOut, msg) }
			}
			res, err := c.Push(cmd.Context(), args[0], opts)
			out := cmd.OutOrStdout()
			if err != nil {
				reportPushError(out, g, err)
				return err
			}
			if g.json() {
				return writeJSON(out, res)
			}
			printFindings(out, res.Warnings)
			printPush(out, c.BaseURL(), res)
			return nil
		},
	}
	cmd.Flags().BoolVar(&opts.DryRun, "dry-run", false, "report what would change without changing anything")
	cmd.Flags().BoolVar(&opts.Prune, "prune", false, "remove sections and lessons no longer in the course, except lessons learners have used")
	cmd.Flags().BoolVar(&opts.Force, "force", false, "overwrite a newer copy in the academy")
	return cmd
}

// reportPushError prints validation findings and conflicts in detail.
func reportPushError(w io.Writer, g *globals, err error) {
	var ve *academyos.ValidationError
	var ae *academyos.APIError
	switch {
	case errors.As(err, &ve):
		if g.json() {
			if werr := writeJSON(w, map[string]any{"findings": ve.Findings}); werr != nil {
				fmt.Fprintln(w, werr)
			}
			return
		}
		printFindings(w, ve.Findings)
	case errors.As(err, &ae):
		if g.json() {
			if werr := writeJSON(w, ae); werr != nil {
				fmt.Fprintln(w, werr)
			}
			return
		}
		printFindings(w, ae.Findings)
		for _, p := range ae.Changed {
			fmt.Fprintln(w, "  differs in the academy:", p)
		}
	}
}

func printPush(w io.Writer, base string, r *academyos.PushResult) {
	resp := r.Response
	verb := "applied"
	if resp.DryRun {
		verb = "would apply (dry run)"
	}
	fmt.Fprintf(w, "%s %s to %s in %s, revision %d\n", verb, r.CourseID, base, r.Org, resp.Revision)
	fmt.Fprintf(w, "  files     %d in the package, %d uploaded\n", r.Files, r.Uploaded)
	if res := resp.Result; res != nil {
		for _, row := range []struct {
			name string
			c    academyos.Counts
		}{{"sections", res.Sections}, {"lessons", res.Lessons}, {"quizzes", res.Quizzes}} {
			fmt.Fprintf(w, "  %-9s %d created, %d updated, %d unchanged, %d kept, %d removed\n",
				row.name, row.c.Created, row.c.Updated, row.c.Unchanged, row.c.Kept, row.c.Removed)
		}
	}
	if resp.Applied {
		fmt.Fprintln(w, "recorded the academy's IDs in published.json")
	}
}

func newPullCommand(g *globals) *cobra.Command {
	var opts academyos.PullOptions
	cmd := &cobra.Command{
		Use:   "pull <dir>",
		Short: "Write the academy's copy of a course into its directory",
		Long: "pull downloads the files the academy has for the course in <dir> that differ here, writes them in place, " +
			"and records the academy's revision in published.json. Files only here are left alone. " +
			"It refuses to overwrite files with uncommitted changes, or, outside git, any differing file, unless --force is given. " +
			"Commit first, then review what changed with git diff. " +
			"Pass --course to pull a course into an empty directory; the directory is the course's, so a course kept at " +
			"courses/<id> in its repository is pulled into repo/courses/<id>.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := g.client()
			if err != nil {
				return err
			}
			opts.Org = g.org
			errOut := cmd.ErrOrStderr()
			if !g.json() {
				opts.Progress = func(msg string) { fmt.Fprintln(errOut, msg) }
			}
			res, err := c.Pull(cmd.Context(), args[0], opts)
			out := cmd.OutOrStdout()
			if err != nil {
				var lc *academyos.LocalChangesError
				if errors.As(err, &lc) {
					if g.json() {
						if werr := writeJSON(out, map[string]any{"local_changes": lc.Paths, "not_in_git": lc.NotInGit}); werr != nil {
							return errors.Join(err, werr)
						}
					} else {
						for _, p := range lc.Paths {
							fmt.Fprintln(out, "  changed here:", p)
						}
					}
				}
				return err
			}
			if g.json() {
				return writeJSON(out, res)
			}
			printPull(out, c.BaseURL(), res)
			return nil
		},
	}
	cmd.Flags().StringVar(&opts.CourseID, "course", "", "the course's ID, to pull into an empty directory")
	cmd.Flags().BoolVar(&opts.DryRun, "dry-run", false, "list what would change without writing anything")
	cmd.Flags().BoolVar(&opts.Force, "force", false, "overwrite local changes")
	return cmd
}

func printPull(w io.Writer, base string, r *academyos.PullResult) {
	verb := "pulled"
	if r.DryRun {
		verb = "would pull (dry run)"
	}
	fmt.Fprintf(w, "%s %s from %s in %s, revision %d\n", verb, r.CourseID, base, r.Org, r.Revision)
	fmt.Fprintf(w, "  files  %d in the academy: %d added, %d updated, %d unchanged\n", r.Files, len(r.Added), len(r.Updated), r.Unchanged)
	for _, p := range r.Added {
		fmt.Fprintln(w, "  added  ", p)
	}
	for _, p := range r.Updated {
		fmt.Fprintln(w, "  updated", p)
	}
	if !r.DryRun {
		fmt.Fprintln(w, "recorded the academy's revision in published.json")
	}
}

func newPublishCommand(g *globals) *cobra.Command {
	return &cobra.Command{
		Use:   "publish <dir|course-id>",
		Short: "Publish a course",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := g.client()
			if err != nil {
				return err
			}
			org, id, err := c.PublishCourse(cmd.Context(), args[0], g.org)
			if err != nil {
				return err
			}
			if g.json() {
				return writeJSON(cmd.OutOrStdout(), map[string]string{"org": org, "course": id, "status": "published"})
			}
			fmt.Fprintf(cmd.OutOrStdout(), "published %s in %s\n", id, org)
			return nil
		},
	}
}
