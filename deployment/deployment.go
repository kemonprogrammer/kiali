package deployment

import (
	"fmt"
	"strings"
	"time"
)

type Deployment struct {
	// deployment
	ID        int64     `json:"id"`
	URL       string    `json:"url"`
	SHA       string    `json:"sha"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// commits
	ComparisonURL string    `json:"comparison_url,omitempty"`
	Added         []*Commit `json:"added,omitempty"`
	Removed       []*Commit `json:"removed,omitempty"`
}

func (d *Deployment) String() string {
	if d == nil {
		return "<nil>"
	}

	var sb strings.Builder
	for _, commit := range d.Added {
		sb.WriteString(fmt.Sprintf(" + %s\n", commit.Title))
	}
	for _, commit := range d.Removed {
		sb.WriteString(fmt.Sprintf(" - %s\n", commit.Title))
	}

	return fmt.Sprintf(
		"Deployment(\n id: %d,\n deployUrl: %s,\n sha: %q,\n created: %v,\n commits: \n%s)\n",
		d.ID,
		d.URL,
		d.SHA,
		d.CreatedAt,
		sb.String(),
	)
}

type Commit struct {
	SHA   string `json:"sha,omitempty"`
	Title string `json:"title,omitempty"`
	URL   string `json:"url,omitempty"`
}

func (c *Commit) String() string {
	if c == nil {
		return "<nil>"
	}
	return fmt.Sprintf(
		"Commit(sha: %q, title: %s, url: %s",
		c.SHA,
		c.Title,
		c.URL,
	)
}
