package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	ghclient "github.com/guancioul/oss-ops/internal/github"
)

type Evaluation struct {
	Verdict  string  `json:"verdict"`  // yes/maybe/no
	Score    float64 `json:"score"`    // 0-100
	Reason   string  `json:"reason"`
	TimeEst  string  `json:"time_est"`
	Approach string  `json:"approach"`
}

type Evaluator struct {
	client *anthropic.Client
}

func New(apiKey string) *Evaluator {
	client := anthropic.NewClient(option.WithAPIKey(apiKey))
	return &Evaluator{client: &client}
}

func (e *Evaluator) Evaluate(ctx context.Context, issue ghclient.Issue, userProfile string) (*Evaluation, error) {
	prompt := fmt.Sprintf(`You are evaluating a GitHub issue as a potential open source contribution.

User profile: %s

Issue: %s/%s #%d
Title: %s
Labels: %s
Body (first 1000 chars): %.1000s

Evaluate this issue and respond with JSON only:
{
  "verdict": "yes|maybe|no",
  "score": <0-100 integer, overall suitability for contribution>,
  "reason": "one sentence explanation",
  "time_est": "e.g. 2-4 hours / 1-2 days / 1 week",
  "approach": "brief suggested approach (1-2 sentences)"
}`,
		userProfile,
		issue.Owner, issue.Repo, issue.Number,
		issue.Title,
		strings.Join(issue.Labels, ", "),
		issue.Body,
	)

	msg, err := e.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.ModelClaudeSonnet4_6,
		MaxTokens: 512,
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)),
		},
	})
	if err != nil {
		return nil, err
	}

	var result Evaluation
	if len(msg.Content) == 0 {
		return &result, nil
	}
	text := msg.Content[0].Text
	// Extract JSON from response
	start := strings.Index(text, "{")
	end := strings.LastIndex(text, "}") + 1
	if start >= 0 && end > start {
		json.Unmarshal([]byte(text[start:end]), &result)
	}
	return &result, nil
}
