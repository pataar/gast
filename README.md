# gast

**G**itLab **A**ctivity **S**tream **T**UI — a terminal dashboard that mirrors your GitLab activity feed with live polling.

![gast screenshot](assets/screenshot.png)

## Features

- Live-updating feed of all GitLab project activity (pushes, merges, comments, issues, approvals)
- Color-coded events for quick scanning
- Project names and timestamps at a glance
- Configurable poll interval and page size
- Keyboard-driven navigation

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

This prompts for your GitLab host, personal access token, poll interval, and page size — validates everything (including a test API call) — and writes the config to `~/.config/gast/config.toml`.

Then start the TUI:

```bash
gast
```

## Configuration

Config file location: `~/.config/gast/config.toml` (follows [XDG Base Directory](https://specifications.freedesktop.org/basedir-spec/latest/) on Linux/macOS, `%AppData%` on Windows).

```toml
gitlab_host = "https://gitlab.example.com"
token = "glpat-xxxxxxxxxxxxxxxxxxxx"
poll_interval = "30s"
page_size = 50
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
--config    Path to config file
--host      GitLab host URL
--token     GitLab personal access token
--interval  Poll interval (e.g. 30s, 1m)
```

Priority order: CLI flags > environment variables > config file > defaults.

## Keybindings

| Key | Action |
|---|---|
| `j` / `k` | Scroll down / up |
| `g` / `G` | Go to top / bottom |
| `r` | Force refresh |
| `?` | Toggle help |
| `q` / `Ctrl+C` | Quit |

## License

[MIT](LICENSE)
