package test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	ghclient "github.com/guancioul/oss-ops/internal/github"
	"github.com/guancioul/oss-ops/internal/model"
	"github.com/guancioul/oss-ops/internal/scan"
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

func TestBuildBatch_NewIssue(t *testing.T) {
	fetched := []ghclient.Issue{issue("https://github.com/owner/repo/issues/1")}

	batch, _ := scan.BuildBatch(fetched, nil, map[string]int{})

	assert.Len(t, batch, 1)
	assert.Equal(t, -1, batch[0].Idx)
	assert.Equal(t, "candidate", batch[0].Issue.Status)
}

func TestBuildBatch_SkipsAssignedNewIssue(t *testing.T) {
	fetched := []ghclient.Issue{issue("https://github.com/owner/repo/issues/1", "someone")}

	batch, _ := scan.BuildBatch(fetched, nil, map[string]int{})

	assert.Empty(t, batch)
}

func TestBuildBatch_UpdatesAssignedExisting(t *testing.T) {
	url := "https://github.com/owner/repo/issues/1"
	existing := []model.Issue{tracked(url, "candidate")}
	fetched := []ghclient.Issue{issue(url, "someone")}

	batch, _ := scan.BuildBatch(fetched, existing, map[string]int{url: 0})

	assert.Len(t, batch, 1)
	assert.Equal(t, 0, batch[0].Idx)
	assert.Equal(t, "skip", batch[0].Issue.Status)
}

func TestBuildBatch_DoesNotUpdateNonCandidate(t *testing.T) {
	url := "https://github.com/owner/repo/issues/1"
	existing := []model.Issue{tracked(url, "in-progress")}
	fetched := []ghclient.Issue{issue(url, "someone")}

	batch, _ := scan.BuildBatch(fetched, existing, map[string]int{url: 0})

	assert.Empty(t, batch)
}

func TestBuildBatch_DeduplicatesFetched(t *testing.T) {
	url := "https://github.com/owner/repo/issues/1"
	fetched := []ghclient.Issue{issue(url), issue(url)}

	batch, _ := scan.BuildBatch(fetched, nil, map[string]int{})

	assert.Len(t, batch, 1)
}

func TestAppendClosed_MarksClosedAsSkip(t *testing.T) {
	url := "https://github.com/owner/repo/issues/1"
	existing := []model.Issue{tracked(url, "candidate")}
	scannedRepos := map[string]bool{"owner/repo": true}

	batch := scan.AppendClosed(nil, existing, map[string]bool{}, scannedRepos, map[string]bool{})

	assert.Len(t, batch, 1)
	assert.Equal(t, "skip", batch[0].Issue.Status)
}

func TestAppendClosed_SkipsUnscannedRepo(t *testing.T) {
	url := "https://github.com/owner/repo/issues/1"
	existing := []model.Issue{tracked(url, "candidate")}

	batch := scan.AppendClosed(nil, existing, map[string]bool{}, map[string]bool{}, map[string]bool{})

	assert.Empty(t, batch)
}

func TestAppendClosed_DoesNotDoubleUpdate(t *testing.T) {
	url := "https://github.com/owner/repo/issues/1"
	existing := []model.Issue{tracked(url, "candidate")}
	existingBatch := []scan.IssueChange{{Idx: 0, Issue: existing[0]}}
	scannedRepos := map[string]bool{"owner/repo": true}

	batch := scan.AppendClosed(existingBatch, existing, map[string]bool{}, scannedRepos, map[string]bool{})

	assert.Len(t, batch, 1)
}
