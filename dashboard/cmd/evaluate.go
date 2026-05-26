package cmd

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/guancioul/oss-radar/internal/ai"
	"github.com/guancioul/oss-radar/internal/data"
	ghclient "github.com/guancioul/oss-radar/internal/github"
)

var evaluateCmd = &cobra.Command{
	Use:   "evaluate <issue-url>",
	Short: "Evaluate an issue with rule-based scoring + Claude AI",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		issueURL := args[0]

		owner, repo, number, err := parseIssueURL(issueURL)
		if err != nil {
			return fmt.Errorf("invalid issue URL: %w", err)
		}

		token := viper.GetString("github_token")
		client := ghclient.New(token)
		ctx := context.Background()

		iss, err := client.GetIssue(ctx, owner, repo, number)
		if err != nil {
			return err
		}

		apiKey := viper.GetString("claude_api_key")
		if apiKey == "" {
			return fmt.Errorf("claude_api_key not set in config")
		}

		profile := viper.GetString("profile.goal")
		evaluator := ai.New(apiKey)
		eval, err := evaluator.Evaluate(ctx, *iss, profile)
		if err != nil {
			return err
		}

		fmt.Printf("\n=== Evaluation: %s #%d ===\n", repo, number)
		fmt.Printf("Title:    %s\n", iss.Title)
		fmt.Printf("Labels:   %s\n", strings.Join(iss.Labels, ", "))
		fmt.Printf("Verdict:  %s\n", eval.Verdict)
		fmt.Printf("Reason:   %s\n", eval.Reason)
		fmt.Printf("Time est: %s\n", eval.TimeEst)
		fmt.Printf("Approach: %s\n", eval.Approach)

		// Update issues.json
		issues := data.LoadIssues(dataDir)
		for i := range issues {
			if issues[i].URL == issueURL {
				issues[i].AIVerdict = eval.Verdict
				issues[i].AIReason = eval.Reason
				issues[i].TimeEst = eval.TimeEst
			}
		}
		return data.SaveIssues(dataDir, issues)
	},
}

func init() {
	rootCmd.AddCommand(evaluateCmd)
}

func parseIssueURL(rawURL string) (owner, repo string, number int, err error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) < 4 {
		err = fmt.Errorf("expected github.com/owner/repo/issues/N")
		return
	}
	owner = parts[0]
	repo = parts[1]
	number, err = strconv.Atoi(parts[3])
	return
}
