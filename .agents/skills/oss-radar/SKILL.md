---
name: oss-ops
description: Discover, score, and track open source contribution opportunities
arguments: mode
user-invocable: true
argument-hint: "[scan | sync | evaluate | track <pr-url> [--issue <url>] | explore <org>]"
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
| `explore <org>` | Discover contribution opportunities across a GitHub org |

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
  /oss-ops explore <org>                     → Discover contribution opportunities in a GitHub org

Config: edit config.yaml to set repos, labels, priority, focus_areas
```

---

## scan

```bash
GITHUB_TOKEN=$(gh auth token) $PROJECT_ROOT/oss-ops scan --config $PROJECT_ROOT/config.yaml
```

Show the full output. Summarise: how many issues added and which repos were scanned.

---

## sync

```bash
GITHUB_TOKEN=$(gh auth token) $PROJECT_ROOT/oss-ops sync
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
2. Read `profile.goal`, `profile.skills`, `profile.github`, and `profile.custom_prompt` from `$PROJECT_ROOT/config.yaml`.
   If `custom_prompt` is non-empty, append it as an additional constraint when evaluating each issue.
3. Fetch the user's merged PR history to understand their contribution style:
   ```bash
   gh pr list --author <profile.github> --state merged --limit 20 \
     --json title,repository,mergedAt \
     --jq '[.[] | {title: .title, repo: .repository.nameWithOwner, merged: .mergedAt}]'
   ```
   Use this history as context when evaluating — prefer issues that match the user's past contribution patterns (language, domain, PR size).
4. For each issue, **first** check if it is already claimed:
   ```bash
   gh issue view <number> --repo <owner>/<repo> --json assignees,closedByPullRequestsReferences \
     --jq '{assignees: [.assignees[].login], prs: [.closedByPullRequestsReferences[].number]}'
   ```
   - If `assignees` contains the user (`profile.github`) or `prs` contains a PR authored by the user → set `status: in-progress`, preserve existing `score`. Skip evaluation.
   - If `assignees` contains someone else → set `status: skip`, preserve existing `score`. Skip evaluation.
   - If `prs` is non-empty but the PR belongs to someone else → set `status: skip`, preserve existing `score`. Skip evaluation.
   - **Never zero out or remove an existing `score` when updating status.**
   - Run these checks in parallel for all issues before proceeding.
4. For each remaining (unclaimed) issue:
   a. Fetch the full issue content:
      ```bash
      gh issue view <number> --repo <owner>/<repo> --json title,body,labels,comments
      ```
   b. Evaluate against the user's profile and PR history. Produce:
      - **verdict**: `yes` / `maybe` / `no`
      - **score**: 0–100 integer (overall suitability — weight heavily toward past contribution patterns)
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
      - `ai_verdict`, `ai_reason`, `time_est`, `score`
      - `report_path`: relative path from project root, e.g. `reports/strimzi-test-container-212.md`
      - `status` → `"evaluated"`

4. Write the updated issues.yaml back to disk.
5. Print a summary table of all evaluated issues with verdict and report path.

---

## track <pr-url> [--issue <url>]

```bash
GITHUB_TOKEN=$(gh auth token) $PROJECT_ROOT/oss-ops track <pr-url> --issue <issue-url> --config $PROJECT_ROOT/config.yaml
```

`--issue` is required — the binary's interactive prompt doesn't work inside Claude. If the user omitted it, ask for the issue URL before running.

---

## explore <org>

Do NOT run the binary. Explore directly using your own intelligence and the `gh` CLI.

1. Read `profile.goal` and `profile.skills` from `$PROJECT_ROOT/config.yaml`.

2. List active repos in the org (up to 30, sorted by recent push):
   ```bash
   gh repo list <org> --limit 30 --json name,description,pushedAt,isArchived \
     --jq '[.[] | select(.isArchived == false)]'
   ```

3. For each repo, fetch open issues with contribution-friendly labels:
   ```bash
   gh issue list --repo <org>/<repo> --limit 20 \
     --label "good first issue,help wanted,good-start" \
     --json number,title,labels,updatedAt,url
   ```
   Skip repos with 0 matching issues.

4. **Before any evaluation**, check every candidate issue for existing PRs and assignees in parallel:
   ```bash
   gh issue view <number> --repo <org>/<repo> --json assignees,closedByPullRequestsReferences \
     --jq '{assignees: [.assignees[].login], prs: [.closedByPullRequestsReferences[].number]}'
   ```
   - If `assignees` is non-empty → discard immediately.
   - If `prs` is non-empty → discard immediately.
   - Only proceed to full evaluation for issues that pass both checks.

5. For each remaining (unclaimed) issue, evaluate against the user's profile:
   - Fetch full issue content:
     ```bash
     gh issue view <number> --repo <org>/<repo> --json title,body,labels,comments
     ```
   - Score the opportunity (use the same scoring logic as `evaluate`):
     - **verdict**: `yes` / `maybe` / `no`
     - **reason**: one sentence
     - **time_est**: e.g. `2-4 hours`, `1-2 days`
   - Only keep `yes` and `maybe` verdicts.

6. Present a ranked table (yes first, then maybe), sorted by time_est ascending:

   ```
   ## Contribution Opportunities in <org>

   | # | Repo | Issue | Verdict | Est | Reason |
   |---|------|-------|---------|-----|--------|
   | 1 | repo/name | #123 Title | ✅ yes | 2-4h | ... |
   | 2 | repo/name | #456 Title | 🤔 maybe | 1-2d | ... |
   ```

7. Ask the user: "Want me to add any of these to issues.yaml for tracking?"
   If yes, append each selected issue to `$PROJECT_ROOT/issues.yaml` with `status: candidate`.

---

## dashboard

The TUI cannot run inside Claude Code. Tell the user to run:

```bash
cd /Users/guancioul/Documents/Projects/oss-ops
GITHUB_TOKEN=$(gh auth token) ./cli/oss-ops dashboard
```
