package github

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	gh "github.com/google/go-github/v62/github"
	"golang.org/x/oauth2"
)

var issueRefRe = regexp.MustCompile(`#(\d+)`)

type Client struct {
	gh    *gh.Client
	token string
}

func New(token string) *Client {
	if token == "" {
		return &Client{gh: gh.NewClient(nil)}
	}
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(context.Background(), ts)
	return &Client{gh: gh.NewClient(tc), token: token}
}

// PRRef identifies a pull request.
type PRRef struct {
	Owner  string
	Repo   string
	Number int
}

// GetClosingIssuesBatch returns a map of "owner/repo#prNumber" → closing issue numbers,
// fetching all PRs in a single GraphQL request.
func (c *Client) GetClosingIssuesBatch(ctx context.Context, prs []PRRef) (map[string][]int, error) {
	if len(prs) == 0 {
		return nil, nil
	}

	// Build query with one alias per PR
	var sb strings.Builder
	sb.WriteString("{\n")
	for i, pr := range prs {
		sb.WriteString(fmt.Sprintf(`  r%d: repository(owner: %q, name: %q) {
    pr%d: pullRequest(number: %d) {
      closingIssuesReferences(first: 25) { nodes { number } }
    }
  }
`, i, pr.Owner, pr.Repo, i, pr.Number))
	}
	sb.WriteString("}")

	body, _ := json.Marshal(map[string]string{"query": sb.String()})
	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.github.com/graphql", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var raw struct {
		Data map[string]map[string]*struct {
			ClosingIssuesReferences struct {
				Nodes []struct{ Number int }
			}
		}
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, err
	}

	result := make(map[string][]int)
	for i, pr := range prs {
		repoAlias := fmt.Sprintf("r%d", i)
		prAlias := fmt.Sprintf("pr%d", i)
		repoData, ok := raw.Data[repoAlias]
		if !ok {
			continue
		}
		prData, ok := repoData[prAlias]
		if !ok || prData == nil {
			continue
		}
		key := fmt.Sprintf("%s/%s#%d", pr.Owner, pr.Repo, pr.Number)
		for _, n := range prData.ClosingIssuesReferences.Nodes {
			result[key] = append(result[key], n.Number)
		}
	}
	return result, nil
}

type Issue struct {
	Number    int
	Title     string
	Body      string
	URL       string
	Labels    []string
	Assignees []string
	Comments  int
	UpdatedAt time.Time
	Repo      string
	Owner     string
	State     string // "open" / "closed"
}

func (c *Client) ListIssues(ctx context.Context, owner, repo string, labels []string) ([]Issue, error) {
	seen := make(map[int]bool)
	var issues []Issue

	fetchLabel := func(label string) error {
		opts := &gh.IssueListByRepoOptions{
			State:       "open",
			Labels:      []string{label},
			ListOptions: gh.ListOptions{PerPage: 50},
		}
		for {
			ghIssues, resp, err := c.gh.Issues.ListByRepo(ctx, owner, repo, opts)
			if err != nil {
				return err
			}
			for _, i := range ghIssues {
				if i.PullRequestLinks != nil || seen[i.GetNumber()] {
					continue
				}
				seen[i.GetNumber()] = true
				issue := Issue{
					Number:    i.GetNumber(),
					Title:     i.GetTitle(),
					Body:      i.GetBody(),
					URL:       i.GetHTMLURL(),
					Comments:  i.GetComments(),
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
		return nil
	}

	if len(labels) == 0 {
		return issues, nil
	}
	for _, label := range labels {
		if err := fetchLabel(label); err != nil {
			return nil, err
		}
	}
	return issues, nil
}

// SearchOrgIssues uses the Search API to find open issues across an entire org
// with ANY of the given labels (OR logic). Each result carries the actual repo
// name extracted from the HTML URL.
func (c *Client) SearchOrgIssues(ctx context.Context, org string, labels []string) ([]Issue, error) {
	seen := make(map[string]bool)
	var issues []Issue

	for _, label := range labels {
		q := fmt.Sprintf(`org:%s label:"%s" state:open is:issue`, org, label)
		opts := &gh.SearchOptions{ListOptions: gh.ListOptions{PerPage: 50}}
		for {
			result, resp, err := c.gh.Search.Issues(ctx, q, opts)
			if err != nil {
				return nil, err
			}
			for _, i := range result.Issues {
				if i.PullRequestLinks != nil || seen[i.GetHTMLURL()] {
					continue
				}
				seen[i.GetHTMLURL()] = true
				repoName := repoFromURL(i.GetRepositoryURL())
				issue := Issue{
					Number:    i.GetNumber(),
					Title:     i.GetTitle(),
					Body:      i.GetBody(),
					URL:       i.GetHTMLURL(),
					Comments:  i.GetComments(),
					UpdatedAt: i.GetUpdatedAt().Time,
					Repo:      repoName,
					Owner:     org,
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
	}
	return issues, nil
}

// repoFromURL extracts "repo-name" from "https://api.github.com/repos/owner/repo-name".
func repoFromURL(u string) string {
	parts := strings.Split(strings.TrimRight(u, "/"), "/")
	if len(parts) == 0 {
		return ""
	}
	return parts[len(parts)-1]
}

type PR struct {
	Number    int
	Title     string
	Body      string
	URL       string
	State     string // "open" / "closed"
	Merged    bool
	Repo      string
	Owner     string
	UpdatedAt time.Time
	// Issue linked via closing keyword (Fixes/Closes/Resolves #N)
	LinkedIssueNumber int
}

// GetAuthenticatedUser returns the login of the authenticated user.
func (c *Client) GetAuthenticatedUser(ctx context.Context) (string, error) {
	u, _, err := c.gh.Users.Get(ctx, "")
	if err != nil {
		return "", err
	}
	return u.GetLogin(), nil
}

// SearchUserPRs returns all PRs authored by the user across public repos.
func (c *Client) SearchUserPRs(ctx context.Context, username string) ([]PR, error) {
	var prs []PR
	q := fmt.Sprintf("author:%s type:pr is:public", username)
	opts := &gh.SearchOptions{
		Sort:        "updated",
		ListOptions: gh.ListOptions{PerPage: 100},
	}
	for {
		result, resp, err := c.gh.Search.Issues(ctx, q, opts)
		if err != nil {
			return nil, err
		}
		for _, i := range result.Issues {
			if i.PullRequestLinks == nil {
				continue
			}
			owner, repoName := ownerRepoFromURL(i.GetRepositoryURL())
			pr := PR{
				Number:            i.GetNumber(),
				Title:             i.GetTitle(),
				Body:              i.GetBody(),
				URL:               i.GetHTMLURL(),
				State:             i.GetState(),
				Merged:            !i.GetPullRequestLinks().GetMergedAt().IsZero(),
				Repo:              repoName,
				Owner:             owner,
				UpdatedAt:         i.GetUpdatedAt().Time,
				LinkedIssueNumber: extractLinkedIssue(i.GetBody()),
			}
			prs = append(prs, pr)
		}
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}
	return prs, nil
}

// extractLinkedIssue finds the first #N reference in the PR body.
func extractLinkedIssue(body string) int {
	re := issueRefRe.FindStringSubmatch(body)
	if len(re) < 2 {
		return 0
	}
	var n int
	fmt.Sscanf(re[1], "%d", &n)
	return n
}

// ownerRepoFromURL extracts owner and repo from "https://api.github.com/repos/owner/repo".
func ownerRepoFromURL(u string) (owner, repo string) {
	parts := strings.Split(strings.TrimRight(u, "/"), "/")
	if len(parts) < 2 {
		return "", ""
	}
	return parts[len(parts)-2], parts[len(parts)-1]
}

// SearchInvolvedIssues finds issues in a specific repo where the user is involved.
func (c *Client) SearchInvolvedIssues(ctx context.Context, owner, repo, username string) ([]Issue, error) {
	var issues []Issue
	q := fmt.Sprintf("involves:%s repo:%s/%s is:issue", username, owner, repo)
	opts := &gh.SearchOptions{ListOptions: gh.ListOptions{PerPage: 100}}
	for {
		result, resp, err := c.gh.Search.Issues(ctx, q, opts)
		if err != nil {
			return nil, err
		}
		for _, i := range result.Issues {
			if i.PullRequestLinks != nil {
				continue
			}
			issue := Issue{
				Number:    i.GetNumber(),
				Title:     i.GetTitle(),
				Body:      i.GetBody(),
				URL:       i.GetHTMLURL(),
				Comments:  i.GetComments(),
				UpdatedAt: i.GetUpdatedAt().Time,
				Repo:      repo,
				Owner:     owner,
				State:     i.GetState(),
			}
			for _, l := range i.Labels {
				issue.Labels = append(issue.Labels, l.GetName())
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

// SearchOrgInvolvedIssues finds issues across an entire org where the user is involved.
func (c *Client) SearchOrgInvolvedIssues(ctx context.Context, org, username string) ([]Issue, error) {
	var issues []Issue
	q := fmt.Sprintf("involves:%s org:%s is:issue", username, org)
	opts := &gh.SearchOptions{ListOptions: gh.ListOptions{PerPage: 100}}
	for {
		result, resp, err := c.gh.Search.Issues(ctx, q, opts)
		if err != nil {
			return nil, err
		}
		for _, i := range result.Issues {
			if i.PullRequestLinks != nil {
				continue
			}
			repoName := repoFromURL(i.GetRepositoryURL())
			issue := Issue{
				Number:    i.GetNumber(),
				Title:     i.GetTitle(),
				Body:      i.GetBody(),
				URL:       i.GetHTMLURL(),
				Comments:  i.GetComments(),
				UpdatedAt: i.GetUpdatedAt().Time,
				Repo:      repoName,
				Owner:     org,
				State:     i.GetState(),
			}
			for _, l := range i.Labels {
				issue.Labels = append(issue.Labels, l.GetName())
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

// SearchOrgPRs finds PRs authored by username across an entire org.
func (c *Client) SearchOrgPRs(ctx context.Context, org, username string) ([]PR, error) {
	var prs []PR
	q := fmt.Sprintf("author:%s org:%s type:pr", username, org)
	opts := &gh.SearchOptions{ListOptions: gh.ListOptions{PerPage: 100}}
	for {
		result, resp, err := c.gh.Search.Issues(ctx, q, opts)
		if err != nil {
			return nil, err
		}
		for _, i := range result.Issues {
			if i.PullRequestLinks == nil {
				continue
			}
			o, repoName := ownerRepoFromURL(i.GetRepositoryURL())
			pr := PR{
				Number:            i.GetNumber(),
				Title:             i.GetTitle(),
				Body:              i.GetBody(),
				URL:               i.GetHTMLURL(),
				State:             i.GetState(),
				Merged:            !i.GetPullRequestLinks().GetMergedAt().IsZero(),
				Repo:              repoName,
				Owner:             o,
				UpdatedAt:         i.GetUpdatedAt().Time,
				LinkedIssueNumber: extractLinkedIssue(i.GetBody()),
			}
			prs = append(prs, pr)
		}
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}
	return prs, nil
}

// SearchRepoPRs finds PRs authored by username in a specific repo.
func (c *Client) SearchRepoPRs(ctx context.Context, owner, repo, username string) ([]PR, error) {
	var prs []PR
	q := fmt.Sprintf("author:%s repo:%s/%s type:pr", username, owner, repo)
	opts := &gh.SearchOptions{ListOptions: gh.ListOptions{PerPage: 100}}
	for {
		result, resp, err := c.gh.Search.Issues(ctx, q, opts)
		if err != nil {
			return nil, err
		}
		for _, i := range result.Issues {
			if i.PullRequestLinks == nil {
				continue
			}
			pr := PR{
				Number:            i.GetNumber(),
				Title:             i.GetTitle(),
				Body:              i.GetBody(),
				URL:               i.GetHTMLURL(),
				State:             i.GetState(),
				Merged:            !i.GetPullRequestLinks().GetMergedAt().IsZero(),
				Repo:              repo,
				Owner:             owner,
				UpdatedAt:         i.GetUpdatedAt().Time,
				LinkedIssueNumber: extractLinkedIssue(i.GetBody()),
			}
			prs = append(prs, pr)
		}
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}
	return prs, nil
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
		Comments:  i.GetComments(),
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
