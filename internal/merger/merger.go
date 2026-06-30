package merger

import (
	"context"
	"errors"
	"fmt"
	"time"

	"koenighotze.de/github-housekeeping/internal/config"
	"koenighotze.de/github-housekeeping/pkg/github"
)

const commentSentinel = "<!-- github-housekeeping -->"

var (
	ErrCIFailed      = errors.New("PR CI checks failed")
	ErrMainCIFailed  = errors.New("main branch CI failed after merge")
	ErrMainCITimeout = errors.New("timed out waiting for main branch CI")
)

func Merge(ctx context.Context, client github.Client, repo config.Repository, pr github.PullRequest, policy config.Policy) error {
	if err := assertPRChecksGreen(ctx, client, repo, pr); err != nil {
		return err
	}

	if err := client.ApprovePR(ctx, repo.Owner, repo.Repo, pr.Number); err != nil {
		return fmt.Errorf("approving PR #%d: %w", pr.Number, err)
	}

	if err := client.MergePR(ctx, repo.Owner, repo.Repo, pr.Number); err != nil {
		return fmt.Errorf("merging PR #%d: %w", pr.Number, err)
	}

	return pollMainCI(ctx, client, repo, policy)
}

func assertPRChecksGreen(ctx context.Context, client github.Client, repo config.Repository, pr github.PullRequest) error {
	runs, err := client.GetCheckRuns(ctx, repo.Owner, repo.Repo, pr.Head.SHA)
	if err != nil {
		return fmt.Errorf("getting check runs for PR #%d: %w", pr.Number, err)
	}

	if !allGreen(runs) {
		if err := postCommentOnce(ctx, client, repo, pr.Number,
			commentSentinel+"\n⏸ Skipped by github-housekeeping: CI checks not green."); err != nil {
			return fmt.Errorf("posting skip comment: %w", err)
		}
		return ErrCIFailed
	}
	return nil
}

func pollMainCI(ctx context.Context, client github.Client, repo config.Repository, policy config.Policy) error {
	deadline := time.Now().Add(policy.CIPoll.Timeout)

	for {
		sha, err := client.GetMainSHA(ctx, repo.Owner, repo.Repo)
		if err != nil {
			return fmt.Errorf("getting main SHA: %w", err)
		}

		runs, err := client.GetCheckRuns(ctx, repo.Owner, repo.Repo, sha)
		if err != nil {
			return fmt.Errorf("getting main check runs: %w", err)
		}

		if allGreen(runs) {
			return nil
		}

		if anyFailed(runs) {
			return ErrMainCIFailed
		}

		if time.Now().After(deadline) {
			return ErrMainCITimeout
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(policy.CIPoll.Interval):
		}
	}
}

func allGreen(runs []github.CheckRun) bool {
	if len(runs) == 0 {
		return false
	}
	for _, r := range runs {
		if r.Status != "completed" {
			return false
		}
		if r.Conclusion != "success" && r.Conclusion != "skipped" {
			return false
		}
	}
	return true
}

func anyFailed(runs []github.CheckRun) bool {
	for _, r := range runs {
		if r.Status == "completed" && r.Conclusion == "failure" {
			return true
		}
	}
	return false
}

func postCommentOnce(ctx context.Context, client github.Client, repo config.Repository, number int, body string) error {
	exists, err := client.CommentExists(ctx, repo.Owner, repo.Repo, number, commentSentinel)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	return client.PostComment(ctx, repo.Owner, repo.Repo, number, body)
}
