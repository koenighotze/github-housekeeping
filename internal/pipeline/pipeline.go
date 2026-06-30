package pipeline

import (
	"context"
	"errors"
	"fmt"

	"koenighotze.de/github-housekeeping/internal/config"
	"koenighotze.de/github-housekeeping/internal/merger"
	"koenighotze.de/github-housekeeping/internal/reporter"
	"koenighotze.de/github-housekeeping/internal/semver"
	"koenighotze.de/github-housekeeping/pkg/github"
)

const commentSentinel = "<!-- github-housekeeping -->"

var errStopRepo = errors.New("stop processing this repo")

func Run(ctx context.Context, cfg *config.Config, client github.Client, rep *reporter.Reporter) error {
	for _, repo := range cfg.Repositories {
		if err := processRepo(ctx, cfg.Policy, client, repo, rep); err != nil {
			return fmt.Errorf("processing %s/%s: %w", repo.Owner, repo.Repo, err)
		}
	}
	return nil
}

func processRepo(ctx context.Context, policy config.Policy, client github.Client, repo config.Repository, rep *reporter.Reporter) error {
	prs, err := client.ListDependabotPRs(ctx, repo.Owner, repo.Repo)
	if err != nil {
		return fmt.Errorf("listing PRs: %w", err)
	}

	for _, pr := range prs {
		if err := processPR(ctx, policy, client, repo, pr, rep); err != nil {
			if errors.Is(err, errStopRepo) {
				return nil
			}
			return err
		}
	}
	return nil
}

func processPR(ctx context.Context, policy config.Policy, client github.Client, repo config.Repository, pr github.PullRequest, rep *reporter.Reporter) error {
	bump, err := semver.ClassifyBump(pr.Title)
	if err != nil {
		postHeldComment(ctx, client, repo, pr, "unrecognised version bump pattern")
		rep.RecordHeld(repo.Owner, repo.Repo, pr.Head.Ref, "unrecognised version bump pattern")
		return nil
	}

	if bump == semver.Major {
		postHeldComment(ctx, client, repo, pr, "major version bump requires human review")
		rep.RecordHeld(repo.Owner, repo.Repo, pr.Head.Ref, "major")
		return nil
	}

	if !isBumpAllowed(bump, policy) {
		postHeldComment(ctx, client, repo, pr, fmt.Sprintf("%s version bump not in policy allow-list", bump))
		rep.RecordHeld(repo.Owner, repo.Repo, pr.Head.Ref, bump.String()+" not in allow-list")
		return nil
	}

	if err := merger.Merge(ctx, client, repo, pr, policy); err != nil {
		switch {
		case errors.Is(err, merger.ErrCIFailed):
			rep.RecordHeld(repo.Owner, repo.Repo, pr.Head.Ref, "PR CI not green")
			return nil
		case errors.Is(err, merger.ErrMainCIFailed), errors.Is(err, merger.ErrMainCITimeout):
			rep.RecordFailed(repo.Owner, repo.Repo, "main CI red after merge — remaining PRs skipped")
			return errStopRepo
		default:
			return err
		}
	}

	rep.RecordMerged(repo.Owner, repo.Repo, pr.Head.Ref, bump.String())
	return nil
}

func isBumpAllowed(bump semver.BumpKind, policy config.Policy) bool {
	bumpStr := bump.String()
	for _, allowed := range policy.Merge.Allow {
		if allowed == bumpStr {
			return true
		}
	}
	return false
}

func postHeldComment(ctx context.Context, client github.Client, repo config.Repository, pr github.PullRequest, reason string) {
	exists, err := client.CommentExists(ctx, repo.Owner, repo.Repo, pr.Number, commentSentinel)
	if err != nil || exists {
		return
	}
	body := fmt.Sprintf("%s\n⏸ Skipped by github-housekeeping: %s.", commentSentinel, reason)
	_ = client.PostComment(ctx, repo.Owner, repo.Repo, pr.Number, body)
}
