package data

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/guancioul/oss-radar/internal/model"
)

type Store struct {
	Issues   []model.Issue `json:"issues"`
	LastScan string        `json:"last_scan,omitempty"`
}

func LoadIssues(dir string) []model.Issue {
	path := filepath.Join(dir, "issues.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var store Store
	if err := json.Unmarshal(data, &store); err != nil {
		return nil
	}
	return store.Issues
}

func SaveIssues(dir string, issues []model.Issue) error {
	store := Store{Issues: issues}
	data, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "issues.json"), data, 0644)
}

func NormalizeStatus(raw string) string {
	switch raw {
	case "in-progress", "in_progress":
		return "in-progress"
	case "candidate", "evaluating", "merged", "skip":
		return raw
	default:
		return "candidate"
	}
}

func StatusPriority(status string) int {
	switch NormalizeStatus(status) {
	case "in-progress":
		return 0
	case "evaluating":
		return 1
	case "candidate":
		return 2
	case "merged":
		return 3
	case "skip":
		return 4
	default:
		return 5
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
