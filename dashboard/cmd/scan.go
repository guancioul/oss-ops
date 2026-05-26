package cmd

import (
	"context"
	"fmt"
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
			fmt.Printf("Scanning %s/%s...\n", repo.Owner, repo.Repo)
			issues, err := client.ListIssues(ctx, repo.Owner, repo.Repo, repo.Labels)
			if err != nil {
				fmt.Printf("  Error: %v\n", err)
				continue
			}
			for _, iss := range issues {
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
					Repo:      repo.Owner + "/" + repo.Repo,
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
