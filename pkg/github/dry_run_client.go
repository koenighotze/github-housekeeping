package github

import (
	"context"
	"fmt"
	"os"
)

type dryRunClient struct {
	real Client
}

func NewDryRunClient(token string) Client {
	return &dryRunClient{real: NewClient(token)}
}

func (d *dryRunClient) ListDependabotPRs(ctx context.Context, owner, repo string) ([]PullRequest, error) {
	return d.real.ListDependabotPRs(ctx, owner, repo)
}

func (d *dryRunClient) GetCheckRuns(ctx context.Context, owner, repo, sha string) ([]CheckRun, error) {
	return d.real.GetCheckRuns(ctx, owner, repo, sha)
}

func (d *dryRunClient) MergePR(_ context.Context, owner, repo string, number int) error {
	fmt.Fprintf(os.Stderr, "[dry-run] would merge %s/%s#%d\n", owner, repo, number)
	return nil
}

func (d *dryRunClient) GetMainSHA(ctx context.Context, owner, repo string) (string, error) {
	return d.real.GetMainSHA(ctx, owner, repo)
}

func (d *dryRunClient) PostComment(_ context.Context, owner, repo string, number int, body string) error {
	fmt.Fprintf(os.Stderr, "[dry-run] would comment on %s/%s#%d: %s\n", owner, repo, number, body)
	return nil
}

func (d *dryRunClient) CommentExists(ctx context.Context, owner, repo string, number int, sentinel string) (bool, error) {
	return d.real.CommentExists(ctx, owner, repo, number, sentinel)
}
