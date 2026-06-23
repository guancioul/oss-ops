package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/guancioul/oss-ops/internal/data"
	ghclient "github.com/guancioul/oss-ops/internal/github"
	"github.com/guancioul/oss-ops/internal/model"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync contribution history from GitHub into issues.yaml",
	RunE: func(cmd *cobra.Command, args []string) error {
		token := viper.GetString("github_token")
		client := ghclient.New(token)
		ctx := context.Background()

		// Get Github username from config
		username := viper.GetString("profile.github")
		if username == "" {
			u, err := client.GetAuthenticatedUser(ctx)
			if err != nil {
				return fmt.Errorf("could not detect GitHub username (set profile.github in config): %w", err)
			}
			username = u
		}
		fmt.Printf("Syncing issues for %s...\n\n", username)

		var repos []model.RepoConfig
		if err := viper.UnmarshalKey("repos", &repos); err != nil {
			return err
		}

		// Load existing tracked issues and build lookup maps
		tracked := data.LoadIssues(dataDir)
		byURL := make(map[string]int)
		byRepoNum := make(map[string]int)
		for i, iss := range tracked {
			byURL[iss.URL] = i
			byRepoNum[fmt.Sprintf("%s#%d", iss.Repo, iss.Number)] = i
		}

		updated, added := 0, 0

		for _, repo := range repos {
			targets := repo.Repos
			if len(targets) == 0 && repo.Repo != "" {
				targets = []string{repo.Repo}
			}

			// org-level: targets is empty → use [""] as a sentinel for org-wide search
			searchTargets := targets
			if len(searchTargets) == 0 {
				searchTargets = []string{""}
			}

			var allIssues []ghclient.Issue
			var allPRs []ghclient.PR

			for _, repoName := range searchTargets {
				if repoName == "" {
					fmt.Printf("Scanning org %s for your issues...\n", repo.Owner)
				} else {
					fmt.Printf("Scanning %s/%s for your issues...\n", repo.Owner, repoName)
				}
				iss, err := client.SearchInvolvedIssues(ctx, repo.Owner, repoName, username)
				if err != nil {
					fmt.Printf("  Error: %v\n", err)
					continue
				}
				allIssues = append(allIssues, iss...)

				prs, err := client.SearchPRs(ctx, repo.Owner, repoName, username)
				if err != nil {
					fmt.Printf("  Error fetching PRs: %v\n", err)
					continue
				}
				allPRs = append(allPRs, prs...)
			}

			// Build PR lookup via batched GraphQL closingIssuesReferences
			prByIssueKey := make(map[string]*model.PR)
			var refs []ghclient.PRRef
			for _, pr := range allPRs {
				refs = append(refs, ghclient.PRRef{Owner: pr.Owner, Repo: pr.Repo, Number: pr.Number})
			}
			closing, err := client.GetClosingIssuesBatch(ctx, refs)
			if err != nil {
				fmt.Printf("  Warning: GraphQL batch failed (%v), falling back to body parsing\n", err)
			}
			for _, pr := range allPRs {
				prKey := fmt.Sprintf("%s/%s#%d", pr.Owner, pr.Repo, pr.Number)
				d := pr.Domain()
				nums, ok := closing[prKey]
				if !ok || len(nums) == 0 {
					if pr.LinkedIssueNumber > 0 {
						k := fmt.Sprintf("%s/%s#%d", pr.Owner, pr.Repo, pr.LinkedIssueNumber)
						prByIssueKey[k] = &d
					}
					continue
				}
				for _, n := range nums {
					k := fmt.Sprintf("%s/%s#%d", pr.Owner, pr.Repo, n)
					prByIssueKey[k] = &d
				}
			}

			for _, iss := range allIssues {
				repoFull := iss.Owner + "/" + iss.Repo
				key := fmt.Sprintf("%s#%d", repoFull, iss.Number)

				pr := prByIssueKey[key]

				if idx, ok := byRepoNum[key]; ok {
					existing := &tracked[idx]
					status := issueStatus(iss, pr, existing.Status)
					newPRURL := ""
					newPRNum := 0
					if pr != nil {
						newPRURL = pr.URL
						newPRNum = pr.Number
					}
					if existing.Status == status && existing.PRURL == newPRURL {
						continue
					}
					existing.Status = status
					if newPRURL != "" {
						existing.PRURL = newPRURL
						existing.PRNumber = newPRNum
					}
					fmt.Printf("  updated  %s #%d → %s\n", repoFull, iss.Number, status)
					updated++
				} else {
					if _, exists := byURL[iss.URL]; exists {
						continue
					}
					status := issueStatus(iss, pr, "")
					newIss := model.Issue{
						Number:    iss.Number,
						Repo:      repoFull,
						Title:     iss.Title,
						URL:       iss.URL,
						Labels:    iss.Labels,
						Status:    status,
						UpdatedAt: iss.UpdatedAt.Format("2006-01-02"),
						FoundAt:   time.Now().Format("2006-01-02"),
					}
					if pr != nil {
						newIss.PRURL = pr.URL
						newIss.PRNumber = pr.Number
					}
					tracked = append(tracked, newIss)
					byURL[iss.URL] = len(tracked) - 1
					byRepoNum[key] = len(tracked) - 1
					fmt.Printf("  added    %s #%d %s → %s\n", repoFull, iss.Number, truncate(iss.Title, 50), status)
					added++
				}
			}
		}

		if err := data.SaveIssues(dataDir, tracked); err != nil {
			return err
		}
		fmt.Printf("\nDone. Updated: %d  Added: %d  Total: %d\n", updated, added, len(tracked))
		return nil
	},
}

func issueStatus(iss ghclient.Issue, pr *model.PR, existingStatus string) string {
	if pr != nil {
		switch pr.Status {
		case model.PRStatusMerged:
			return "merged"
		case model.PRStatusOpen:
			return "in-progress"
		default:
			return "rejected"
		}
	}
	if iss.State == "closed" {
		return "skip"
	}
	// No PR, issue still open — preserve existing status if it's more specific than candidate.
	// sync finding you as "involved" doesn't mean the status should regress.
	if existingStatus != "" && existingStatus != "candidate" {
		return existingStatus
	}
	return "candidate"
}

func init() {
	rootCmd.AddCommand(syncCmd)
}
