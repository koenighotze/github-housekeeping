package github

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestClient(t *testing.T, mux *http.ServeMux) Client {
	t.Helper()
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return newClientWithBase("test-token", srv.URL)
}

func TestListDependabotPRs(t *testing.T) {
	t.Run("should return only dependabot PRs", func(t *testing.T) {
		mux := http.NewServeMux()
		mux.HandleFunc("/repos/acme/frontend/pulls", func(w http.ResponseWriter, r *http.Request) {
			prs := []PullRequest{
				{Number: 1, Title: "Bump lodash from 4.17.20 to 4.17.21", User: struct {
					Login string `json:"login"`
				}{Login: "dependabot[bot]"}},
				{Number: 2, Title: "My manual PR", User: struct {
					Login string `json:"login"`
				}{Login: "octocat"}},
			}
			w.Header().Set("Content-Type", "application/json")
			assert.NoError(t, json.NewEncoder(w).Encode(prs))
		})

		client := newTestClient(t, mux)
		prs, err := client.ListDependabotPRs(context.Background(), "acme", "frontend")

		require.NoError(t, err)
		assert.Len(t, prs, 1)
		assert.Equal(t, 1, prs[0].Number)
	})

	t.Run("should return error on non-200", func(t *testing.T) {
		mux := http.NewServeMux()
		mux.HandleFunc("/repos/acme/frontend/pulls", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusForbidden)
		})

		client := newTestClient(t, mux)
		_, err := client.ListDependabotPRs(context.Background(), "acme", "frontend")

		assert.Error(t, err)
	})
}

func TestGetCheckRuns(t *testing.T) {
	t.Run("should return check runs for a SHA", func(t *testing.T) {
		mux := http.NewServeMux()
		mux.HandleFunc("/repos/acme/frontend/commits/abc123/check-runs", func(w http.ResponseWriter, r *http.Request) {
			result := map[string]any{
				"check_runs": []CheckRun{
					{Name: "test", Status: "completed", Conclusion: "success"},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			assert.NoError(t, json.NewEncoder(w).Encode(result))
		})

		client := newTestClient(t, mux)
		runs, err := client.GetCheckRuns(context.Background(), "acme", "frontend", "abc123")

		require.NoError(t, err)
		assert.Len(t, runs, 1)
		assert.Equal(t, "success", runs[0].Conclusion)
	})
}

func TestApprovePR(t *testing.T) {
	t.Run("should approve a PR successfully", func(t *testing.T) {
		mux := http.NewServeMux()
		mux.HandleFunc("/repos/acme/frontend/pulls/42/reviews", func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodPost, r.Method)
			var body map[string]string
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			assert.Equal(t, "APPROVE", body["event"])
			w.WriteHeader(http.StatusOK)
			_, werr := w.Write([]byte(`{"id":1}`))
			assert.NoError(t, werr)
		})

		client := newTestClient(t, mux)
		err := client.ApprovePR(context.Background(), "acme", "frontend", 42)

		assert.NoError(t, err)
	})

	t.Run("should return error on approve failure", func(t *testing.T) {
		mux := http.NewServeMux()
		mux.HandleFunc("/repos/acme/frontend/pulls/42/reviews", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnprocessableEntity)
		})

		client := newTestClient(t, mux)
		err := client.ApprovePR(context.Background(), "acme", "frontend", 42)

		assert.Error(t, err)
	})
}

func TestMergePR(t *testing.T) {
	t.Run("should merge a PR successfully", func(t *testing.T) {
		mux := http.NewServeMux()
		mux.HandleFunc("/repos/acme/frontend/pulls/42/merge", func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodPut, r.Method)
			w.WriteHeader(http.StatusOK)
			_, werr := w.Write([]byte(`{"merged":true}`))
			assert.NoError(t, werr)
		})

		client := newTestClient(t, mux)
		err := client.MergePR(context.Background(), "acme", "frontend", 42)

		assert.NoError(t, err)
	})

	t.Run("should return error on merge failure", func(t *testing.T) {
		mux := http.NewServeMux()
		mux.HandleFunc("/repos/acme/frontend/pulls/42/merge", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusMethodNotAllowed)
		})

		client := newTestClient(t, mux)
		err := client.MergePR(context.Background(), "acme", "frontend", 42)

		assert.Error(t, err)
	})
}

func TestGetMainSHA(t *testing.T) {
	t.Run("should return the HEAD SHA", func(t *testing.T) {
		mux := http.NewServeMux()
		mux.HandleFunc("/repos/acme/frontend/commits/HEAD", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			assert.NoError(t, json.NewEncoder(w).Encode(map[string]string{"sha": "deadbeef"}))
		})

		client := newTestClient(t, mux)
		sha, err := client.GetMainSHA(context.Background(), "acme", "frontend")

		require.NoError(t, err)
		assert.Equal(t, "deadbeef", sha)
	})
}

func TestPostComment(t *testing.T) {
	t.Run("should post a comment", func(t *testing.T) {
		mux := http.NewServeMux()
		mux.HandleFunc("/repos/acme/frontend/issues/7/comments", func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodPost, r.Method)
			w.WriteHeader(http.StatusCreated)
			_, werr := w.Write([]byte(`{"id":1}`))
			assert.NoError(t, werr)
		})

		client := newTestClient(t, mux)
		err := client.PostComment(context.Background(), "acme", "frontend", 7, "hello")

		assert.NoError(t, err)
	})
}

func TestCommentExists(t *testing.T) {
	t.Run("should return true when sentinel comment exists", func(t *testing.T) {
		mux := http.NewServeMux()
		mux.HandleFunc("/repos/acme/frontend/issues/7/comments", func(w http.ResponseWriter, r *http.Request) {
			comments := []map[string]string{{"body": "<!-- github-housekeeping -->\nSkipped."}}
			w.Header().Set("Content-Type", "application/json")
			assert.NoError(t, json.NewEncoder(w).Encode(comments))
		})

		client := newTestClient(t, mux)
		exists, err := client.CommentExists(context.Background(), "acme", "frontend", 7, "<!-- github-housekeeping -->")

		require.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("should return false when no sentinel comment", func(t *testing.T) {
		mux := http.NewServeMux()
		mux.HandleFunc("/repos/acme/frontend/issues/7/comments", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, werr := w.Write([]byte(`[]`))
			assert.NoError(t, werr)
		})

		client := newTestClient(t, mux)
		exists, err := client.CommentExists(context.Background(), "acme", "frontend", 7, "<!-- github-housekeeping -->")

		require.NoError(t, err)
		assert.False(t, exists)
	})
}
