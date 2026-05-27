# Changelog

## [Unreleased]

### Added
- `needs-evaluate` status: mark issues for AI evaluation from TUI
- `evaluated` status: issues that have been assessed by Claude
- `rejected` status: contributions that were closed without merge
- TUI `R` key: open evaluation report in viewer
- TUI `p` key: open PR URL in browser
- Auto-fetch GitHub token via `gh auth token` if not set in config
- `scan` auto-prunes issues from repos/orgs no longer in config
- Evaluation report system: Claude writes markdown reports to `reports/`, path stored in `issues.yaml`
- `reports/` directory support with `report_path` field in Issue model

### Changed
- Project renamed from **oss-ops** to **oss-ops**
- `issues.json` → `issues.yaml` (auto-migrates on first run)
- `sync` rewritten: searches `involves:user` per configured repo/org instead of global PR search
- `sync` uses batched GraphQL `closingIssuesReferences` to accurately match PRs to issues
- Scorer: accessibility labels (`good-first issue`, `help-wanted`) now take **max** instead of stacking
- Scorer: added body keyword match (+5), comments > 10 penalty (−10)
- Assignee filtering moved to scan time (never stored in issues.yaml)
- `evaluate` removed from CLI binary — Claude Code skill only

### Removed
- `claude_api_key` and `github_token` from `config.yaml` (token auto-fetched)
- `remove` command (replaced by scan auto-prune)
- `evaluate` CLI command

---

Inspired by [career-ops](https://github.com/santifer/career-ops).
