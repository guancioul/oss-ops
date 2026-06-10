package scan

import (
	"fmt"
	"strings"
	"time"

	ghclient "github.com/guancioul/oss-ops/internal/github"
	"github.com/guancioul/oss-ops/internal/model"
)

type ConfigEntry struct {
	Labels []string
}

type IssueChange struct {
	Idx   int // -1 = new
	Issue model.Issue
}

func BuildBatch(allFetched []ghclient.Issue, tracked []model.Issue, byURL map[string]int) (batch []IssueChange, fetchedURLs map[string]bool) {
	fetchedURLs = make(map[string]bool)
	batchedURLs := make(map[string]bool)
	updatedIdxs := make(map[int]bool)

	for _, iss := range allFetched {
		fetchedURLs[iss.URL] = true
		if idx, ok := byURL[iss.URL]; ok {
			if !updatedIdxs[idx] && tracked[idx].Status == "candidate" && len(iss.Assignees) > 0 {
				updated := tracked[idx]
				updated.Status = "skip"
				batch = append(batch, IssueChange{idx, updated})
				updatedIdxs[idx] = true
				fmt.Printf("  assigned  %s/%s #%d → skip\n", iss.Owner, iss.Repo, iss.Number)
			}
		} else if !batchedURLs[iss.URL] && len(iss.Assignees) == 0 {
			batch = append(batch, IssueChange{-1, model.Issue{
				Number:    iss.Number,
				Repo:      iss.Owner + "/" + iss.Repo,
				Title:     iss.Title,
				URL:       iss.URL,
				Labels:    iss.Labels,
				Status:    "candidate",
				UpdatedAt: iss.UpdatedAt.Format("2006-01-02"),
				FoundAt:   time.Now().Format("2006-01-02"),
			}})
			batchedURLs[iss.URL] = true
		}
	}
	return
}

func AppendClosed(batch []IssueChange, tracked []model.Issue, fetchedURLs, scannedRepos, scannedOrgs map[string]bool) []IssueChange {
	updatedIdxs := make(map[int]bool)
	for _, c := range batch {
		if c.Idx >= 0 {
			updatedIdxs[c.Idx] = true
		}
	}
	for i, iss := range tracked {
		if iss.Status != "candidate" || updatedIdxs[i] || fetchedURLs[iss.URL] {
			continue
		}
		org, _, _ := strings.Cut(iss.Repo, "/")
		if !scannedRepos[iss.Repo] && !scannedOrgs[org] {
			continue
		}
		updated := iss
		updated.Status = "skip"
		batch = append(batch, IssueChange{i, updated})
		fmt.Printf("  closed    %s #%d → skip\n", iss.Repo, iss.Number)
	}
	return batch
}

func ApplyBatch(batch []IssueChange, tracked *[]model.Issue) (added, updated int) {
	for _, c := range batch {
		if c.Idx >= 0 {
			(*tracked)[c.Idx] = c.Issue
			updated++
		} else {
			*tracked = append(*tracked, c.Issue)
			added++
		}
	}
	return
}

func PruneUnconfigured(tracked *[]model.Issue, configuredRepos, configuredOrgs map[string]ConfigEntry) int {
	pruned := 0
	kept := (*tracked)[:0]
	for _, iss := range *tracked {
		org, _, _ := strings.Cut(iss.Repo, "/")
		_, inRepos := configuredRepos[iss.Repo]
		_, inOrgs := configuredOrgs[org]
		if inRepos || inOrgs {
			kept = append(kept, iss)
		} else {
			pruned++
		}
	}
	*tracked = kept
	return pruned
}
