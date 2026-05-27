package scorer

import (
	"strings"
	"time"

	ghclient "github.com/guancioul/oss-ops/internal/github"
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

	// Label bonuses — accessibility labels take the max, not sum
	accessBonus := 0.0
	needsProposal := false
	blocked := false
	for _, label := range issue.Labels {
		switch strings.ToLower(label) {
		case "good-start", "good first issue":
			if 15 > accessBonus {
				accessBonus = 15
			}
		case "help-wanted", "help wanted":
			if 10 > accessBonus {
				accessBonus = 10
			}
		case "needs-proposal", "needs proposal":
			needsProposal = true
		case "blocked":
			blocked = true
		}
	}
	score += accessBonus
	if needsProposal {
		score += 10
	}
	if blocked {
		score -= 20
	}

	// Staleness penalty
	daysSinceUpdate := time.Since(issue.UpdatedAt).Hours() / 24
	if daysSinceUpdate < 30 {
		score += 10
	} else if daysSinceUpdate > 180 {
		score -= 15
	}

	// Focus area keyword match (title + body)
	titleLower := strings.ToLower(issue.Title)
	bodyLower := strings.ToLower(issue.Body)
	for _, area := range repoCfg.FocusAreas {
		areaLower := strings.ToLower(area)
		if strings.Contains(titleLower, areaLower) {
			score += 10
			break
		}
		if strings.Contains(bodyLower, areaLower) {
			score += 5
			break
		}
	}

	// Competition signals
	if issue.Comments > 10 {
		score -= 10
	}

	if score > 100 {
		score = 100
	}
	if score < 0 {
		score = 0
	}
	return score
}
