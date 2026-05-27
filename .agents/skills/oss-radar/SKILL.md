---
name: oss-ops
description: Discover, score, and track open source contribution opportunities
arguments: mode
user-invocable: true
argument-hint: "[scan | sync | evaluate | track <pr-url> [--issue <url>]]"
license: MIT
---

# oss-ops

All commands run the Go binary at `$PROJECT_ROOT/cli/oss-ops`.
`$PROJECT_ROOT` is the repo root containing `config.yaml` — default: `/Users/guancioul/Documents/Projects/oss-ops`.

Invoke as:
```bash
GITHUB_TOKEN=$(gh auth token) $PROJECT_ROOT/cli/oss-ops <subcommand> --config $PROJECT_ROOT/config.yaml
```

If the binary does not exist, build it first:
```bash
cd $PROJECT_ROOT/cli && go build -o oss-ops .
```

---

## Mode Routing

| Input | Action |
|-------|--------|
| (empty) | Show menu |
| `scan` | Run scan |
| `sync` | Run sync |
| `evaluate` | Evaluate all needs-evaluate issues |
| `track <pr-url> [--issue <url>]` | Run track |
| `dashboard` | Print TUI instructions |

---

## Discovery Mode (no arguments)

Print:
```
oss-ops — Open Source Contribution Tracker

  /oss-ops scan                              → Scan repos / orgs for open issues
  /oss-ops sync                              → Sync your GitHub PR history into issues.json
  /oss-ops evaluate                          → Evaluate all needs-evaluate issues with AI
  /oss-ops track <pr-url> --issue <url>      → Link a PR to a tracked issue
  /oss-ops dashboard                         → How to open the TUI

Config: edit config.yaml to set repos, labels, priority, focus_areas
```

---

## scan

```bash
GITHUB_TOKEN=$(gh auth token) $PROJECT_ROOT/cli/oss-ops scan --config $PROJECT_ROOT/config.yaml
```

Show the full output. Summarise: how many issues added and which repos were scanned.

---

## sync

```bash
GITHUB_TOKEN=$(gh auth token) $PROJECT_ROOT/cli/oss-ops sync
```

Searches all public PRs authored by the authenticated user. For each PR:
- If it references a tracked issue via closing keyword (Fixes/Closes/Resolves #N) → updates that issue's status
- Otherwise → adds the PR itself as a new record

Status mapping: PR open → `in-progress`, PR merged → `merged`, PR closed → `skip`.

Show the full output and summarise: how many updated vs added.

---

## evaluate

Do NOT run the binary. Evaluate directly using your own intelligence.

1. Read `$PROJECT_ROOT/issues.yaml` and find all issues where `status == "needs-evaluate"`.
2. Read `profile.goal` and `profile.skills` from `$PROJECT_ROOT/config.yaml`.
3. For each issue:
   a. Fetch the full issue content:
      ```bash
      gh issue view <number> --repo <owner>/<repo> --json title,body,labels,comments
      ```
   b. Evaluate against the user's profile. Produce:
      - **verdict**: `yes` / `maybe` / `no`
      - **reason**: one sentence
      - **time_est**: e.g. `2-4 hours`, `1-2 days`
      - **approach**: detailed approach (several paragraphs — what to read, where to start, pitfalls)
   c. Write a markdown report to `$PROJECT_ROOT/reports/<owner>-<repo>-<number>.md`:
      ```markdown
      # <title>

      **Repo**: owner/repo  **Issue**: #number  **URL**: <url>
      **Verdict**: yes/maybe/no  **Est**: time_est

      ## Reason
      ...

      ## Approach
      ...

      ## Issue Summary
      (brief summary of the issue content)
      ```
   d. Update the issue entry in issues.yaml:
      - `ai_verdict`, `ai_reason`, `time_est`
      - `report_path`: relative path from project root, e.g. `reports/strimzi-test-container-212.md`
      - `status` → `"evaluated"`

4. Write the updated issues.yaml back to disk.
5. Print a summary table of all evaluated issues with verdict and report path.

---

## track <pr-url> [--issue <url>]

```bash
GITHUB_TOKEN=$(gh auth token) $PROJECT_ROOT/cli/oss-ops track <pr-url> --issue <issue-url> --config $PROJECT_ROOT/config.yaml
```

`--issue` is required — the binary's interactive prompt doesn't work inside Claude. If the user omitted it, ask for the issue URL before running.

---

## dashboard

The TUI cannot run inside Claude Code. Tell the user to run:

```bash
cd /Users/guancioul/Documents/Projects/oss-ops
GITHUB_TOKEN=$(gh auth token) ./cli/oss-ops dashboard
```
