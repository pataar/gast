# gast

**G**itLab **A**ctivity **S**tream **T**UI — a terminal dashboard that mirrors your GitLab activity feed with live polling.

![gast screenshot](assets/screenshot.png)

## Features

- Live-updating feed of all GitLab project activity (pushes, merges, comments, issues, approvals)
- Color-coded events with per-author coloring
- Open events or projects directly in the browser
- @mention notifications (in-app badge + optional desktop alerts)
- Filter by project or group
- Mouse and keyboard navigation

## Install

### Homebrew

```bash
brew install pataar/tap/gast
```

### Go

```bash
go install github.com/pataar/gast@latest
```

### Binary releases

Download pre-built binaries from the [Releases](https://github.com/pataar/gast/releases) page.

### From source

```bash
git clone https://github.com/pataar/gast.git
cd gast
go build -o gast .
```

## Quick start

Run the interactive configuration wizard:

```bash
gast configure
```

This prompts for your GitLab host, personal access token, poll interval, page size, full project path preference, and desktop notifications — validates everything (including a test API call) — and writes the config to `~/.config/gast/config.toml`.

Then start the TUI:

```bash
gast
```

## Configuration

Config file location: `~/.config/gast/config.toml` (follows [XDG Base Directory](https://specifications.freedesktop.org/basedir-spec/latest/) on Linux/macOS, `%AppData%` on Windows).

```toml
gitlab_host = "https://gitlab.example.com"
notifications = false
page_size = 50
poll_interval = "30s"
show_full_project_path = false
token = "glpat-xxxxxxxxxxxxxxxxxxxx"
```

The token needs the `read_api` scope (or `api`).

### Environment variable overrides

| Variable | Config key |
|---|---|
| `GITLAB_ACTIVITY_HOST` | `gitlab_host` |
| `GITLAB_ACTIVITY_TOKEN` | `token` |
| `GITLAB_ACTIVITY_INTERVAL` | `poll_interval` |
| `GITLAB_ACTIVITY_PAGE_SIZE` | `page_size` |

### CLI flags

```
--config             Path to config file
--host               GitLab host URL
--token              GitLab personal access token
--interval           Poll interval (e.g. 30s, 1m)
--full-project-path  Show full project path instead of short name
--project            Filter to projects matching these names (comma-separated)
--group              Filter to groups matching these prefixes (comma-separated)
--demo               Run with fake data (no GitLab connection)
```

Priority order: CLI flags > environment variables > config file > defaults.

### Filtering

Show only events from specific projects or groups:

```bash
gast --project notification-service,dashboard
gast --group acme/backend
```

Projects match by substring, groups match by path prefix.

## Keybindings

| Key | Action |
|---|---|
| `j` / `k` | Select next / previous event |
| `g` / `G` | Select first / last event |
| `o` / `Enter` | Open selected event in browser |
| `p` | Open project page in browser |
| `r` | Force refresh |
| `t` | Toggle relative / absolute timestamps |
| `?` | Toggle help |
| `q` / `Ctrl+C` | Quit |

Mouse wheel scrolling is also supported.

## License

[MIT](LICENSE)
