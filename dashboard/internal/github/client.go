package github

import (
	"context"
	"time"

	gh "github.com/google/go-github/v62/github"
	"golang.org/x/oauth2"
)

type Client struct {
	gh *gh.Client
}

func New(token string) *Client {
	if token == "" {
		return &Client{gh: gh.NewClient(nil)}
	}
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(context.Background(), ts)
	return &Client{gh: gh.NewClient(tc)}
}

type Issue struct {
	Number    int
	Title     string
	Body      string
	URL       string
	Labels    []string
	Assignees []string
	UpdatedAt time.Time
	Repo      string
	Owner     string
}

func (c *Client) ListIssues(ctx context.Context, owner, repo string, labels []string) ([]Issue, error) {
	var issues []Issue
	opts := &gh.IssueListByRepoOptions{
		State:       "open",
		Labels:      labels,
		ListOptions: gh.ListOptions{PerPage: 50},
	}
	for {
		ghIssues, resp, err := c.gh.Issues.ListByRepo(ctx, owner, repo, opts)
		if err != nil {
			return nil, err
		}
		for _, i := range ghIssues {
			if i.PullRequestLinks != nil {
				continue
			}
			issue := Issue{
				Number:    i.GetNumber(),
				Title:     i.GetTitle(),
				Body:      i.GetBody(),
				URL:       i.GetHTMLURL(),
				UpdatedAt: i.GetUpdatedAt().Time,
				Repo:      repo,
				Owner:     owner,
			}
			for _, l := range i.Labels {
				issue.Labels = append(issue.Labels, l.GetName())
			}
			for _, a := range i.Assignees {
				issue.Assignees = append(issue.Assignees, a.GetLogin())
			}
			issues = append(issues, issue)
		}
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}
	return issues, nil
}

func (c *Client) GetIssue(ctx context.Context, owner, repo string, number int) (*Issue, error) {
	i, _, err := c.gh.Issues.Get(ctx, owner, repo, number)
	if err != nil {
		return nil, err
	}
	if i.PullRequestLinks != nil {
		return nil, nil
	}
	issue := &Issue{
		Number:    i.GetNumber(),
		Title:     i.GetTitle(),
		Body:      i.GetBody(),
		URL:       i.GetHTMLURL(),
		UpdatedAt: i.GetUpdatedAt().Time,
		Repo:      repo,
		Owner:     owner,
	}
	for _, l := range i.Labels {
		issue.Labels = append(issue.Labels, l.GetName())
	}
	for _, a := range i.Assignees {
		issue.Assignees = append(issue.Assignees, a.GetLogin())
	}
	return issue, nil
}
