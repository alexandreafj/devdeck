# DevDeck

A terminal dashboard for the things waiting on you — **PRs, reviews, meetings,
and work signals** — in one place.

DevDeck is a [Bubble Tea](https://github.com/charmbracelet/bubbletea) TUI with a
pluggable, left-to-right widget layout (max 4 widgets). It ships two widgets:

- **Github Pull Requests** — your authored, review-requested, and assigned pull
  requests across **all** your repositories, using your existing `gh` login.
- **Google Calendar** — this week's meetings grouped by day, with `enter` to
  open the meeting link. Connect once with `devdeck auth google`.

```text
┌────────── Github Pull Requests ───────┐  ┌──────── Google Calendar ────────┐
│ Authored · Review requested · Assigned│  │ Today                            │
│                                        │  │ > 09:00   Standup                │
│ > acme/api                             │  │   14:00   1:1 with Ana           │
│   Fix Smart Bid bit mask               │  │ Tomorrow                         │
│   opened 1d ago by bruno · review req. │  │   All day Company offsite        │
│                                        │  │ Sun Jun 14                       │
│   acme/dashboard                       │  │   10:00   Roadmap review         │
│   Add AB test definitions page         │  │                                  │
│   opened 2w ago by ana · review req.   │  │                                  │
└────────────────────────────────────────┘  └──────────────────────────────────┘
tab section · shift+tab widget · 1-4 jump · ↑/↓ move · / filter · enter open · r refresh · q quit
```

## Requirements

- **Go 1.26+** (to build/install)
- **[GitHub CLI](https://cli.github.com/) (`gh`)**, authenticated:
  `gh auth login` (for the Github Pull Requests widget)
- **A Google Cloud OAuth client** (only for the Google Calendar widget) — see
  [Google Calendar setup](#google-calendar-setup) below.

## Install

```sh
git clone https://github.com/alexandreafj/devdeck.git
cd devdeck
make install
```

`make install` builds DevDeck with `go install` (into `$(go env GOPATH)/bin`)
and, if that directory isn't already on your `PATH`, appends it to `~/.zshrc`
for you (idempotently). Restart your shell (or `source ~/.zshrc`) and run:

```sh
devdeck
```

Remove it any time with `make uninstall`.

> Prefer a local build without touching your shell? `make build` drops the
> binary at `./bin/devdeck`.

## Usage

Run `devdeck`. The Github Pull Requests widget loads automatically; add the
Google Calendar widget via config after connecting your account (see below).

| Key | Action |
|---|---|
| `tab` | Cycle sections (Authored / Review requested / Assigned) |
| `shift+tab` | Switch between widgets |
| `1`–`4` | Jump to a widget |
| `↑` / `↓` (or `k` / `j`) | Move the selection |
| `←` / `→` (or `h` / `l`) | Switch section (same as `tab`) |
| `/` | Filter the list (fuzzy) — type to filter, `⌫` to narrow, `esc` to clear |
| `enter` | Open the selected PR / meeting link in your browser |
| `r` / `R` | Refresh focused widget / all widgets |
| `q` / `ctrl+c` | Quit |

## Configuration

On first run DevDeck **writes a documented starter `config.yml`** you can edit
(see [`config.example.yml`](./config.example.yml) for the same content):

- **macOS:** `~/Library/Application Support/devdeck/config.yml`
- **Linux:** `~/.config/devdeck/config.yml`

Choose your widgets there — the dashboard shows up to 4, left to right. **Add,
remove, reorder, or duplicate** entries under `widgets:` and restart `devdeck`;
each widget authenticates on its own (`gh auth login` for GitHub,
`devdeck auth google` for Calendar). The list also **auto-refreshes** on a timer
(set per widget with `refresh:` — e.g. `30s`, `5m`, or omit/`0` to disable).

## Google Calendar setup

The Google Calendar widget reads your primary calendar with the read-only
`calendar.readonly` scope. Because DevDeck is a desktop app, you supply your own
Google OAuth client (nothing secret is shipped in this repo) — and
`devdeck auth google` **walks you through it**:

```sh
devdeck auth google
```

The guided setup:

1. Opens the Google Cloud console to **enable the Calendar API**, and prints the
   links to configure the **OAuth consent screen** (User type *External*; add your
   own address as a *Test user*) and to **create an OAuth client ID** of type
   **Desktop app** → *Download JSON*.
2. **Auto-detects** the downloaded `client_secret_*.json` in your `~/Downloads`
   (or paste/drag its path) and **copies it into DevDeck's own config folder** as
   `google-credentials.json`:
   - **macOS:** `~/Library/Application Support/devdeck/`
   - **Linux:** `~/.config/devdeck/`

   Because DevDeck keeps its own copy, you can delete the downloaded file
   afterwards without affecting DevDeck.
3. Opens your browser for consent and caches the token next to the credentials
   (`google-token.json`); it refreshes automatically.

Re-running `devdeck auth google` later reuses the saved credentials and just
refreshes consent. Then add the widget to your `config.yml`
(see [`config.example.yml`](./config.example.yml)):

```yaml
widgets:
  - type: google_calendar
    title: "Google Calendar"
    refresh: 5m
```

Run `devdeck` — it shows this week's meetings grouped under **Today**,
**Tomorrow**, and `Mon Jun 15`-style headers. Until you connect, the widget shows
a "Not connected — run `devdeck auth google`" prompt instead of an error.

## Development

Built test-first (TDD); see [`AGENTS.md`](./AGENTS.md) for the full contributor
guide.

```sh
make ci        # vet + lint + race tests + 80% coverage gate (run before pushing)
make test      # quick unit tests
make cover     # coverage report (HTML)
```

CI runs the same checks on every code PR and posts a coverage comment; merges
are blocked if coverage drops below **80%**. Docs-only PRs (`**.md`) skip the
heavy CI and report the required checks green automatically.

## Testing

Built test-first — all I/O sits behind interfaces (`exec.CommandRunner`,
`github.PRProvider`, `gcal.EventProvider`), so the suite never invokes `gh`,
Google, or the network.

```sh
make test        # quick unit tests (go test ./...)
make test-race   # race detector
make cover       # coverage report (HTML)

go test ./internal/github/... -v -race           # one package, verbose
go test ./internal/ui/... -run TestDashboard -v   # a single test by name
```

| Metric | Count |
|---|---|
| Test files | 20 |
| Test functions | 121 |
| Language | Go |

## License

See [LICENSE](./LICENSE).
