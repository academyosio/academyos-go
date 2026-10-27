package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/academyosio/course-spec/validate"
	"github.com/spf13/cobra"

	"github.com/academyosio/academyos-go/academyos"
)

// Exit codes.
const (
	exitFailure    = 1
	exitValidation = 2
	exitStale      = 3
	exitAuth       = 4
)

type globals struct {
	apiURL string
	org    string
	format string
}

func newRootCommand() *cobra.Command {
	g := &globals{}
	root := &cobra.Command{
		Use:           "academyos",
		Short:         "Check, upload, and publish courses to AcademyOS",
		Long:          "academyos works with courses kept as course-spec directories. It talks to " + academyos.DefaultBaseURL + " unless --api-url or ACADEMYOS_API_URL names another server, and reads the API key from ACADEMYOS_API_KEY.",
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(*cobra.Command, []string) error {
			if g.format != "text" && g.format != "json" {
				return fmt.Errorf("--format must be text or json, not %q", g.format)
			}
			return nil
		},
	}
	root.PersistentFlags().StringVar(&g.apiURL, "api-url", "", "AcademyOS server (default $ACADEMYOS_API_URL, else "+academyos.DefaultBaseURL+")")
	root.PersistentFlags().StringVar(&g.org, "org", "", "academy slug (default: the API key's academy)")
	root.PersistentFlags().StringVar(&g.format, "format", "text", "output format: text or json")
	root.AddCommand(newWhoamiCommand(g), newCourseCommand(g), newVersionCommand())
	return root
}

// client builds an API client from the flags and environment.
func (g *globals) client() (*academyos.Client, error) {
	u := g.apiURL
	if u == "" {
		u = os.Getenv("ACADEMYOS_API_URL")
	}
	return academyos.New(academyos.Options{BaseURL: u, APIKey: os.Getenv("ACADEMYOS_API_KEY")})
}

func (g *globals) json() bool { return g.format == "json" }

func writeJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// exitCode maps an error to the command's exit status.
func exitCode(err error) int {
	var ve *academyos.ValidationError
	if errors.As(err, &ve) {
		return exitValidation
	}
	var ae *academyos.APIError
	if errors.As(err, &ae) {
		switch {
		case ae.Code == "invalid_course" || ae.Code == "invalid_manifest":
			return exitValidation
		case ae.Code == "stale_revision" || ae.Code == "course_exists":
			return exitStale
		case ae.Status == http.StatusUnauthorized || ae.Status == http.StatusForbidden:
			return exitAuth
		}
	}
	return exitFailure
}

func printFindings(w io.Writer, fs []validate.Finding) {
	for _, f := range fs {
		fmt.Fprintln(w, f.String())
	}
}

func newVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the version",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, _ []string) {
			fmt.Fprintln(cmd.OutOrStdout(), "academyos", academyos.Version())
		},
	}
}

func newWhoamiCommand(g *globals) *cobra.Command {
	return &cobra.Command{
		Use:   "whoami",
		Short: "Show who the API key acts for",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := g.client()
			if err != nil {
				return err
			}
			me, err := c.Me(cmd.Context())
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			if g.json() {
				return writeJSON(out, me)
			}
			fmt.Fprintf(out, "%s <%s> on %s\n", me.Name, me.Email, c.BaseURL())
			if k := me.APIKey; k != nil {
				fmt.Fprintf(out, "key %q in %s, scopes %v", k.Name, k.OrganizationSlug, k.Scopes)
				if k.ExpiresAt != "" {
					fmt.Fprintf(out, ", expires %s", k.ExpiresAt)
				}
				fmt.Fprintln(out)
			}
			for _, m := range me.Memberships {
				fmt.Fprintf(out, "  %s: %s\n", m.OrganizationSlug, m.Role)
			}
			return nil
		},
	}
}
