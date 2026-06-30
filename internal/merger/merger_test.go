package merger

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"koenighotze.de/github-housekeeping/internal/config"
	"koenighotze.de/github-housekeeping/pkg/github"
)

type mockGithubClient struct {
	checkRunsPerSHA map[string][]github.CheckRun
	mainSHA         string
	approveCalled   bool
	approveErr      error
	mergeCalled     bool
	mergeErr        error
	postCommentBody string
	commentExists   bool
}

func (m *mockGithubClient) ListDependabotPRs(_ context.Context, _, _ string) ([]github.PullRequest, error) {
	return nil, nil
}

func (m *mockGithubClient) GetCheckRuns(_ context.Context, _, _, sha string) ([]github.CheckRun, error) {
	return m.checkRunsPerSHA[sha], nil
}

func (m *mockGithubClient) ApprovePR(_ context.Context, _, _ string, _ int) error {
	m.approveCalled = true
	return m.approveErr
}

func (m *mockGithubClient) MergePR(_ context.Context, _, _ string, _ int) error {
	m.mergeCalled = true
	return m.mergeErr
}

func (m *mockGithubClient) GetMainSHA(_ context.Context, _, _ string) (string, error) {
	return m.mainSHA, nil
}

func (m *mockGithubClient) PostComment(_ context.Context, _, _ string, _ int, body string) error {
	m.postCommentBody = body
	return nil
}

func (m *mockGithubClient) CommentExists(_ context.Context, _, _ string, _ int, _ string) (bool, error) {
	return m.commentExists, nil
}

func testPolicy() config.Policy {
	return config.Policy{
		Merge: config.MergePolicy{Allow: []string{"patch", "minor"}},
		CIPoll: config.CIPollPolicy{
			Timeout:  5 * time.Second,
			Interval: 10 * time.Millisecond,
		},
	}
}

func testPR() github.PullRequest {
	pr := github.PullRequest{Number: 1, Title: "Bump foo from 1.0.0 to 1.0.1"}
	pr.Head.SHA = "prsha"
	return pr
}

func TestMerge(t *testing.T) {
	t.Run("should merge when all PR checks pass", func(t *testing.T) {
		client := &mockGithubClient{
			checkRunsPerSHA: map[string][]github.CheckRun{
				"prsha":   {{Name: "ci", Status: "completed", Conclusion: "success"}},
				"mainsha": {{Name: "ci", Status: "completed", Conclusion: "success"}},
			},
			mainSHA: "mainsha",
		}

		err := Merge(context.Background(), client, config.Repository{Owner: "a", Repo: "b"}, testPR(), testPolicy())

		require.NoError(t, err)
		assert.True(t, client.approveCalled, "should approve PR before merging")
		assert.True(t, client.mergeCalled)
	})

	t.Run("should not merge when approve fails", func(t *testing.T) {
		client := &mockGithubClient{
			checkRunsPerSHA: map[string][]github.CheckRun{
				"prsha": {{Name: "ci", Status: "completed", Conclusion: "success"}},
			},
			approveErr: errors.New("approval failed"),
		}

		err := Merge(context.Background(), client, config.Repository{Owner: "a", Repo: "b"}, testPR(), testPolicy())

		assert.ErrorContains(t, err, "approval failed")
		assert.False(t, client.mergeCalled)
	})

	t.Run("should skip and post comment when PR CI is not green", func(t *testing.T) {
		client := &mockGithubClient{
			checkRunsPerSHA: map[string][]github.CheckRun{
				"prsha": {{Name: "ci", Status: "completed", Conclusion: "failure"}},
			},
		}

		err := Merge(context.Background(), client, config.Repository{Owner: "a", Repo: "b"}, testPR(), testPolicy())

		assert.ErrorIs(t, err, ErrCIFailed)
		assert.False(t, client.mergeCalled)
	})

	t.Run("should not double-post comment when sentinel already exists", func(t *testing.T) {
		client := &mockGithubClient{
			checkRunsPerSHA: map[string][]github.CheckRun{
				"prsha": {{Name: "ci", Status: "completed", Conclusion: "failure"}},
			},
			commentExists: true,
		}

		_ = Merge(context.Background(), client, config.Repository{Owner: "a", Repo: "b"}, testPR(), testPolicy())

		assert.Empty(t, client.postCommentBody)
	})

	t.Run("should return error when main CI fails after merge", func(t *testing.T) {
		client := &mockGithubClient{
			checkRunsPerSHA: map[string][]github.CheckRun{
				"prsha":   {{Name: "ci", Status: "completed", Conclusion: "success"}},
				"mainsha": {{Name: "ci", Status: "completed", Conclusion: "failure"}},
			},
			mainSHA: "mainsha",
		}

		err := Merge(context.Background(), client, config.Repository{Owner: "a", Repo: "b"}, testPR(), testPolicy())

		assert.ErrorIs(t, err, ErrMainCIFailed)
	})

	t.Run("should return timeout error when main CI does not finish in time", func(t *testing.T) {
		client := &mockGithubClient{
			checkRunsPerSHA: map[string][]github.CheckRun{
				"prsha":   {{Name: "ci", Status: "completed", Conclusion: "success"}},
				"mainsha": {{Name: "ci", Status: "in_progress", Conclusion: ""}},
			},
			mainSHA: "mainsha",
		}
		policy := testPolicy()
		policy.CIPoll.Timeout = 50 * time.Millisecond

		err := Merge(context.Background(), client, config.Repository{Owner: "a", Repo: "b"}, testPR(), policy)

		assert.ErrorIs(t, err, ErrMainCITimeout)
	})

	t.Run("should return merge error when merge API call fails", func(t *testing.T) {
		client := &mockGithubClient{
			checkRunsPerSHA: map[string][]github.CheckRun{
				"prsha": {{Name: "ci", Status: "completed", Conclusion: "success"}},
			},
			mergeErr: errors.New("merge conflict"),
		}

		err := Merge(context.Background(), client, config.Repository{Owner: "a", Repo: "b"}, testPR(), testPolicy())

		assert.Error(t, err)
		assert.ErrorContains(t, err, "merge conflict")
	})
}
