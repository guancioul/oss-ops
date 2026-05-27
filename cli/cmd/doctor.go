package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"

	ghclient "github.com/guancioul/oss-ops/internal/github"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check configuration and connectivity",
	RunE: func(cmd *cobra.Command, args []string) error {
		ok := true

		check := func(label string, pass bool, detail string) {
			if pass {
				fmt.Printf("  ✓  %s\n", label)
			} else {
				fmt.Printf("  ✗  %s — %s\n", label, detail)
				ok = false
			}
		}

		fmt.Println("\noss-ops doctor")

		// 1. config.yaml
		_, err := os.Stat(cfgFile)
		check("config.yaml exists", err == nil, fmt.Sprintf("not found at %s — copy config.yaml.example", cfgFile))

		// 2. gh CLI logged in
		ghAuthErr := exec.Command("gh", "auth", "status").Run()
		check("gh auth login", ghAuthErr == nil, "run `gh auth login` to authenticate")

		// 3. profile.goal
		goal := viper.GetString("profile.goal")
		check("profile.goal set", goal != "", "set profile.goal in config.yaml so Claude can evaluate issues")

		// 5. repos configured
		var repos []struct {
			Owner string `mapstructure:"owner"`
			Repo  string `mapstructure:"repo"`
		}
		_ = viper.UnmarshalKey("repos", &repos)
		check("repos configured", len(repos) > 0, "add at least one repo to config.yaml")

		// 5. GitHub API reachable (live check via gh auth token)
		if ghAuthErr == nil {
			out, err := exec.Command("gh", "auth", "token").Output()
			if err == nil {
				token := strings.TrimSpace(string(out))
				client := ghclient.New(token)
				_, err = client.ListIssues(context.Background(), "github", "gitignore", nil)
				check("GitHub API reachable", err == nil, fmt.Sprintf("API call failed: %v", err))
			}
		}

		// 7. issues.yaml (optional — warn if malformed, ok if absent)
		issuesPath := filepath.Join(dataDir, "issues.yaml")
		if raw, err := os.ReadFile(issuesPath); err == nil {
			var v map[string]any
			malformed := yaml.Unmarshal(raw, &v) != nil
			check("issues.yaml parses correctly", !malformed, "file exists but is not valid YAML — delete it to reset")
		} else {
			fmt.Printf("  -  issues.yaml not found (will be created on first scan)\n")
		}

		fmt.Println()
		if ok {
			fmt.Println("All checks passed. Run `oss-ops scan` to get started.")
		} else {
			fmt.Println("Fix the issues above, then re-run `oss-ops doctor`.")
			return fmt.Errorf("doctor found problems")
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}
