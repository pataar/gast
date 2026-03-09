# gast - GitLab Activity Stream TUI

## Build & Run

```bash
go build -o gast .
./gast
./gast --host https://gitlab.example.com --token glpat-xxx --interval 15s
```

## Test

```bash
go test ./...
go vet ./...
```

## Project Structure

```
main.go                          # Entry point, delegates to cmd.Execute()
cmd/
  root.go                        # Root cobra command, CLI flags, starts TUI
  configure.go                   # `gast configure` interactive wizard
internal/
  browser/open.go                # Cross-platform browser opening, URL construction
  config/config.go               # Config struct, Load() from XDG path, env overrides
    config_test.go               # Tests for config loading and env overrides
  demo/events.go                 # Fake event data for --demo mode
  gitlab/client.go               # GitLab API client (events, projects, user), project cache
  notify/notify.go               # Cross-platform desktop notifications (macOS/Linux)
  event/
    types.go                     # Domain types: Event, PushData
    formatter.go                 # Lipgloss-styled per-action-type rendering
    formatter_test.go            # Tests for event formatting
  tui/
    model.go                     # Bubbletea Model (Init/Update/View), polling lifecycle
    model_test.go                # Tests for TUI model
    messages.go                  # EventsFetchedMsg, FetchErrorMsg, TickMsg
    commands.go                  # fetchEventsCmd, tickCmd
    keymap.go                    # Key bindings (j/k, o, p, r, t, c, ?, g/G, q)
    styles.go                    # Lipgloss style definitions
```

## Dependencies

- **TUI**: `charm.land/bubbletea/v2`, `charm.land/lipgloss/v2`, `charm.land/bubbles/v2`
- **GitLab API**: `gitlab.com/gitlab-org/api/client-go`
- **CLI/config**: `github.com/spf13/cobra`, `github.com/spf13/viper`

## Charm v2 API Conventions

- Module paths are `charm.land/*` (not `github.com/charmbracelet/*`)
- `bubbletea/v2`: `View()` returns `tea.View`, wrap strings with `tea.NewView(s)`
- `bubbletea/v2`: AltScreen is a field on `tea.View`, not a program option
- `bubbles/v2`: `viewport.New()` takes functional options: `viewport.WithWidth(w)`, `viewport.WithHeight(h)`
- `bubbles/v2`: viewport uses `SetWidth()`, `SetHeight()` setters (fields are unexported)
- `bubbles/v2`: `spinner.New()` takes options, no `.Init()` method — use `.Tick` as cmd

## Configuration

- File: `~/.config/gast/config.toml` (XDG via `os.UserConfigDir()`)
- Env overrides: `GITLAB_ACTIVITY_HOST`, `GITLAB_ACTIVITY_TOKEN`, `GITLAB_ACTIVITY_INTERVAL`, `GITLAB_ACTIVITY_PAGE_SIZE`
- Config fields: `gitlab_host`, `token`, `poll_interval`, `page_size`, `show_full_project_path`, `notifications`
- Priority: CLI flags > env vars > config file > defaults
- Run `gast configure` for interactive setup with validation

## Key Features

- Event selection with cursor (j/k) — `o`/`Enter` opens in browser, `p` opens project
- @mention detection with header badge + optional desktop notifications (`notifications = true`)
- Togglable relative timestamps (`t` key)
- Project/group filtering via `--project` and `--group` CLI flags
- Mouse wheel scrolling, exponential backoff on errors, silent background polling
- Demo mode (`--demo`) for screenshots

## GitLab API

- Uses `gitlab.com/gitlab-org/api/client-go` library for all API calls
- Events fetched via `ListCurrentUserContributionEvents` with `Scope: "all"`
- Project names resolved via `GetProject` with in-memory cache
- Token requires `read_api` scope (or `api`)

## Releasing

- GoReleaser config: `.goreleaser.yaml`
- Tag with `v*` to trigger release workflow
- Homebrew tap: `pataar/homebrew-tap`
