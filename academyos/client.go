// Package academyos is a client for the AcademyOS API: course packages,
// files, publishing, and the caller's identity.
package academyos

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"runtime/debug"
	"strings"
	"time"

	"github.com/academyosio/course-spec/validate"
)

// DefaultBaseURL is the AcademyOS service used unless another is given.
const DefaultBaseURL = "https://app.academyos.io"

// Options configure a Client.
type Options struct {
	// BaseURL is the server, such as https://app.academyos.io or
	// http://localhost:8080. Empty means DefaultBaseURL.
	BaseURL string
	// APIKey is an AcademyOS API key, aos_live_….
	APIKey string
	// HTTPClient sends requests; nil means a client with a timeout.
	HTTPClient *http.Client
	// UserAgent identifies the caller; empty means academyos-go/<version>.
	UserAgent string
}

// Client calls the AcademyOS API. It is safe for concurrent use.
type Client struct {
	base      *url.URL
	key       string
	http      *http.Client
	userAgent string
}

// New returns a Client. It refuses to send a key over plain HTTP except to
// localhost and loopback addresses.
func New(opts Options) (*Client, error) {
	raw := strings.TrimRight(opts.BaseURL, "/")
	if raw == "" {
		raw = DefaultBaseURL
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || (u.Scheme != "https" && u.Scheme != "http") {
		return nil, fmt.Errorf("the API URL %q must be an http or https URL", opts.BaseURL)
	}
	if u.Scheme == "http" && !isLoopback(u.Hostname()) {
		return nil, fmt.Errorf("the API URL %q must use https; plain http is allowed only for localhost", raw)
	}
	if opts.APIKey == "" {
		return nil, errors.New("an API key is required: set ACADEMYOS_API_KEY")
	}
	hc := opts.HTTPClient
	if hc == nil {
		hc = &http.Client{Timeout: 2 * time.Minute}
	}
	ua := opts.UserAgent
	if ua == "" {
		ua = "academyos-go/" + Version()
	}
	return &Client{base: u, key: opts.APIKey, http: hc, userAgent: ua}, nil
}

// BaseURL returns the server the client talks to.
func (c *Client) BaseURL() string { return c.base.String() }

func isLoopback(host string) bool {
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// Version returns this module's version, or "dev" in a local build.
func Version() string {
	if bi, ok := debug.ReadBuildInfo(); ok {
		for _, d := range bi.Deps {
			if d.Path == "github.com/academyosio/academyos-go" && d.Version != "" {
				return d.Version
			}
		}
		if bi.Main.Path == "github.com/academyosio/academyos-go" && bi.Main.Version != "" && bi.Main.Version != "(devel)" {
			return bi.Main.Version
		}
	}
	return "dev"
}

// APIError is a non-2xx response.
type APIError struct {
	Status int
	// Code is a machine-readable reason, such as stale_revision,
	// course_exists, invalid_course, or unauthorized.
	Code    string
	Message string
	// Findings are the server's validation findings.
	Findings []validate.Finding
	// Changed are, for stale_revision, paths that differ from the server's copy.
	Changed []string
}

func (e *APIError) Error() string {
	msg := e.Message
	if msg == "" {
		msg = http.StatusText(e.Status)
	}
	if e.Code != "" {
		return fmt.Sprintf("%s (%s, HTTP %d)", msg, e.Code, e.Status)
	}
	return fmt.Sprintf("%s (HTTP %d)", msg, e.Status)
}

// errorBody covers both the package routes' errors and the API's problem
// format.
type errorBody struct {
	Code     string             `json:"code"`
	Message  string             `json:"message"`
	Findings []validate.Finding `json:"findings"`
	Changed  []string           `json:"changed"`
	Title    string             `json:"title"`
	Detail   string             `json:"detail"`
}

func (c *Client) do(ctx context.Context, method, path string, body io.Reader, contentType string, out any) error {
	u := *c.base
	u.Path = strings.TrimRight(c.base.Path, "/") + path
	req, err := http.NewRequestWithContext(ctx, method, u.String(), body)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.key)
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "application/json")
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("%s %s: %w", method, path, err)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 64<<20))
	if cerr := resp.Body.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return fmt.Errorf("%s %s: read response: %w", method, path, err)
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		e := &APIError{Status: resp.StatusCode}
		var eb errorBody
		if json.Unmarshal(data, &eb) == nil {
			e.Code, e.Message, e.Findings, e.Changed = eb.Code, eb.Message, eb.Findings, eb.Changed
			if e.Message == "" {
				e.Message = firstNonEmpty(eb.Detail, eb.Title)
			}
		}
		if e.Code == "" && (resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden) {
			e.Code = "unauthorized"
		}
		return e
	}
	switch v := out.(type) {
	case nil:
		return nil
	case *[]byte:
		*v = data
		return nil
	default:
		if err := json.Unmarshal(data, out); err != nil {
			return fmt.Errorf("%s %s: decode response: %w", method, path, err)
		}
		return nil
	}
}

func (c *Client) doJSON(ctx context.Context, method, path string, in, out any) error {
	var body io.Reader
	ct := ""
	if in != nil {
		data, err := json.Marshal(in)
		if err != nil {
			return err
		}
		body, ct = bytes.NewReader(data), "application/json"
	}
	return c.do(ctx, method, path, body, ct, out)
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
