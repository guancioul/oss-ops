# Proposal 001 — Architecture Refactor

## Status

Draft — open for discussion

---

## 1. Current State

### What the repo does

`oss-ops` is a personal CLI + TUI tool that:

1. Scans configured GitHub repos/orgs for open issues matching label filters
2. Scores issues by `goal`, `skills`, `custom_prompt`, GitHub contribution history; user selects candidates for deeper AI (Claude) evaluation
3. Tracks issues locally in `issues.yaml` through a pipeline:

   ```text
   candidate → needs-evaluate → evaluated → in-progress → merged
                                                         → rejected
   candidate → skip
   ```

4. Displays the pipeline in a Bubble Tea TUI dashboard

### Current problems

1. **`cmd/` does too much.** `scan.go` owns config parsing, GitHub fetching, batch
   building, and saving — it is the orchestration layer, not just a CLI adapter.

2. **No interface boundaries.** `cmd/scan.go` imports `data`, `github`, and `scan`
   directly. There is no way to test the command logic without real I/O.

3. **Duplicate / overlapping types.** `repoConfig` (cmd) and `ConfigEntry` (scan) both
   represent "a configured repo with labels" — two structs for the same concept.

4. **Dead code.** `scorer/scorer.go` is not imported by anything. `ai/evaluator.go`
   has no associated command.

5. **Stale statuses.** `NormalizeStatus` and pipeline tabs reference `needs-evaluate`,
   `evaluated`, `rejected` — states that can only be set by the removed commands.

---

## 2. Decisions Already Made

| Decision | Status |
| --- | --- |
| Remove `track` command | Done |
| Keep `sync` command | Decided |
| Remove `ai/evaluator.go` | Done |
| Remove `scorer/scorer.go` | Done |
| Keep `model/pr.go` | Decided |
| Centralize domain types into `internal/model` | In progress (`RepoConfig` added) |
| Follow SOLID when implementing new features | Decided (documented in CLAUDE.md) |

---

## 3. Proposed Architecture

### Principle

> `cmd` depends only on `port` interfaces — never on `app` or infrastructure directly.
> `port` defines what each command needs. `app` implements it, composing `scan`, `data`, and `github`.

### Layer diagram

```text
cmd/
  root.go       — config init, flags, composition root
  scan.go       — parse flags → call port.Scanner → print result
  doctor.go     — parse flags → call port.Doctor  → print result
  dashboard.go  — launch TUI

internal/
  port/
    scanner.go  — type Scanner interface { Scan(ctx, cfg) (ScanResult, error) }
    doctor.go   — type Doctor  interface { Check(ctx) (DoctorResult, error) }

  app/          — implements port interfaces; orchestrates scan/, data/, github/
    scan.go     — scanService{ gh *github.Client, store *data.Store } implements port.Scanner
    doctor.go   — doctorService{ gh *github.Client } implements port.Doctor

  model/        — pure domain types, no dependencies
    issue.go    — Issue, PipelineMetrics
    config.go   — RepoConfig
    pr.go       — PR, PRStatus

  scan/         — pure batch logic (no I/O)
    batch.go    — BuildBatch, ApplyBatch, IssueChange

  data/         — YAML persistence
    issues.go   — Store{}, Load, Save, ComputeMetrics

  github/       — GitHub API adapter
    client.go   — Client{}, ListIssues, SearchOrgIssues, ...

  ui/           — Bubble Tea TUI
    screens/
      pipeline.go
      progress.go
  theme/
```

### Import rules

| Package | May import | Must NOT import |
| --- | --- | --- |
| `cmd` | `port`, `model`, `ui`, `theme` | `app`, `scan`, `data`, `github` directly |
| `port` | `model` | everything else |
| `app` | `port`, `model`, `scan`, `data`, `github` | `cmd`, `ui` |
| `scan` | `model` | `data`, `github`, `app`, `cmd` |
| `data` | `model` | `scan`, `github`, `app`, `cmd` |
| `github` | `model` | `scan`, `data`, `app`, `cmd` |
| `ui` | `model`, `data`, `theme` | `app`, `cmd`, `port` |

### Composition root

`cmd/root.go` wires concrete implementations into the interfaces:

```go
store  := data.NewStore(dataDir)
client := github.New(token)

var scanner port.Scanner = app.NewScanService(client, store, repos)
var doctor  port.Doctor  = app.NewDoctorService(client, cfgFile)

rootCmd.AddCommand(cmd.NewScanCmd(scanner))
rootCmd.AddCommand(cmd.NewDoctorCmd(doctor))
```

---

## 4. Open Questions

| # | Question | Options | Recommendation |
| --- | --- | --- | --- |
| 1 | Where do interfaces live? | `port/`, inline in `cmd/`, `app/port/` | `internal/port/` — consumed by `cmd`, single place |
| 2 | Return types from interfaces | `(ScanResult, error)` struct vs multiple returns | Struct — easier to extend without breaking interface |
| 3 | Dashboard wiring | Dashboard reads data directly vs goes through port | Direct `data.LoadIssues` is fine — read-only display |
| 4 | Stale statuses in pipeline | Remove `needs-evaluate`, `evaluated`, `rejected` tabs | Remove `needs-evaluate`/`evaluated`; keep `rejected` |
| 5 | Viewer screen | Remove (no AI reports) or keep as general file viewer | Remove — no trigger without evaluate command |

---

## 5. Milestones

### Milestone 0 — Foundation (current work)

Clean up and establish the architecture. No new user-facing features.

- Remove `track`, AI evaluator, scorer, dead code (keep `sync`)
- Establish three-layer architecture (`cmd` / `app` / infrastructure)
- Centralize types in `internal/model`
- Commands: `scan`, `doctor`, `dashboard`

### Milestone 1 — AI Scoring

User-triggered AI evaluation on selected candidate issues.

- User marks a candidate as `needs-evaluate` from the dashboard
- `oss-ops evaluate <issue-url>` calls Claude with issue content + full user profile (`goal`, `skills`, `custom_prompt`, GitHub contribution history)
- Claude returns verdict (yes / maybe / no), reason, time estimate, and suggested approach
- Result stored on the `Issue` and visible in dashboard
- Requires: `claude_api_key` in `config.yaml`, `profile.goal`, and `profile.github`

---

## 6. Migration Steps (proposed order)

1. Remove `internal/ai/`, `internal/scorer/`, `internal/model/pr.go`
2. Audit `internal/github/client.go` — remove functions only used by removed code
3. Remove PR/AI fields from `model/issue.go` that `sync` doesn't need
4. Clean up stale statuses in `data/issues.go` and pipeline tabs
5. Remove `internal/ui/screens/viewer.go` and its dashboard wiring
6. Create `internal/port/` with `Scanner` and `Doctor` interfaces
7. Create `internal/app/` implementing those interfaces, composing `scan/`, `data/`, `github/`
8. Refactor `cmd/scan.go` and `cmd/doctor.go` to depend on `port` interfaces
9. Wire everything in `cmd/root.go` (composition root)
10. Update tests: `cmd` tests use interface fakes; `app` tests use real `data/` and `github/`

---

## 7. What Is NOT Changing

- `cmd/dashboard.go` + `internal/ui/screens/pipeline.go` — TUI stays as-is
- `internal/scan/batch.go` — pure logic, already clean
- `internal/data/issues.go` — persistence layer, already clean
- `internal/github/client.go` — keep after removing sync-only methods
- `config.yaml` format — no changes to user-facing config
