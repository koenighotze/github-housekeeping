package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"koenighotze.de/github-housekeeping/internal/config"
	"koenighotze.de/github-housekeeping/internal/pipeline"
	"koenighotze.de/github-housekeeping/internal/reporter"
	gh "koenighotze.de/github-housekeeping/pkg/github"
	"koenighotze.de/github-housekeeping/pkg/onepassword"
)

var errHeldOrFailed = errors.New("run completed: some PRs were held or failed")

func main() {
	if err := rootCmd().Execute(); err != nil {
		if !errors.Is(err, errHeldOrFailed) {
			fmt.Fprintln(os.Stderr, err)
		}
		os.Exit(1)
	}
}

func rootCmd() *cobra.Command {
	var cfgPath string
	var dryRun bool

	cmd := &cobra.Command{
		Use:           "housekeeping",
		Short:         "Merge safe Dependabot PRs and keep repositories healthy",
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return run(cmd.Context(), cfgPath, dryRun)
		},
	}

	cmd.Flags().StringVar(&cfgPath, "config", "config.yaml", "path to config file")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "log actions without merging or commenting")

	return cmd
}

func run(ctx context.Context, cfgPath string, dryRun bool) error {
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	op := onepassword.NewClient()
	token, err := op.GetSecret(cfg.GitHub.TokenRef)
	if err != nil {
		return fmt.Errorf("fetching GitHub token: %w", err)
	}

	client := gh.NewClient(token)
	if dryRun {
		fmt.Fprintln(os.Stderr, "dry-run mode: no merges or comments will be made")
		client = gh.NewDryRunClient(token)
	}

	rep := reporter.New(os.Stdout)

	if err := pipeline.Run(ctx, cfg, client, rep); err != nil {
		rep.PrintSummary()
		return err
	}

	rep.PrintSummary()
	if rep.ExitCode() != 0 {
		return errHeldOrFailed
	}
	return nil
}
