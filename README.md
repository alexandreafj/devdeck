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
tab next · 1-4 jump · ↑/↓ move · enter open · r refresh · R all · q quit
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
| `tab` / `shift+tab` | Next / previous widget |
| `1`–`4` | Jump to widget |
| `↑` / `↓` (or `k` / `j`) | Move selection within a widget |
| `←` / `→` (or `h` / `l`) | Switch tab inside the GitHub widget |
| `enter` | Open the selected PR in your browser |
| `r` / `R` | Refresh focused widget / all widgets |
| `q` / `ctrl+c` | Quit |

## Configuration

DevDeck runs with sensible defaults and needs no config. To customise, copy
[`config.example.yml`](./config.example.yml) to:

- **macOS:** `~/Library/Application Support/devdeck/config.yml`
- **Linux:** `~/.config/devdeck/config.yml`

You can set the visible widgets, their titles, and which PR tabs to show.

## Development

Built test-first (TDD); see [`AGENTS.md`](./AGENTS.md) for the full contributor
guide.

```sh
make ci        # vet + lint + race tests + 80% coverage gate (run before pushing)
make test      # quick unit tests
make cover     # coverage report (HTML)
```

CI runs the same checks on every PR and posts a coverage comment; merges are
blocked if coverage drops below **80%**.

## Roadmap

- v0.1 — GitHub PRs (this release)
- Next — Google Calendar, CI status, and `command`-type custom widgets

## License

See [LICENSE](./LICENSE).
