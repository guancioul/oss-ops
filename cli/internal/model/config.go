package model

type RepoConfig struct {
	Owner  string   `mapstructure:"owner"`
	Repo   string   `mapstructure:"repo"`
	Repos  []string `mapstructure:"repos"`
	Labels []string `mapstructure:"labels"`
}
