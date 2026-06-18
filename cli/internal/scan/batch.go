package scan

import (
	"fmt"
	"strings"
	"time"

	ghclient "github.com/guancioul/oss-ops/internal/github"
	"github.com/guancioul/oss-ops/internal/model"
)

type IssueChange struct {
	Idx    int // -1 = new
	Issue  model.Issue
	Delete bool
}

func BuildBatch(
	allFetched []ghclient.Issue,
	tracked []model.Issue,
	byURL map[string]int,
	configuredRepos, configuredOrgs map[string]model.RepoConfig,
	scannedRepos, scannedOrgs map[string]bool,
	myGitHub string,
) []IssueChange {
	var batch []IssueChange
	fetchedURLs := make(map[string]bool)
	batchedURLs := make(map[string]bool)
	handledIdxs := make(map[int]bool)

	for _, iss := range allFetched {
		fetchedURLs[iss.URL] = true
		assignedToMe := myGitHub != "" && containsLogin(iss.Assignees, myGitHub)
		if idx, ok := byURL[iss.URL]; ok {
			if !handledIdxs[idx] && tracked[idx].Status == "candidate" && len(iss.Assignees) > 0 {
				updated := tracked[idx]
				if assignedToMe {
					updated.Status = "in-progress"
					fmt.Printf("  assigned  %s/%s #%d → in-progress\n", iss.Owner, iss.Repo, iss.Number)
				} else {
					updated.Status = "skip"
					fmt.Printf("  assigned  %s/%s #%d → skip\n", iss.Owner, iss.Repo, iss.Number)
				}
				batch = append(batch, IssueChange{Idx: idx, Issue: updated})
				handledIdxs[idx] = true
			}
		} else if !batchedURLs[iss.URL] && (len(iss.Assignees) == 0 || assignedToMe) {
			status := "candidate"
			if assignedToMe {
				status = "in-progress"
				fmt.Printf("  assigned  %s/%s #%d → in-progress\n", iss.Owner, iss.Repo, iss.Number)
			}
			batch = append(batch, IssueChange{Idx: -1, Issue: model.Issue{
				Number:    iss.Number,
				Repo:      iss.Owner + "/" + iss.Repo,
				Title:     iss.Title,
				URL:       iss.URL,
				Labels:    iss.Labels,
				Status:    status,
				UpdatedAt: iss.UpdatedAt.Format("2006-01-02"),
				FoundAt:   time.Now().Format("2006-01-02"),
			}})
			batchedURLs[iss.URL] = true
		}
	}

	for i, iss := range tracked {
		if iss.Status != "candidate" || handledIdxs[i] {
			continue
		}
		org, _, _ := strings.Cut(iss.Repo, "/")
		var cfgEntry model.RepoConfig
		if e, ok := configuredRepos[iss.Repo]; ok {
			cfgEntry = e
		} else if e, ok := configuredOrgs[org]; ok {
			cfgEntry = e
		} else {
			// repo not in config at all
			batch = append(batch, IssueChange{Idx: i, Delete: true})
			handledIdxs[i] = true
			fmt.Printf("  pruned    %s #%d (repo not in config)\n", iss.Repo, iss.Number)
			continue
		}
		switch {
		case (scannedRepos[iss.Repo] || scannedOrgs[org]) && !fetchedURLs[iss.URL]:
			batch = append(batch, IssueChange{Idx: i, Delete: true})
			handledIdxs[i] = true
			fmt.Printf("  removed   %s #%d (closed or label changed)\n", iss.Repo, iss.Number)
		case len(cfgEntry.Labels) > 0 && len(iss.Labels) > 0 && !hasAnyLabel(iss.Labels, cfgEntry.Labels):
			batch = append(batch, IssueChange{Idx: i, Delete: true})
			handledIdxs[i] = true
			fmt.Printf("  pruned    %s #%d (no matching config label)\n", iss.Repo, iss.Number)
		}
	}

	return batch
}

func containsLogin(logins []string, target string) bool {
	for _, l := range logins {
		if l == target {
			return true
		}
	}
	return false
}

func hasAnyLabel(issueLabels, configLabels []string) bool {
	set := make(map[string]bool, len(configLabels))
	for _, l := range configLabels {
		set[l] = true
	}
	for _, l := range issueLabels {
		if set[l] {
			return true
		}
	}
	return false
}

func ApplyBatch(batch []IssueChange, tracked *[]model.Issue) (added, updated, pruned int) {
	deleteIdxs := make(map[int]bool)
	for _, c := range batch {
		if c.Delete {
			deleteIdxs[c.Idx] = true
			pruned++
			continue
		}
		if c.Idx >= 0 {
			(*tracked)[c.Idx] = c.Issue
			updated++
		} else {
			*tracked = append(*tracked, c.Issue)
			added++
		}
	}
	if len(deleteIdxs) > 0 {
		kept := (*tracked)[:0]
		for i, iss := range *tracked {
			if !deleteIdxs[i] {
				kept = append(kept, iss)
			}
		}
		*tracked = kept
	}
	return
}
