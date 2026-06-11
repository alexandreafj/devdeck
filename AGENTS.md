# AGENTS.md — DevDeck

Guidance for any agent (or human) working in this repository. Read this before
making changes.

## Overview

**DevDeck** is a terminal dashboard (Bubble Tea TUI) for "the things waiting on
you." It renders a configurable, left-to-right set of **widgets** (max 4). v0.1
ships one widget: **GitHub PRs** — authored / review-requested / assigned PRs
across all your repos, fetched through the authenticated `gh` CLI.

The dashboard is widget-based by design: new sources (Calendar, CI, Jira) are
added by implementing the `ui.Widget` interface, not by editing the core.

## Commands

Use the Makefile (`make help` lists everything):

| Command | What it does |
|---|---|
| `make build` | Build the binary into `./bin/devdeck` |
| `make run` | Run from source |
| `make test` | Unit tests |
| `make test-race` | Tests with the race detector |
| `make cover` | Coverage profile + HTML report |
| `make cover-check` | Fail if total coverage `< 80%` (mirrors CI) |
| `make vet` / `make lint` | `go vet` / `golangci-lint` |
| `make ci` | `vet` + `lint` + `test-race` + `cover-check` (run this before pushing) |
| `make install` | `go install` to `$(go env GOPATH)/bin`, auto-adds it to `~/.zshrc` if needed |
| `make uninstall` | Remove the installed binary |

Raw equivalents: `go test ./...`, `go build ./cmd/devdeck`, `go vet ./...`.

**Go 1.26** is required (see `go.mod`).

## Repo shape

```
cmd/devdeck/        Thin entrypoint (composition root only — no logic).
internal/
  domain/           Source-agnostic Item type shared by all widgets.
  timeutil/         Pure relative-time formatting ("2w ago").
  exec/             CommandRunner interface + OS impl (the shell-out seam).
    exectest/       FakeRunner for tests (records calls, scripts output).
  github/           PR model, gh-backed PRProvider, OpenPR, mode parsing.
  config/           YAML config loading with defaults.
  ui/               Widget interface, responsive Layout, Dashboard model.
  widgets/
    githubprs/      The GitHub PR widget (implements ui.Widget).
  app/              Builds widgets from config (tested wiring).
.github/workflows/  CI (test + lint + coverage gate + PR comment).
```

## Testing / TDD rules (load-bearing)

This project is built **test-first (TDD)**. Honour it:

1. **No production code without a failing test first.** Write the test, watch it
   fail for the right reason, then write the minimal code to pass.
2. **I/O lives behind interfaces.** All external commands go through
   `exec.CommandRunner`; GitHub data comes through `github.PRProvider`. Tests
   substitute `exectest.FakeRunner` / a fake provider — **never invoke `gh` or
   the network in a unit test.**
3. **Logic lives in pure functions** (`timeutil.RelativeTime`, `ui.Layout`,
   `PR.ToItem`, `github.ParseMode`) so it is trivially testable.
4. **Coverage ≥ 80% total**, enforced by `make cover-check` and CI. Keep it
   meaningful — don't write vacuous tests for fieldless structs; do test
   behaviour, edge cases, and error paths.
5. The Bubble Tea `Dashboard`/widgets are tested by driving `Update` with
   synthetic messages and asserting state — no TTY, no golden files required.

## How to add a widget

1. Create `internal/widgets/<name>/`. **Write the test first.**
2. Implement `ui.Widget` (`ID`, `Title`, `Init`, `Update`, `View`, `Refresh`).
   Depend on an interface for any I/O so it can be faked.
3. Async results: return a `tea.Cmd` from `Init`/`Refresh`/`Update` that emits a
   message carrying your widget's `ID` (the dashboard broadcasts non-key
   messages to every widget — ignore those not addressed to you).
4. Register the type in `internal/app/build.go` (`buildWidget` switch) and add a
   default title. Add a test in `internal/app/build_test.go`.
5. Document the new `type` in `config.example.yml`.

## Conventions

- **Go 1.26**, formatted with `gofmt`; lint config in `.golangci.yml`
  (golangci-lint v1.64.x — CI is pinned to the same version).
- Keep `cmd/devdeck/main.go` thin; put testable logic in `internal/...`.
- Exported identifiers get doc comments (enforced by `revive`).
- Run `make ci` before pushing; CI will block merges that fail tests, lint, or
  drop coverage below 80%.
