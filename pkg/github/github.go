package github

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const defaultBaseURL = "https://api.github.com"

type PullRequest struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
	Head   struct {
		SHA string `json:"sha"`
		Ref string `json:"ref"`
	} `json:"head"`
	User struct {
		Login string `json:"login"`
	} `json:"user"`
}

type CheckRun struct {
	Name       string `json:"name"`
	Status     string `json:"status"`
	Conclusion string `json:"conclusion"`
}

type Client interface {
	ListDependabotPRs(ctx context.Context, owner, repo string) ([]PullRequest, error)
	GetCheckRuns(ctx context.Context, owner, repo, sha string) ([]CheckRun, error)
	ApprovePR(ctx context.Context, owner, repo string, number int) error
	MergePR(ctx context.Context, owner, repo string, number int) error
	GetMainSHA(ctx context.Context, owner, repo string) (string, error)
	PostComment(ctx context.Context, owner, repo string, number int, body string) error
	CommentExists(ctx context.Context, owner, repo string, number int, sentinel string) (bool, error)
}

type restClient struct {
	token   string
	baseURL string
	http    *http.Client
}

func NewClient(token string) Client {
	return &restClient{
		token:   token,
		baseURL: defaultBaseURL,
		http:    &http.Client{Timeout: 30 * time.Second},
	}
}

func newClientWithBase(token, baseURL string) Client {
	return &restClient{
		token:   token,
		baseURL: baseURL,
		http:    &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *restClient) do(ctx context.Context, method, path string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return c.http.Do(req)
}

func (c *restClient) ListDependabotPRs(ctx context.Context, owner, repo string) ([]PullRequest, error) {
	resp, err := c.do(ctx, http.MethodGet,
		fmt.Sprintf("/repos/%s/%s/pulls?state=open&per_page=100", owner, repo), nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github: list PRs %s/%s: status %d", owner, repo, resp.StatusCode)
	}

	var prs []PullRequest
	if err := json.NewDecoder(resp.Body).Decode(&prs); err != nil {
		return nil, err
	}

	var out []PullRequest
	for _, pr := range prs {
		if pr.User.Login == "dependabot[bot]" {
			out = append(out, pr)
		}
	}
	return out, nil
}

func (c *restClient) GetCheckRuns(ctx context.Context, owner, repo, sha string) ([]CheckRun, error) {
	resp, err := c.do(ctx, http.MethodGet,
		fmt.Sprintf("/repos/%s/%s/commits/%s/check-runs?per_page=100", owner, repo, sha), nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github: get check runs %s/%s@%s: status %d", owner, repo, sha, resp.StatusCode)
	}

	var result struct {
		CheckRuns []CheckRun `json:"check_runs"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result.CheckRuns, nil
}

func (c *restClient) ApprovePR(ctx context.Context, owner, repo string, number int) error {
	payload, _ := json.Marshal(map[string]string{"event": "APPROVE"})
	resp, err := c.do(ctx, http.MethodPost,
		fmt.Sprintf("/repos/%s/%s/pulls/%d/reviews", owner, repo, number),
		strings.NewReader(string(payload)))
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("github: approve PR %s/%s#%d: status %d", owner, repo, number, resp.StatusCode)
	}
	return nil
}

func (c *restClient) MergePR(ctx context.Context, owner, repo string, number int) error {
	body := strings.NewReader(`{"merge_method":"merge"}`)
	resp, err := c.do(ctx, http.MethodPut,
		fmt.Sprintf("/repos/%s/%s/pulls/%d/merge", owner, repo, number), body)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("github: merge PR %s/%s#%d: status %d", owner, repo, number, resp.StatusCode)
	}
	return nil
}

func (c *restClient) GetMainSHA(ctx context.Context, owner, repo string) (string, error) {
	resp, err := c.do(ctx, http.MethodGet,
		fmt.Sprintf("/repos/%s/%s/commits/HEAD", owner, repo), nil)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("github: get HEAD %s/%s: status %d", owner, repo, resp.StatusCode)
	}

	var commit struct {
		SHA string `json:"sha"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&commit); err != nil {
		return "", err
	}
	return commit.SHA, nil
}

func (c *restClient) PostComment(ctx context.Context, owner, repo string, number int, body string) error {
	payload, _ := json.Marshal(map[string]string{"body": body})
	resp, err := c.do(ctx, http.MethodPost,
		fmt.Sprintf("/repos/%s/%s/issues/%d/comments", owner, repo, number),
		strings.NewReader(string(payload)))
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("github: post comment %s/%s#%d: status %d", owner, repo, number, resp.StatusCode)
	}
	return nil
}

func (c *restClient) CommentExists(ctx context.Context, owner, repo string, number int, sentinel string) (bool, error) {
	resp, err := c.do(ctx, http.MethodGet,
		fmt.Sprintf("/repos/%s/%s/issues/%d/comments?per_page=100", owner, repo, number), nil)
	if err != nil {
		return false, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("github: list comments %s/%s#%d: status %d", owner, repo, number, resp.StatusCode)
	}

	var comments []struct {
		Body string `json:"body"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&comments); err != nil {
		return false, err
	}

	for _, c := range comments {
		if strings.Contains(c.Body, sentinel) {
			return true, nil
		}
	}
	return false, nil
}
