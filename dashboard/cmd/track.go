package cmd

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/guancioul/oss-radar/internal/data"
)

var trackCmd = &cobra.Command{
	Use:   "track <pr-url>",
	Short: "Mark an issue as in-progress with a PR link",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		prURL := args[0]
		issueURL, _ := cmd.Flags().GetString("issue")

		issues := data.LoadIssues(dataDir)
		if issueURL == "" {
			// Try to match by repo/number from PR URL
			owner, repo, _, err := parsePRURL(prURL)
			if err == nil {
				repoFull := owner + "/" + repo
				for i := range issues {
					if issues[i].Repo == repoFull && issues[i].Status == "candidate" {
						fmt.Printf("Matched: %s #%d — %s\n", issues[i].Repo, issues[i].Number, issues[i].Title)
						fmt.Print("Mark as in-progress? [y/N] ")
						var resp string
						fmt.Scanln(&resp)
						if strings.ToLower(resp) == "y" {
							issues[i].Status = "in-progress"
							issues[i].PRURL = prURL
							return data.SaveIssues(dataDir, issues)
						}
					}
				}
			}
			return fmt.Errorf("use --issue <issue-url> to specify which issue this PR belongs to")
		}

		for i := range issues {
			if issues[i].URL == issueURL {
				issues[i].Status = "in-progress"
				issues[i].PRURL = prURL
				fmt.Printf("Tracked: %s as in-progress with PR %s\n", issues[i].Title, prURL)
				return data.SaveIssues(dataDir, issues)
			}
		}
		return fmt.Errorf("issue not found: %s", issueURL)
	},
}

func init() {
	trackCmd.Flags().String("issue", "", "Issue URL to associate with this PR")
	rootCmd.AddCommand(trackCmd)
}

func parsePRURL(rawURL string) (owner, repo string, number int, err error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) < 4 {
		err = fmt.Errorf("expected github.com/owner/repo/pull/N")
		return
	}
	owner = parts[0]
	repo = parts[1]
	number, err = strconv.Atoi(parts[3])
	return
}
