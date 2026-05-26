# oss-radar

Claude Code skill for discovering, scoring, and tracking open source contribution opportunities.

## Usage

Most commands run as a **Claude Code skill**:

```
/oss-radar scan       → Scan configured repos / orgs for open issues
/oss-radar sync       → Sync your GitHub PR history into issues.yaml
/oss-radar evaluate   → AI evaluation for one issue (Claude Code only)
/oss-radar track      → Link a PR to a tracked issue
```

Build once from the project root:

```bash
go build -o oss-radar ./dashboard
```

Then run any command:

```bash
./oss-radar doctor                 # check config + connectivity
./oss-radar scan                   # scan repos for issues
./oss-radar sync                   # sync your GitHub PR history into issues.yaml
./oss-radar dashboard              # open TUI
./oss-radar track <pr-url>         # link a PR to a tracked issue
./oss-radar track <pr-url> --issue <issue-url>  # link to a specific issue
```

To install to PATH:

```bash
go build -o /usr/local/bin/oss-radar ./dashboard
# then run from anywhere:
oss-radar doctor
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

## To-Do
* [ ] Improve evaluate method