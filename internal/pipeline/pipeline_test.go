package pipeline

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"koenighotze.de/github-housekeeping/internal/config"
	"koenighotze.de/github-housekeeping/internal/reporter"
	"koenighotze.de/github-housekeeping/pkg/github"
)

type mockClient struct {
	prs             map[string][]github.PullRequest
	checkRunsPerSHA map[string][]github.CheckRun
	mainSHA         string
	mergedPRs       []int
	comments        []string
	commentExists   bool
}

func (m *mockClient) ListDependabotPRs(_ context.Context, _, repo string) ([]github.PullRequest, error) {
	return m.prs[repo], nil
}

func (m *mockClient) GetCheckRuns(_ context.Context, _, _, sha string) ([]github.CheckRun, error) {
	return m.checkRunsPerSHA[sha], nil
}

func (m *mockClient) MergePR(_ context.Context, _, _ string, number int) error {
	m.mergedPRs = append(m.mergedPRs, number)
	return nil
}

func (m *mockClient) GetMainSHA(_ context.Context, _, _ string) (string, error) {
	return m.mainSHA, nil
}

func (m *mockClient) PostComment(_ context.Context, _, _ string, _ int, body string) error {
	m.comments = append(m.comments, body)
	return nil
}

func (m *mockClient) CommentExists(_ context.Context, _, _ string, _ int, _ string) (bool, error) {
	return m.commentExists, nil
}

func testConfig(repos ...config.Repository) *config.Config {
	return &config.Config{
		Repositories: repos,
		Policy: config.Policy{
			Merge: config.MergePolicy{Allow: []string{"patch", "minor"}},
			CIPoll: config.CIPollPolicy{
				Timeout:  100 * time.Millisecond,
				Interval: 10 * time.Millisecond,
			},
		},
	}
}

func prWithSHA(number int, title, sha string) github.PullRequest {
	pr := github.PullRequest{Number: number, Title: title}
	pr.Head.SHA = sha
	pr.User.Login = "dependabot[bot]"
	return pr
}

func greenChecks() []github.CheckRun {
	return []github.CheckRun{{Name: "ci", Status: "completed", Conclusion: "success"}}
}

func TestRun(t *testing.T) {
	t.Run("should merge a valid patch PR", func(t *testing.T) {
		client := &mockClient{
			prs: map[string][]github.PullRequest{
				"frontend": {prWithSHA(1, "Bump lodash from 4.17.20 to 4.17.21", "prsha")},
			},
			checkRunsPerSHA: map[string][]github.CheckRun{
				"prsha":   greenChecks(),
				"mainsha": greenChecks(),
			},
			mainSHA: "mainsha",
		}

		var buf bytes.Buffer
		r := reporter.New(&buf)
		cfg := testConfig(config.Repository{Owner: "acme", Repo: "frontend"})

		err := Run(context.Background(), cfg, client, r)

		require.NoError(t, err)
		assert.Equal(t, []int{1}, client.mergedPRs)
		assert.Equal(t, 0, r.ExitCode())
	})

	t.Run("should hold a major PR and not merge it", func(t *testing.T) {
		client := &mockClient{
			prs: map[string][]github.PullRequest{
				"frontend": {prWithSHA(2, "Bump cobra from 1.8.1 to 2.0.0", "prsha")},
			},
			checkRunsPerSHA: map[string][]github.CheckRun{
				"prsha": greenChecks(),
			},
		}

		var buf bytes.Buffer
		r := reporter.New(&buf)
		cfg := testConfig(config.Repository{Owner: "acme", Repo: "frontend"})

		err := Run(context.Background(), cfg, client, r)

		require.NoError(t, err)
		assert.Empty(t, client.mergedPRs)
		assert.Equal(t, 1, r.ExitCode())
	})

	t.Run("should stop processing repo when main CI fails", func(t *testing.T) {
		client := &mockClient{
			prs: map[string][]github.PullRequest{
				"frontend": {
					prWithSHA(1, "Bump lodash from 4.17.20 to 4.17.21", "sha1"),
					prWithSHA(2, "Bump react from 17.0.1 to 17.0.2", "sha2"),
				},
			},
			checkRunsPerSHA: map[string][]github.CheckRun{
				"sha1":    greenChecks(),
				"sha2":    greenChecks(),
				"mainsha": {{Name: "ci", Status: "completed", Conclusion: "failure"}},
			},
			mainSHA: "mainsha",
		}

		var buf bytes.Buffer
		r := reporter.New(&buf)
		cfg := testConfig(config.Repository{Owner: "acme", Repo: "frontend"})

		err := Run(context.Background(), cfg, client, r)

		require.NoError(t, err)
		assert.Len(t, client.mergedPRs, 1, "should stop after first merge fails main CI")
		assert.Equal(t, 1, r.ExitCode())
	})
}
