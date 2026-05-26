package scorer

import (
	"strings"
	"time"

	ghclient "github.com/guancioul/oss-radar/internal/github"
)

type RepoConfig struct {
	Priority   string   // high/medium/low
	FocusAreas []string
}

func Score(issue ghclient.Issue, repoCfg RepoConfig) float64 {
	score := 50.0

	// Priority bonus
	switch repoCfg.Priority {
	case "high":
		score += 20
	case "medium":
		score += 10
	case "low":
		score += 5
	}

	// Label bonuses
	for _, label := range issue.Labels {
		switch strings.ToLower(label) {
		case "good-start", "good first issue":
			score += 15
		case "help-wanted", "help wanted":
			score += 10
		case "needs-proposal", "needs proposal":
			score += 10
		case "blocked":
			score -= 20
		}
	}

	// Staleness penalty
	daysSinceUpdate := time.Since(issue.UpdatedAt).Hours() / 24
	if daysSinceUpdate < 30 {
		score += 10
	} else if daysSinceUpdate > 180 {
		score -= 15
	}

	// Focus area keyword match
	titleLower := strings.ToLower(issue.Title)
	for _, area := range repoCfg.FocusAreas {
		if strings.Contains(titleLower, strings.ToLower(area)) {
			score += 10
			break
		}
	}

	if score > 100 {
		score = 100
	}
	if score < 0 {
		score = 0
	}
	return score
}
