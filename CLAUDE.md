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
  config/config.go               # Config struct, Load() from XDG path, env overrides
  gitlab/client.go               # Raw HTTP client for GET /events?scope=all, project cache
  event/
    types.go                     # Domain types: Event, PushData
    formatter.go                 # Lipgloss-styled per-action-type rendering
  tui/
    model.go                     # Bubbletea Model (Init/Update/View), polling lifecycle
    messages.go                  # EventsFetchedMsg, FetchErrorMsg, TickMsg
    commands.go                  # fetchEventsCmd, tickCmd
    keymap.go                    # Key bindings (q, j/k, r, ?, g/G)
    styles.go                    # Lipgloss style definitions
```

## Dependencies

- **TUI**: `charm.land/bubbletea/v2`, `charm.land/lipgloss/v2`, `charm.land/bubbles/v2`
- **GitLab API**: `github.com/xanzy/go-gitlab` (for project resolution only; events use raw HTTP for `scope=all` support)
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
- Priority: CLI flags > env vars > config file > defaults
- Run `gast configure` for interactive setup with validation

## GitLab API

- Uses `GET /events?scope=all` (raw HTTP) to mirror `/dashboard/activity?filter=projects`
- Project names resolved via `GET /projects/:id` with in-memory cache
- Token requires `read_api` scope (or `api`)

## Releasing

- GoReleaser config: `.goreleaser.yaml`
- Tag with `v*` to trigger release workflow
- Homebrew tap: `pataar/homebrew-tap`
