package github

// Mode is one of the three relationships a pull request can have to the current
// user. The GitHub widget shows each as a switchable tab.
type Mode int

const (
	// Authored is pull requests the user opened.
	Authored Mode = iota
	// ReviewRequested is pull requests waiting on the user's review.
	ReviewRequested
	// Assigned is pull requests assigned to the user.
	Assigned
)

// Modes is the canonical ordered set, used to build the widget's tabs.
var Modes = []Mode{Authored, ReviewRequested, Assigned}

// Label is the human-facing tab title, e.g. "Review requested".
func (m Mode) Label() string {
	switch m {
	case Authored:
		return "Authored"
	case ReviewRequested:
		return "Review requested"
	case Assigned:
		return "Assigned"
	default:
		return "Unknown"
	}
}

// meta is the lowercase relationship phrase used in an Item's Meta line,
// e.g. "opened 2w ago by ana · review requested".
func (m Mode) meta() string {
	switch m {
	case Authored:
		return "authored"
	case ReviewRequested:
		return "review requested"
	case Assigned:
		return "assigned"
	default:
		return "unknown"
	}
}

// searchFlag is the `gh search prs` filter flag for the mode.
func (m Mode) searchFlag() string {
	switch m {
	case Authored:
		return "--author"
	case ReviewRequested:
		return "--review-requested"
	case Assigned:
		return "--assignee"
	default:
		return ""
	}
}
