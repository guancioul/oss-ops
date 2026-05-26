package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string
var dataDir string

var rootCmd = &cobra.Command{
	Use:   "oss-radar",
	Short: "Discover, score, and track open source contribution opportunities",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "config.yaml", "config file")
	rootCmd.PersistentFlags().StringVar(&dataDir, "data", "", "directory containing issues.yaml (default: same dir as config)")
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	}
	viper.AutomaticEnv()
	_ = viper.ReadInConfig()

	// Fall back to `gh auth token` if github_token is not set
	if viper.GetString("github_token") == "" {
		if out, err := exec.Command("gh", "auth", "token").Output(); err == nil {
			token := strings.TrimSpace(string(out))
			if token != "" {
				viper.Set("github_token", token)
			}
		}
	}

	// If --data was not explicitly set, default to the config file's directory.
	if dataDir == "" {
		if cfgFile != "" {
			dataDir = filepath.Dir(cfgFile)
		} else {
			dataDir = "."
		}
	}
}
