# DevDeck

A terminal dashboard for the things waiting on you — **PRs, reviews, meetings,
and work signals** — in one place.

DevDeck is a [Bubble Tea](https://github.com/charmbracelet/bubbletea) TUI with a
pluggable, left-to-right widget layout (max 4 widgets). **v0.1** ships the
GitHub PR widget: your authored, review-requested, and assigned pull requests
across **all** your repositories, using your existing `gh` login.

```text
┌──────────────── PRs ─────────────────┐
│ Authored · Review requested · Assigned│
│                                        │
│ > acme/api                             │
│   Fix Smart Bid bit mask               │
│   opened 1d ago by bruno · review req. │
│                                        │
│   acme/dashboard                       │
│   Add AB test definitions page         │
│   opened 2w ago by ana · review req.   │
└────────────────────────────────────────┘
tab section · shift+tab widget · 1-4 jump · ↑/↓ move · / filter · enter open · r refresh · q quit
```

## Requirements

- **Go 1.26+** (to build/install)
- **[GitHub CLI](https://cli.github.com/) (`gh`)**, authenticated:
  `gh auth login`

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

Run `devdeck`. The GitHub widget loads automatically.

| Key | Action |
|---|---|
| `tab` | Cycle sections (Authored / Review requested / Assigned) |
| `shift+tab` | Switch between widgets |
| `1`–`4` | Jump to a widget |
| `↑` / `↓` (or `k` / `j`) | Move the selection |
| `←` / `→` (or `h` / `l`) | Switch section (same as `tab`) |
| `/` | Filter the list (fuzzy) — type to filter, `⌫` to narrow, `esc` to clear |
| `enter` | Open the selected PR in your browser |
| `r` / `R` | Refresh focused widget / all widgets |
| `q` / `ctrl+c` | Quit |

## Configuration

DevDeck runs with sensible defaults and needs no config. To customise, copy
[`config.example.yml`](./config.example.yml) to:

- **macOS:** `~/Library/Application Support/devdeck/config.yml`
- **Linux:** `~/.config/devdeck/config.yml`

You can set the visible widgets, their titles, and which PR sections to show.
The list also **auto-refreshes** on a timer (default every 60s; set per widget
with `refresh:` — e.g. `30s`, `5m`, or omit/`0` to disable), so PRs you've
already handled drop off without pressing `r`.

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
`github.PRProvider`), so the suite never invokes `gh` or the network.

```sh
make test        # quick unit tests (go test ./...)
make test-race   # race detector
make cover       # coverage report (HTML)

go test ./internal/github/... -v -race           # one package, verbose
go test ./internal/ui/... -run TestDashboard -v   # a single test by name
```

| Metric | Count |
|---|---|
| Test files | 14 |
| Test functions | 77 |
| Language | Go |

## License

See [LICENSE](./LICENSE).
