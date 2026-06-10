package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/guancioul/oss-ops/internal/data"
	ghclient "github.com/guancioul/oss-ops/internal/github"
	"github.com/guancioul/oss-ops/internal/scan"
)

type repoConfig struct {
	Owner  string   `mapstructure:"owner"`
	Repo   string   `mapstructure:"repo"`
	Repos  []string `mapstructure:"repos"`
	Labels []string `mapstructure:"labels"`
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

		// Step 1: config → maps
		configuredRepos, configuredOrgs := buildConfigMaps(repos)

		// Step 2: existing → byURL
		tracked := data.LoadIssues(dataDir)
		byURL := make(map[string]int)
		for i, iss := range tracked {
			byURL[iss.URL] = i
		}

		// Step 3: fetch all from GitHub
		ctx := context.Background()
		allFetched, scannedRepos, scannedOrgs := fetchAll(ctx, client, configuredRepos, configuredOrgs)

		// Step 4: build batch
		batch, fetchedURLs := scan.BuildBatch(allFetched, tracked, byURL)
		batch = scan.AppendClosed(batch, tracked, fetchedURLs, scannedRepos, scannedOrgs)

		// Step 5: apply batch
		added, updated := scan.ApplyBatch(batch, &tracked)

		// Prune repos no longer in config
		pruned := scan.PruneUnconfigured(&tracked, configuredRepos, configuredOrgs)
		if pruned > 0 {
			fmt.Printf("Pruned %d issues from repos no longer in config.\n", pruned)
		}

		if err := data.SaveIssues(dataDir, tracked); err != nil {
			return err
		}
		fmt.Printf("\nAdded: %d  Updated: %d  Total: %d\n", added, updated, len(tracked))
		return nil
	},
}

func buildConfigMaps(repos []repoConfig) (configuredRepos, configuredOrgs map[string]scan.ConfigEntry) {
	configuredRepos = make(map[string]scan.ConfigEntry)
	configuredOrgs = make(map[string]scan.ConfigEntry)
	for _, r := range repos {
		targets := r.Repos
		if len(targets) == 0 && r.Repo != "" {
			targets = []string{r.Repo}
		}
		if len(targets) == 0 {
			configuredOrgs[r.Owner] = scan.ConfigEntry{Labels: r.Labels}
		} else {
			for _, rn := range targets {
				configuredRepos[r.Owner+"/"+rn] = scan.ConfigEntry{Labels: r.Labels}
			}
		}
	}
	return
}

func fetchAll(ctx context.Context, client *ghclient.Client, configuredRepos, configuredOrgs map[string]scan.ConfigEntry) (
	allFetched []ghclient.Issue,
	scannedRepos map[string]bool,
	scannedOrgs map[string]bool,
) {
	scannedRepos = make(map[string]bool)
	scannedOrgs = make(map[string]bool)

	for repoFull, entry := range configuredRepos {
		parts := strings.SplitN(repoFull, "/", 2)
		fmt.Printf("Scanning %s...\n", repoFull)
		iss, err := client.ListIssues(ctx, parts[0], parts[1], entry.Labels)
		if err != nil {
			fmt.Printf("  Error: %v\n", err)
			continue
		}
		scannedRepos[repoFull] = true
		allFetched = append(allFetched, iss...)
	}
	for org, entry := range configuredOrgs {
		fmt.Printf("Scanning org %s...\n", org)
		iss, err := client.SearchOrgIssues(ctx, org, entry.Labels)
		if err != nil {
			fmt.Printf("  Error: %v\n", err)
			continue
		}
		scannedOrgs[org] = true
		allFetched = append(allFetched, iss...)
	}
	return
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
