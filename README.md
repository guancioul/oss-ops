# oss-ops

Claude Code skill for discovering, scoring, and tracking open source contribution opportunities.

> Inspired by [career-ops](https://github.com/santifer/career-ops).

## Usage

### Claude Code skills

Run these inside Claude Code (no build required):

| Command | What it does |
|---------|-------------|
| `/oss-ops scan` | Scan configured repos / orgs for open issues |
| `/oss-ops sync` | Sync your GitHub PR history into issues.yaml |
| `/oss-ops evaluate` | AI evaluation of all `needs-evaluate` issues |
| `/oss-ops track <pr-url>` | Link a PR to a tracked issue |
| `/oss-ops explore <org>` | Discover opportunities across any GitHub org |
| `/oss-ops dashboard` | How to open the TUI |

### CLI

Build once from the project root:

```bash
go build -o oss-ops ./cli
```

Then run directly:

```bash
./oss-ops doctor                              # check config + connectivity
./oss-ops scan                                # scan repos for issues
./oss-ops sync                                # sync your GitHub PR history
./oss-ops dashboard                           # open TUI
./oss-ops track <pr-url> --issue <issue-url>  # link a PR to a specific issue
```

To install to PATH:

```bash
go build -o /usr/local/bin/oss-ops ./cli
```

## Setup

```bash
cp config.yaml.example config.yaml
# Fill in: github_token, claude_api_key, repos, profile
```

GitHub token is auto-fetched via `gh auth token` — no need to set it manually.

```yaml
profile:
  goal: "Contribute to cloud native and distributed systems projects."
  skills: [Go, Java, Kubernetes, Kafka, Rust]

repos:
  # Specific repo
  - owner: strimzi
    repo: strimzi-kafka-operator
    labels: [help-wanted, good-start]
    priority: high                   # high / medium / low  (+20 / +10 / +5)
    focus_areas: [kafka, operator]

  # Entire org — scans all repos in the org via GitHub Search API
  - owner: open-telemetry
    labels: [good first issue, help wanted]
```

Labels use **OR** logic: issues matching any listed label are included.
Omit `repo` to scan an entire GitHub org.

`scan` automatically removes issues from repos/orgs no longer present in `config.yaml`.

## Explore

`/oss-ops explore <org>` discovers contribution opportunities across an entire GitHub org — without requiring it to be in `config.yaml`.

```text
/oss-ops explore open-telemetry
/oss-ops explore kubernetes
```

It will:

1. List the org's active repos (up to 30, sorted by recent activity)
2. Fetch open issues labelled `good first issue` or `help wanted`
3. Check each issue for existing PRs and assignees — already-claimed issues are skipped immediately
4. Evaluate the remaining issues against your `profile.goal` and `profile.skills`
5. Return a ranked table of `yes` / `maybe` opportunities with time estimates
6. Offer to append selected issues to `issues.yaml` as `candidate`

> **Tip:** Popular orgs get `good first issue` entries claimed within hours. Run explore and move fast.

## Scoring (0–100)

Accessibility labels (`good-first issue`, `help-wanted`) take the **max**, not sum — having both doesn't stack.

| Signal | Points |
|--------|--------|
| Base | 50 |
| Repo priority: high / medium / low | +20 / +10 / +5 |
| Label: good-first issue *(best of accessibility group)* | +15 |
| Label: help-wanted *(best of accessibility group)* | +10 |
| Label: needs-proposal | +10 |
| Label: blocked | −20 |
| Updated within 30 days | +10 |
| Not updated in 180+ days | −15 |
| Title matches a `focus_areas` keyword | +10 |
| Body matches a `focus_areas` keyword | +5 |
| Comments > 10 | −10 |

Issues with an assignee are filtered out at scan time and never appear.

## Issue Pipeline

`candidate` → `evaluating` → `in-progress` → `merged` / `skip`

## Tech

Go + [Bubble Tea](https://github.com/charmbracelet/bubbletea) + [Lipgloss](https://github.com/charmbracelet/lipgloss) (Catppuccin Mocha) + GitHub API + Claude API (`claude-sonnet-4-6`)
