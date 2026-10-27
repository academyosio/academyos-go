package academyos

import (
	"bytes"
	"context"
	"net/http"
	"net/url"
)

// Me is the caller and, when the request used an API key, the key.
type Me struct {
	ID          string       `json:"id"`
	Email       string       `json:"email"`
	Name        string       `json:"name"`
	Memberships []Membership `json:"memberships"`
	APIKey      *APIKey      `json:"api_key,omitempty"`
}

// Membership is the caller's role in one academy.
type Membership struct {
	OrganizationSlug string `json:"organization_slug"`
	OrganizationName string `json:"organization_name"`
	Role             string `json:"role"`
}

// APIKey describes the key a request used.
type APIKey struct {
	Name             string   `json:"name"`
	OrganizationSlug string   `json:"organization_slug"`
	Scopes           []string `json:"scopes"`
	ExpiresAt        string   `json:"expires_at,omitempty"`
}

// Me returns the caller.
func (c *Client) Me(ctx context.Context) (*Me, error) {
	var me Me
	if err := c.doJSON(ctx, http.MethodGet, "/api/v1/users/me", nil, &me); err != nil {
		return nil, err
	}
	return &me, nil
}

// File is one file in a package.
type File struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Size   int64  `json:"size"`
}

// Manifest lists a package's files.
type Manifest struct {
	// CoursePath is the course directory within the package root.
	CoursePath string `json:"coursePath"`
	Files      []File `json:"files"`
}

// Package is a course's package on the server.
type Package struct {
	CourseID string   `json:"course_id"`
	Revision int      `json:"revision"`
	Manifest Manifest `json:"manifest"`
}

// PutPackageRequest proposes a package.
type PutPackageRequest struct {
	Manifest     Manifest `json:"manifest"`
	BaseRevision *int     `json:"base_revision,omitempty"`
	DryRun       bool     `json:"dry_run,omitempty"`
	Prune        bool     `json:"prune,omitempty"`
	Force        bool     `json:"force,omitempty"`
}

// Counts reports what happened to one kind of row.
type Counts struct {
	Created   int `json:"created"`
	Updated   int `json:"updated"`
	Unchanged int `json:"unchanged"`
	Kept      int `json:"kept"`
	Removed   int `json:"removed"`
}

// ApplyResult reports what applying a package did.
type ApplyResult struct {
	CourseUpdated bool   `json:"course_updated"`
	Sections      Counts `json:"sections"`
	Lessons       Counts `json:"lessons"`
	Quizzes       Counts `json:"quizzes"`
}

// PutPackageResponse lists missing files, or what applying did.
type PutPackageResponse struct {
	Applied  bool              `json:"applied"`
	Missing  []string          `json:"missing,omitempty"`
	DryRun   bool              `json:"dry_run,omitempty"`
	Created  bool              `json:"created,omitempty"`
	CourseID string            `json:"course_id,omitempty"`
	Status   string            `json:"status,omitempty"`
	Revision int               `json:"revision,omitempty"`
	Result   *ApplyResult      `json:"result,omitempty"`
	Lessons  map[string]string `json:"lessons,omitempty"`
	Quizzes  map[string]string `json:"quizzes,omitempty"`
}

func coursePath(org, courseID string) string {
	return "/api/v1/organizations/" + url.PathEscape(org) + "/courses/" + url.PathEscape(courseID)
}

// GetPackage returns a course's package manifest and revision.
func (c *Client) GetPackage(ctx context.Context, org, courseID string) (*Package, error) {
	var p Package
	if err := c.doJSON(ctx, http.MethodGet, coursePath(org, courseID)+"/package", nil, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

// PutPackage proposes a package. The response lists missing checksums, or
// reports what applying the package did.
func (c *Client) PutPackage(ctx context.Context, org, courseID string, req *PutPackageRequest) (*PutPackageResponse, error) {
	var r PutPackageResponse
	if err := c.doJSON(ctx, http.MethodPut, coursePath(org, courseID)+"/package", req, &r); err != nil {
		return nil, err
	}
	return &r, nil
}

// PutFile uploads one package file by its SHA-256.
func (c *Client) PutFile(ctx context.Context, org, sha256 string, content []byte) error {
	p := "/api/v1/organizations/" + url.PathEscape(org) + "/files/" + url.PathEscape(sha256)
	return c.do(ctx, http.MethodPut, p, bytes.NewReader(content), "application/octet-stream", nil)
}

// GetFile downloads one package file.
func (c *Client) GetFile(ctx context.Context, org, sha256 string) ([]byte, error) {
	var data []byte
	p := "/api/v1/organizations/" + url.PathEscape(org) + "/files/" + url.PathEscape(sha256)
	if err := c.do(ctx, http.MethodGet, p, nil, "", &data); err != nil {
		return nil, err
	}
	return data, nil
}

// Publish publishes a course.
func (c *Client) Publish(ctx context.Context, org, courseID string) error {
	return c.doJSON(ctx, http.MethodPost, coursePath(org, courseID)+"/publish", nil, nil)
}
