package test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	ghclient "github.com/guancioul/oss-ops/internal/github"
	"github.com/guancioul/oss-ops/internal/model"
	"github.com/guancioul/oss-ops/internal/scan"
)

var (
	noCfg      = map[string]model.RepoConfig{}
	noScanned  = map[string]bool{}
	defaultCfg = map[string]model.RepoConfig{"owner/repo": {}}
)

func issue(url string, assignees ...string) ghclient.Issue {
	return ghclient.Issue{
		Number:    1,
		Title:     "test issue",
		URL:       url,
		Owner:     "owner",
		Repo:      "repo",
		Assignees: assignees,
		UpdatedAt: time.Now(),
	}
}

func tracked(url, status string) model.Issue {
	return model.Issue{URL: url, Status: status, Repo: "owner/repo"}
}

func trackedRepo(url, status, repo string, labels ...string) model.Issue {
	return model.Issue{URL: url, Status: status, Repo: repo, Labels: labels}
}

func build(fetched []ghclient.Issue, issues []model.Issue, byURL map[string]int,
	repos, orgs map[string]model.RepoConfig,
	scannedRepos, scannedOrgs map[string]bool,
) []scan.IssueChange {
	return scan.BuildBatch(fetched, issues, byURL, repos, orgs, scannedRepos, scannedOrgs, "")
}

// --- new issue ---

func TestBuildBatch_NewIssue(t *testing.T) {
	fetched := []ghclient.Issue{issue("https://github.com/owner/repo/issues/1")}

	batch := build(fetched, nil, map[string]int{}, defaultCfg, noCfg, noScanned, noScanned)

	assert.Len(t, batch, 1)
	assert.Equal(t, -1, batch[0].Idx)
	assert.Equal(t, "candidate", batch[0].Issue.Status)
}

func TestBuildBatch_SkipsAssignedNewIssue(t *testing.T) {
	fetched := []ghclient.Issue{issue("https://github.com/owner/repo/issues/1", "someone")}

	batch := build(fetched, nil, map[string]int{}, defaultCfg, noCfg, noScanned, noScanned)

	assert.Empty(t, batch)
}

func TestBuildBatch_AssignedToMeNewIssue(t *testing.T) {
	url := "https://github.com/owner/repo/issues/1"
	fetched := []ghclient.Issue{issue(url, "me")}

	batch := scan.BuildBatch(fetched, nil, map[string]int{}, defaultCfg, noCfg, noScanned, noScanned, "me")

	assert.Len(t, batch, 1)
	assert.Equal(t, -1, batch[0].Idx)
	assert.Equal(t, "in-progress", batch[0].Issue.Status)
}

func TestBuildBatch_AssignedToMeExistingCandidate(t *testing.T) {
	url := "https://github.com/owner/repo/issues/1"
	existing := []model.Issue{tracked(url, "candidate")}
	fetched := []ghclient.Issue{issue(url, "me")}

	batch := scan.BuildBatch(fetched, existing, map[string]int{url: 0}, defaultCfg, noCfg, noScanned, noScanned, "me")

	assert.Len(t, batch, 1)
	assert.Equal(t, 0, batch[0].Idx)
	assert.Equal(t, "in-progress", batch[0].Issue.Status)
}

func TestBuildBatch_DeduplicatesFetched(t *testing.T) {
	url := "https://github.com/owner/repo/issues/1"
	fetched := []ghclient.Issue{issue(url), issue(url)}

	batch := build(fetched, nil, map[string]int{}, defaultCfg, noCfg, noScanned, noScanned)

	assert.Len(t, batch, 1)
}

// --- existing issue ---

func TestBuildBatch_UpdatesAssignedExisting(t *testing.T) {
	url := "https://github.com/owner/repo/issues/1"
	existing := []model.Issue{tracked(url, "candidate")}
	fetched := []ghclient.Issue{issue(url, "someone")}

	batch := build(fetched, existing, map[string]int{url: 0}, defaultCfg, noCfg, noScanned, noScanned)

	assert.Len(t, batch, 1)
	assert.Equal(t, 0, batch[0].Idx)
	assert.Equal(t, "skip", batch[0].Issue.Status)
}

func TestBuildBatch_DoesNotUpdateNonCandidate(t *testing.T) {
	url := "https://github.com/owner/repo/issues/1"
	existing := []model.Issue{tracked(url, "in-progress")}
	fetched := []ghclient.Issue{issue(url, "someone")}

	batch := build(fetched, existing, map[string]int{url: 0}, defaultCfg, noCfg, noScanned, noScanned)

	assert.Empty(t, batch)
}

// --- closed / label changed (scanned but not found) ---

func TestBuildBatch_RemovesClosedCandidate(t *testing.T) {
	url := "https://github.com/owner/repo/issues/1"
	existing := []model.Issue{tracked(url, "candidate")}
	scanned := map[string]bool{"owner/repo": true}

	batch := build(nil, existing, map[string]int{url: 0}, defaultCfg, noCfg, scanned, noScanned)

	assert.Len(t, batch, 1)
	assert.True(t, batch[0].Delete)
}

func TestBuildBatch_KeepsCandidateFromUnscannedRepo(t *testing.T) {
	url := "https://github.com/owner/repo/issues/1"
	existing := []model.Issue{tracked(url, "candidate")}

	batch := build(nil, existing, map[string]int{url: 0}, defaultCfg, noCfg, noScanned, noScanned)

	assert.Empty(t, batch)
}

func TestBuildBatch_DoesNotDoubleRemove(t *testing.T) {
	url := "https://github.com/owner/repo/issues/1"
	existing := []model.Issue{tracked(url, "candidate")}
	fetched := []ghclient.Issue{issue(url, "someone")} // triggers assigned→skip at idx 0
	scanned := map[string]bool{"owner/repo": true}

	batch := build(fetched, existing, map[string]int{url: 0}, defaultCfg, noCfg, scanned, noScanned)

	assert.Len(t, batch, 1)
	assert.False(t, batch[0].Delete) // skip update, not delete
}

// --- prune (repo removed from config) ---

func TestBuildBatch_PrunesCandidateFromDroppedRepo(t *testing.T) {
	issues := []model.Issue{trackedRepo("u1", "candidate", "owner/old-repo")}

	batch := build(nil, issues, map[string]int{"u1": 0}, noCfg, noCfg, noScanned, noScanned)
	_, _, pruned := scan.ApplyBatch(batch, &issues)

	assert.Equal(t, 1, pruned)
	assert.Empty(t, issues)
}

func TestBuildBatch_KeepsNonCandidateFromDroppedRepo(t *testing.T) {
	for _, status := range []string{"in-progress", "evaluating", "merged", "skip"} {
		issues := []model.Issue{trackedRepo("u1", status, "owner/old-repo")}

		batch := build(nil, issues, map[string]int{"u1": 0}, noCfg, noCfg, noScanned, noScanned)

		assert.Empty(t, batch, "status %q should produce no batch changes", status)
	}
}

func TestBuildBatch_KeepsCandidateInConfiguredRepo(t *testing.T) {
	issues := []model.Issue{trackedRepo("u1", "candidate", "owner/repo")}

	batch := build(nil, issues, map[string]int{"u1": 0}, defaultCfg, noCfg, noScanned, noScanned)

	assert.Empty(t, batch)
}

func TestBuildBatch_KeepsCandidateInConfiguredOrg(t *testing.T) {
	issues := []model.Issue{trackedRepo("u1", "candidate", "owner/any-repo")}
	orgs := map[string]model.RepoConfig{"owner": {}}

	batch := build(nil, issues, map[string]int{"u1": 0}, noCfg, orgs, noScanned, noScanned)

	assert.Empty(t, batch)
}

func TestBuildBatch_PrunesLabelMismatch(t *testing.T) {
	// issue has "good first issue" but config now only has "help wanted"
	issues := []model.Issue{trackedRepo("u1", "candidate", "owner/repo", "good first issue")}
	repos := map[string]model.RepoConfig{"owner/repo": {Labels: []string{"help wanted"}}}

	batch := build(nil, issues, map[string]int{"u1": 0}, repos, noCfg, noScanned, noScanned)

	assert.Len(t, batch, 1)
	assert.True(t, batch[0].Delete)
}

func TestBuildBatch_KeepsLabelMatch(t *testing.T) {
	issues := []model.Issue{trackedRepo("u1", "candidate", "owner/repo", "help wanted")}
	repos := map[string]model.RepoConfig{"owner/repo": {Labels: []string{"help wanted"}}}

	batch := build(nil, issues, map[string]int{"u1": 0}, repos, noCfg, noScanned, noScanned)

	assert.Empty(t, batch)
}

func TestBuildBatch_KeepsWhenNoStoredLabels(t *testing.T) {
	// issue has no stored labels — don't prune (labels may just not have been captured)
	issues := []model.Issue{trackedRepo("u1", "candidate", "owner/repo")}
	repos := map[string]model.RepoConfig{"owner/repo": {Labels: []string{"help wanted"}}}

	batch := build(nil, issues, map[string]int{"u1": 0}, repos, noCfg, noScanned, noScanned)

	assert.Empty(t, batch)
}

func TestBuildBatch_MixedPrune(t *testing.T) {
	issues := []model.Issue{
		trackedRepo("u1", "candidate", "owner/old-repo"),   // pruned (not in config)
		trackedRepo("u2", "in-progress", "owner/old-repo"), // survives (non-candidate)
		trackedRepo("u3", "candidate", "owner/kept-repo"),  // survives (in config)
	}
	byURL := map[string]int{"u1": 0, "u2": 1, "u3": 2}
	repos := map[string]model.RepoConfig{"owner/kept-repo": {}}

	batch := build(nil, issues, byURL, repos, noCfg, noScanned, noScanned)
	_, _, pruned := scan.ApplyBatch(batch, &issues)

	assert.Equal(t, 1, pruned)
	assert.Len(t, issues, 2)
	urls := []string{issues[0].URL, issues[1].URL}
	assert.Contains(t, urls, "u2")
	assert.Contains(t, urls, "u3")
}
