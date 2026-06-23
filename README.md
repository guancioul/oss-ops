# oss-ops

Claude Code skill for discovering, scoring, and tracking open source contribution opportunities.

> Inspired by [career-ops](https://github.com/santifer/career-ops).

## Usage

### Claude Code skills

Run these inside Claude Code (no build required):

| Command | What it does |
|---------|-------------|
| `/oss-ops sync` | Sync your GitHub PR history into issues.yaml |
| `/oss-ops scan` | Scan configured repos / orgs for open issues |
| `/oss-ops evaluate` | AI evaluation of all `needs-evaluate` issues |
| `/oss-ops explore <org>` | Discover opportunities across any GitHub org |
| `/oss-ops dashboard` | How to open the TUI |

> **Run `sync` before `scan`.** `scan` only sees `open` issues and prunes any
> `candidate` whose URL it didn't see this run. If you merged a PR and the issue is
> still `candidate` (i.e. you haven't synced yet), `scan` will delete it instead of
> recognizing it as merged. `sync` updates its status to `merged` first, which makes
> `scan` skip it entirely.

### CLI

Build once from the project root:

```bash
go build -o oss-ops ./cli
```

Then run directly:

```bash
./oss-ops doctor                              # check config + connectivity
./oss-ops sync                                # sync your GitHub PR history
./oss-ops scan                                # scan repos for issues
./oss-ops dashboard                           # open TUI
```

To install to PATH:

```bash
go build -o /usr/local/bin/oss-ops ./cli
```

## Setup

```bash
cp config.yaml.example config.yaml
# Fill in: repos, profile
```

GitHub token is auto-fetched via `gh auth token` — no need to set it manually.
No Claude API key is needed either: `evaluate` and `explore` run as Claude Code
skill instructions, not API calls.

```yaml
profile:
  github: ""           # your GitHub username — used to fetch PR history for evaluate/sync
  goal: "Contribute to cloud native and distributed systems projects."
  skills: [Go, Java, Kubernetes, Kafka, Rust]
  custom_prompt: ""    # extra constraint appended during AI evaluation

repos:
  # Specific repo
  - owner: strimzi
    repo: strimzi-kafka-operator
    labels: [help-wanted, good-start]

  # Multiple repos under one owner
  - owner: strimzi
    repos: [strimzi-kafka-operator, strimzi-kafka-bridge]
    labels: [help-wanted]

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

`scan` does not score issues — it just tracks them as `candidate` and filters out
anything already assigned to someone else. Scoring happens during `/oss-ops evaluate`:
Claude reads each candidate's content plus your `profile` and merged PR history, then
assigns a 0–100 suitability score, a `yes`/`maybe`/`no` verdict, a time estimate, and a
written approach (see [SKILL.md](.agents/skills/oss-ops/SKILL.md) for the exact steps).

## Issue Pipeline

```text
candidate → needs-evaluate → evaluated → in-progress → merged
                                                       → rejected
candidate → skip
```

## Tech

Go + [Bubble Tea](https://github.com/charmbracelet/bubbletea) + [Lipgloss](https://github.com/charmbracelet/lipgloss) (Catppuccin Mocha) + GitHub API + Claude API (`claude-sonnet-4-6`)
