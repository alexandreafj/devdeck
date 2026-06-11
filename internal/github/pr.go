// Package github fetches pull requests across all of a user's repositories via
// the authenticated `gh` CLI and maps them onto domain.Item values for display.
// All process execution goes through exec.CommandRunner, so the package is fully
// unit-testable without invoking `gh`.
package github

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/alexandreafj/devdeck/internal/domain"
	"github.com/alexandreafj/devdeck/internal/timeutil"
)

// PR is a single pull request as returned by
// `gh search prs --json title,url,repository,author,createdAt`.
type PR struct {
	Title      string     `json:"title"`
	URL        string     `json:"url"`
	Repository Repository `json:"repository"`
	Author     Author     `json:"author"`
	CreatedAt  time.Time  `json:"createdAt"`
}

// Repository is the repo a PR belongs to.
type Repository struct {
	Name          string `json:"name"`
	NameWithOwner string `json:"nameWithOwner"`
}

// Author is the user who opened a PR.
type Author struct {
	Login string `json:"login"`
}

// DecodePRs parses the JSON array emitted by `gh search prs --json …`.
func DecodePRs(data []byte) ([]PR, error) {
	var prs []PR
	if err := json.Unmarshal(data, &prs); err != nil {
		return nil, fmt.Errorf("decode prs: %w", err)
	}
	return prs, nil
}

// ToItem maps a PR onto a generic dashboard Item for the given mode. now is
// supplied so the relative "opened 2w ago" text is deterministic and testable.
func (p PR) ToItem(now time.Time, mode Mode) domain.Item {
	title := p.Repository.NameWithOwner
	if title == "" {
		title = p.Repository.Name
	}
	return domain.Item{
		Title:     title,
		Subtitle:  p.Title,
		Meta:      fmt.Sprintf("opened %s by %s · %s", timeutil.RelativeTime(p.CreatedAt, now), p.Author.Login, mode.meta()),
		URL:       p.URL,
		CreatedAt: p.CreatedAt,
		Source:    "github",
	}
}
