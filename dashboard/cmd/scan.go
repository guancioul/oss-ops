package cmd

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/guancioul/oss-radar/internal/data"
	ghclient "github.com/guancioul/oss-radar/internal/github"
	"github.com/guancioul/oss-radar/internal/model"
	"github.com/guancioul/oss-radar/internal/scorer"
)

type repoConfig struct {
	Owner      string   `mapstructure:"owner"`
	Repo       string   `mapstructure:"repo"`
	Repos      []string `mapstructure:"repos"`
	Labels     []string `mapstructure:"labels"`
	Priority   string   `mapstructure:"priority"`
	FocusAreas []string `mapstructure:"focus_areas"`
}

var scanCmd = &cobra.Command{
	Use:   "scan",
	Short: "Scan GitHub repos for contribution opportunities",
	RunE: func(cmd *cobra.Command, args []string) error {
		token := viper.GetString("github_token")
		client := ghclient.New(token)

		var repos []repoConfig
		if err := viper.UnmarshalKey("repos", &repos); err != nil || len(repos) == 0 {
			fmt.Println("No repos configured. Add repos to config.yaml")
			return nil
		}

		existing := data.LoadIssues(dataDir)
		existingURLs := make(map[string]bool)
		for _, iss := range existing {
			existingURLs[iss.URL] = true
		}

		var added int
		ctx := context.Background()

		for _, repo := range repos {
			// Expand into a flat list of (owner, repoName) targets.
			// Priority: repos[] > repo > org-level search.
			targets := repo.Repos
			if len(targets) == 0 && repo.Repo != "" {
				targets = []string{repo.Repo}
			}

			var allIssues []ghclient.Issue
			if len(targets) == 0 {
				fmt.Printf("Scanning org %s...\n", repo.Owner)
				iss, err := client.SearchOrgIssues(ctx, repo.Owner, repo.Labels)
				if err != nil {
					fmt.Printf("  Error: %v\n", err)
					continue
				}
				allIssues = iss
			} else {
				for _, repoName := range targets {
					fmt.Printf("Scanning %s/%s...\n", repo.Owner, repoName)
					iss, err := client.ListIssues(ctx, repo.Owner, repoName, repo.Labels)
					if err != nil {
						fmt.Printf("  Error: %v\n", err)
						continue
					}
					allIssues = append(allIssues, iss...)
				}
			}

			for _, iss := range allIssues {
				if len(iss.Assignees) > 0 {
					continue // skip assigned
				}
				if existingURLs[iss.URL] {
					continue // already tracked
				}
				repoCfg := scorer.RepoConfig{
					Priority:   repo.Priority,
					FocusAreas: repo.FocusAreas,
				}
				score := scorer.Score(iss, repoCfg)
				newIss := model.Issue{
					Number:    iss.Number,
					Repo:      iss.Owner + "/" + iss.Repo,
					Title:     iss.Title,
					URL:       iss.URL,
					Labels:    iss.Labels,
					Status:    "candidate",
					Score:     score,
					UpdatedAt: iss.UpdatedAt.Format("2006-01-02"),
					FoundAt:   time.Now().Format("2006-01-02"),
				}
				existing = append(existing, newIss)
				existingURLs[iss.URL] = true
				added++
				fmt.Printf("  + [%.0f] #%d %s\n", score, iss.Number, truncate(iss.Title, 60))
			}
		}

		// Remove issues whose repo/org is no longer in config
		allowedRepos := make(map[string]bool)
		allowedOrgs := make(map[string]bool)
		for _, r := range repos {
			if r.Repo != "" {
				allowedRepos[r.Owner+"/"+r.Repo] = true
			}
			for _, rn := range r.Repos {
				allowedRepos[r.Owner+"/"+rn] = true
			}
			if r.Repo == "" && len(r.Repos) == 0 {
				allowedOrgs[r.Owner] = true
			}
		}

		var pruned int
		kept := existing[:0]
		for _, iss := range existing {
			parts := strings.SplitN(iss.Repo, "/", 2)
			org := parts[0]
			if allowedRepos[iss.Repo] || allowedOrgs[org] {
				kept = append(kept, iss)
			} else {
				pruned++
			}
		}
		existing = kept
		if pruned > 0 {
			fmt.Printf("Pruned %d issues from repos no longer in config.\n", pruned)
		}

		if err := data.SaveIssues(dataDir, existing); err != nil {
			return err
		}
		fmt.Printf("\nAdded %d new issues. Total: %d\n", added, len(existing))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(scanCmd)
}

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n-3]) + "..."
}
