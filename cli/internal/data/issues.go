package data

import (
	"encoding/json"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/guancioul/oss-ops/internal/model"
)

type Store struct {
	Issues   []model.Issue `yaml:"issues"`
	LastScan string        `yaml:"last_scan,omitempty"`
}

func LoadIssues(dir string) []model.Issue {
	path := filepath.Join(dir, "issues.yaml")
	raw, err := os.ReadFile(path)
	if err != nil {
		// fallback: migrate from issues.json if it exists
		return migrateLegacyJSON(dir)
	}
	var store Store
	if err := yaml.Unmarshal(raw, &store); err != nil {
		return []model.Issue{}
	}
	return store.Issues
}

func SaveIssues(dir string, issues []model.Issue) error {
	store := Store{Issues: issues}
	data, err := yaml.Marshal(store)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "issues.yaml"), data, 0644)
}

func NormalizeStatus(raw string) string {
	switch raw {
	case "in-progress", "in_progress":
		return "in-progress"
	case "needs-evaluate", "needs_evaluate":
		return "needs-evaluate"
	case "candidate", "evaluated", "merged", "skip", "rejected":
		return raw
	case "evaluating": // legacy
		return "evaluated"
	default:
		return "candidate"
	}
}

func StatusPriority(status string) int {
	switch NormalizeStatus(status) {
	case "in-progress":
		return 0
	case "evaluated":
		return 1
	case "needs-evaluate":
		return 2
	case "candidate":
		return 3
	case "merged":
		return 4
	case "rejected":
		return 5
	case "skip":
		return 6
	default:
		return 6
	}
}

func ComputeMetrics(issues []model.Issue) model.PipelineMetrics {
	m := model.PipelineMetrics{
		Total:    len(issues),
		ByStatus: make(map[string]int),
	}
	var totalScore float64
	var scored int
	for _, iss := range issues {
		norm := NormalizeStatus(iss.Status)
		m.ByStatus[norm]++
		if iss.Score > 0 {
			totalScore += iss.Score
			scored++
		}
		if norm == "in-progress" {
			m.InProgress++
		}
		if norm == "merged" {
			m.Merged++
		}
	}
	if scored > 0 {
		m.AvgScore = totalScore / float64(scored)
	}
	return m
}

// migrateLegacyJSON reads issues.json, saves as issues.yaml, and removes the old file.
func migrateLegacyJSON(dir string) []model.Issue {
	jsonPath := filepath.Join(dir, "issues.json")
	raw, err := os.ReadFile(jsonPath)
	if err != nil {
		return []model.Issue{}
	}
	type jsonStore struct {
		Issues []model.Issue `json:"issues"`
	}
	var s jsonStore
	if err := json.Unmarshal(raw, &s); err != nil {
		return []model.Issue{}
	}
	_ = SaveIssues(dir, s.Issues)
	_ = os.Remove(jsonPath)
	return s.Issues
}

func UpdateIssueStatus(dir, issueURL, newStatus string) error {
	issues := LoadIssues(dir)
	for i := range issues {
		if issues[i].URL == issueURL {
			issues[i].Status = newStatus
			return SaveIssues(dir, issues)
		}
	}
	return nil
}
