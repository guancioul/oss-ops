# oss-radar

CLI + TUI for discovering, scoring, and tracking open source contribution opportunities.

## Quick Start

```bash
cp config.yaml.example config.yaml
# Edit config.yaml with your GitHub token and repos

cd dashboard
go build -o oss-radar .

./oss-radar scan          # discover issues
./oss-radar evaluate <url>  # AI evaluation
./oss-radar dashboard     # open TUI
./oss-radar track <pr-url> --issue <issue-url>
```

## Commands

| Command | Description |
|---------|-------------|
| `oss-radar scan` | Scan configured repos for open issues |
| `oss-radar evaluate <url>` | Rule-based + Claude scoring for one issue |
| `oss-radar dashboard` | TUI browser (Bubble Tea, Catppuccin Mocha) |
| `oss-radar track <pr-url>` | Link a PR to a tracked issue |

## Tech

Go + [Bubble Tea](https://github.com/charmbracelet/bubbletea) + [Lipgloss](https://github.com/charmbracelet/lipgloss) (Catppuccin Mocha) + GitHub API + Claude API
