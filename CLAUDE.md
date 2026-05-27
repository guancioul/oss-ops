# oss-ops

CLI + TUI for discovering, scoring, and tracking open source contribution opportunities.

## What it does

Scans GitHub repos for open issues, scores them with a rule-based scorer, lets you run Claude AI evaluation on individual issues, and tracks your contribution pipeline in a Bubble Tea TUI.

## Main Files

| Path | Purpose |
|------|---------|
| `config.yaml` | GitHub token, Claude API key, repos, profile |
| `cli/` | Go source (Cobra CLI + Bubble Tea TUI) |
| `cli/internal/scorer/scorer.go` | Rule-based scoring (0–100) |
| `cli/internal/ai/evaluator.go` | Claude AI evaluation |
| `cli/internal/github/client.go` | GitHub API client |
| `cli/internal/data/issues.go` | issues.yaml read/write |
| `cli/internal/model/issue.go` | Issue struct |
| `issues.yaml` | Local tracker (gitignored) |

## Commands

```bash
oss-ops doctor              # check config + connectivity
oss-ops scan                # discover issues from configured repos
oss-ops evaluate <url>      # rule-based + Claude scoring for one issue
oss-ops dashboard           # TUI browser
oss-ops track <pr-url>      # link a PR to a tracked issue
```

## Scoring

Rule-based scorer in `cli/internal/scorer/scorer.go`. Base 50, capped 0–100:

| Signal | Points |
|--------|--------|
| Repo priority high / medium / low | +20 / +10 / +5 |
| Label: good-start / good first issue | +15 |
| Label: help-wanted | +10 |
| Label: needs-proposal | +10 |
| Label: blocked | -20 |
| Updated within 30 days | +10 |
| Not updated in 180+ days | -15 |
| Title matches a focus_area keyword | +10 |

AI evaluation (`oss-ops evaluate`) calls Claude with the user's `profile.goal` and returns verdict (yes/maybe/no), reason, time estimate, and suggested approach.

## Issue Statuses

`candidate` → `evaluating` → `in-progress` → `merged` / `skip`

## Stack

Go + [Bubble Tea](https://github.com/charmbracelet/bubbletea) + [Lipgloss](https://github.com/charmbracelet/lipgloss) (Catppuccin Mocha) + GitHub API + Claude API (claude-sonnet-4-6)

## CI/CD

- GitHub Actions on every PR
- Dependabot monitors Go modules and GitHub Actions
- Branch protection on `main`

## Skills

- `.agents/skills/oss-ops/SKILL.md` — shared skill definition
- `.qwen/skills/oss-ops/SKILL.md` — symlink to the above
