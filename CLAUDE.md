# oss-ops

CLI + TUI for discovering and tracking open source contribution opportunities.

## What it does

Scans GitHub repos for open issues, scores them with a rule-based scorer, and tracks your contribution pipeline in a Bubble Tea TUI.

## Architecture

Three-layer architecture. Follow **SOLID principles** when implementing new features.

```text
cmd/              ← CLI boundary only: parse flags, print output, nothing else
internal/app/     ← orchestration: coordinates layers below, one file per command
internal/
  model/          ← domain types (no dependencies on other internal packages)
  scan/           ← batch logic (pure, no I/O)
  github/         ← GitHub API adapter
  data/           ← YAML persistence adapter
  ui/             ← Bubble Tea TUI
  theme/          ← styling
```

Import rules:

- `cmd` → `app`, `model` only (never imports `scan`, `data`, `github` directly)
- `app` → `model`, `scan`, `data`, `github` (never imports `cmd`, `ui`)
- `scan/data/github` → `model` only (never import each other)
- `ui` → `model`, `data`, `theme` (never imports `app`, `cmd`)

## Main Files

| Path | Purpose |
| --- | --- |
| `config.yaml` | GitHub token, repos, profile |
| `cli/` | Go source (Cobra CLI + Bubble Tea TUI) |
| `cli/internal/app/` | Orchestration layer (use cases) |
| `cli/internal/github/client.go` | GitHub API client |
| `cli/internal/data/issues.go` | issues.yaml read/write |
| `cli/internal/model/` | Domain types |
| `cli/internal/scan/batch.go` | Batch processing logic |
| `issues.yaml` | Local tracker (gitignored) |

## Commands

```bash
oss-ops doctor              # check config + connectivity
oss-ops scan                # discover issues from configured repos
oss-ops dashboard           # TUI browser
```

## Issue Statuses

```text
candidate → needs-evaluate → evaluated → in-progress → merged
                                                      → rejected
candidate → skip
```

## Writing Standards

All Markdown files must pass `markdownlint`. Key rules:

- Fenced code blocks must specify a language (` ```go `, ` ```bash `, ` ```text `, etc.)
- Lists must be surrounded by blank lines
- Table pipes must have spaces on both sides of cell content

## Stack

Go + [Bubble Tea](https://github.com/charmbracelet/bubbletea) + [Lipgloss](https://github.com/charmbracelet/lipgloss) (Catppuccin Mocha) + GitHub API + Claude API (claude-sonnet-4-6)

## CI/CD

- GitHub Actions on every PR
- Dependabot monitors Go modules and GitHub Actions
- Branch protection on `main`

## Skills

- `.agents/skills/oss-ops/SKILL.md` — shared skill definition
- `.qwen/skills/oss-ops/SKILL.md` — symlink to the above
