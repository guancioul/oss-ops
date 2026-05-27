# Changelog

## 0.1.0 (2026-05-27)


### Features

* add doctor command, fix needs-proposal scoring, drop career-ops files ([d5a84c1](https://github.com/guancioul/oss-ops/commit/d5a84c123994ff09c1ee73f248cefabab00a3533))
* overhaul sync, scorer, pipeline, and data format ([d946497](https://github.com/guancioul/oss-ops/commit/d946497a64f2db50866eda22e94e77e2ec4c4925))


### Bug Fixes

* change release-please type from go to simple ([c570a71](https://github.com/guancioul/oss-ops/commit/c570a71b6fb138c86975bf70f5e033bcd5fee9f7))
* remove stale issues.yaml check in dashboard startup ([146484d](https://github.com/guancioul/oss-ops/commit/146484d981bbe20f1e223c1298d6e049f635a458))
* set initial-version to 0.1.0 for release-please ([#11](https://github.com/guancioul/oss-ops/issues/11)) ([bda8c94](https://github.com/guancioul/oss-ops/commit/bda8c94d2000044144d45e8bfba6b054144e6eb3))

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
