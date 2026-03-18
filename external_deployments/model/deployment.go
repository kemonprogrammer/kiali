package model

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type Deployment struct {
	// deployment
	ID          int64     `json:"id"`
	SHA         string    `json:"sha"`
	CreatedAt   time.Time `json:"created_at,omitempty"`
	UpdatedAt   time.Time `json:"updated_at,omitempty"`
	SucceededAt time.Time `json:"succeeded_at,omitempty"`

	// commits
	ComparisonURL string    `json:"comparison_url"`
	Added         []*Commit `json:"added"`
	Removed       []*Commit `json:"removed"`
}

// shadowDeployment struct to print zero timestamps as "" in JSON
type shadowDeployment struct {
	// deployment
	ID          int64      `json:"id"`
	SHA         string     `json:"sha"`
	CreatedAt   *time.Time `json:"created_at,omitempty"`
	UpdatedAt   *time.Time `json:"updated_at,omitempty"`
	SucceededAt *time.Time `json:"succeeded_at,omitempty"`

	// commits
	ComparisonURL string    `json:"comparison_url"`
	Added         []*Commit `json:"added"`
	Removed       []*Commit `json:"removed"`
}

func (d Deployment) String() string {
	var sb strings.Builder
	for _, commit := range d.Added {
		sb.WriteString(fmt.Sprintf(" + %s\n", commit.Title))
	}
	for _, commit := range d.Removed {
		sb.WriteString(fmt.Sprintf(" - %s\n", commit.Title))
	}

	return fmt.Sprintf(
		"Deployment(\n id: %d,\n sha: %q,\n created: %v,\n updated: %v,\n succeeded: %v,\n comparison URL: %s,\n commits: \n%s)\n",
		d.ID,
		d.SHA,
		formatTime(d.CreatedAt),
		formatTime(d.UpdatedAt),
		formatTime(d.SucceededAt),
		d.ComparisonURL,
		sb.String(),
	)
}

func formatTime(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return &t
}

func (d Deployment) MarshalJSON() ([]byte, error) {
	sd := shadowDeployment{
		ID:            d.ID,
		SHA:           d.SHA,
		CreatedAt:     formatTime(d.CreatedAt),
		UpdatedAt:     formatTime(d.UpdatedAt),
		SucceededAt:   formatTime(d.SucceededAt),
		ComparisonURL: d.ComparisonURL,
		Added:         d.Added,
		Removed:       d.Removed,
	}

	// Define an alias to avoid infinite recursion during marshaling
	type Alias shadowDeployment
	return json.Marshal(&struct {
		Alias
	}{
		Alias: (Alias)(sd),
	})
}

type Commit struct {
	SHA   string `json:"sha"`
	Title string `json:"title"`
	URL   string `json:"url"`
}

func (c Commit) String() string {
	return fmt.Sprintf(
		"Commit(sha: %q, title: %s, url: %s",
		c.SHA,
		c.Title,
		c.URL,
	)
}
