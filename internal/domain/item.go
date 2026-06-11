// Package domain holds the source-agnostic data types shared by every DevDeck
// widget. Keeping these here means a widget (GitHub, Calendar, CI, …) only has
// to map its own data into an Item, and the UI layer renders Items uniformly.
package domain

import "time"

// Item is a single, generic row a widget can display. Each source maps its own
// records onto this shape:
//
//	GitHub PR  -> Title: "org/repo", Subtitle: PR title, Meta: "opened 2w ago by ana · review requested"
//	Calendar   -> Title: "10:00 Standup", Subtitle: location, Meta: "Today · 30m"
//
// URL is opened when the user presses enter on the item. CreatedAt is kept so
// widgets can sort or re-render relative times without re-fetching. Source
// names the originating widget (e.g. "github", "calendar").
type Item struct {
	Title     string
	Subtitle  string
	Meta      string
	URL       string
	CreatedAt time.Time
	Source    string
}
