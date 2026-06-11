package github

import (
	"testing"
	"time"
)

const sampleJSON = `[
  {
    "title": "Fix Smart Bid bit mask",
    "url": "https://github.com/acme/api/pull/12",
    "repository": {"name": "api", "nameWithOwner": "acme/api"},
    "author": {"login": "bruno"},
    "createdAt": "2026-06-10T12:00:00Z"
  },
  {
    "title": "Add AB test definitions page",
    "url": "https://github.com/acme/dashboard/pull/7",
    "repository": {"name": "dashboard", "nameWithOwner": "acme/dashboard"},
    "author": {"login": "ana"},
    "createdAt": "2026-05-28T09:30:00Z"
  }
]`

func TestDecodePRs(t *testing.T) {
	prs, err := DecodePRs([]byte(sampleJSON))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(prs) != 2 {
		t.Fatalf("got %d PRs, want 2", len(prs))
	}
	first := prs[0]
	if first.Title != "Fix Smart Bid bit mask" {
		t.Errorf("Title = %q", first.Title)
	}
	if first.Repository.NameWithOwner != "acme/api" {
		t.Errorf("NameWithOwner = %q", first.Repository.NameWithOwner)
	}
	if first.Author.Login != "bruno" {
		t.Errorf("Author.Login = %q", first.Author.Login)
	}
	want := time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)
	if !first.CreatedAt.Equal(want) {
		t.Errorf("CreatedAt = %v, want %v", first.CreatedAt, want)
	}
}

func TestDecodePRsEmpty(t *testing.T) {
	prs, err := DecodePRs([]byte(`[]`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(prs) != 0 {
		t.Fatalf("got %d PRs, want 0", len(prs))
	}
}

func TestDecodePRsInvalid(t *testing.T) {
	if _, err := DecodePRs([]byte(`this is not json`)); err == nil {
		t.Fatal("expected error decoding invalid JSON, got nil")
	}
}

func TestPRToItem(t *testing.T) {
	now := time.Date(2026, 6, 11, 12, 0, 0, 0, time.UTC)
	pr := PR{
		Title:      "Add AB test definitions page",
		URL:        "https://github.com/acme/dashboard/pull/7",
		Repository: Repository{Name: "dashboard", NameWithOwner: "acme/dashboard"},
		Author:     Author{Login: "ana"},
		CreatedAt:  now.Add(-14 * 24 * time.Hour),
	}

	item := pr.ToItem(now, ReviewRequested)

	if item.Title != "acme/dashboard" {
		t.Errorf("Title = %q, want %q", item.Title, "acme/dashboard")
	}
	if item.Subtitle != "Add AB test definitions page" {
		t.Errorf("Subtitle = %q", item.Subtitle)
	}
	if want := "opened 2w ago by ana · review requested"; item.Meta != want {
		t.Errorf("Meta = %q, want %q", item.Meta, want)
	}
	if item.URL != pr.URL {
		t.Errorf("URL = %q, want %q", item.URL, pr.URL)
	}
	if item.Source != "github" {
		t.Errorf("Source = %q, want %q", item.Source, "github")
	}
	if !item.CreatedAt.Equal(pr.CreatedAt) {
		t.Errorf("CreatedAt = %v, want %v", item.CreatedAt, pr.CreatedAt)
	}
}

func TestPRToItemFallsBackToRepoName(t *testing.T) {
	pr := PR{Repository: Repository{Name: "solo"}}
	item := pr.ToItem(time.Now(), Authored)
	if item.Title != "solo" {
		t.Errorf("Title = %q, want %q (fallback to repo name)", item.Title, "solo")
	}
}
