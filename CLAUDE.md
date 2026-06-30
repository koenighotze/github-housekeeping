# CLAUDE.md

This file provides guidance to Claude Code when working with code in this repository.

## Commands

```bash
make build          # compile binary
make test           # unit tests + coverage
make test.all       # unit + integration tests (requires op CLI)
make lint.local     # golangci-lint
make vet            # go fmt + go vet
make run.local      # run without compiling

# Single test
go test -v ./internal/semver/... -run TestClassifyBump
go test -v ./pkg/github/... -run TestListDependabotPRs

# Coverage HTML
make test.report
```

Integration tests require a working `op` (1Password CLI) install with a valid session.
Tag: `//go:build integration`.

## Non-negotiable standards

- No pull request should have more than 300 lines of code changes. If you need more, split into multiple PRs and consider refactoring.
- Try hard to keep files below 1000 lines. If you need more, consider refactoring.
- Avoid spaghetti code. Prefer simpler, human-readable code.
- **Test-first**: write the failing test before writing any implementation.

## Architecture

One-shot CLI that reads `config.yaml`, fetches a GitHub PAT from 1Password via `op read`,
then for each configured repository: lists Dependabot PRs, merges patch/minor bumps when CI
passes, posts a PR comment on skipped/failed PRs, and prints a summary to stdout.

**Data flow:**

```
config.yaml
  └─ internal/config → Config{}
       └─ pkg/onepassword → PAT via "op read <token_ref>"
            └─ pkg/github → GithubClient (REST API)
                 └─ internal/pipeline → for each repo:
                      ├─ internal/semver → ClassifyBump(pr.Title)
                      ├─ internal/merger → MergePR + poll main CI
                      └─ internal/reporter → stdout summary + PR comments
```

**Key design choices:**

- `pkg/cli.CommandRunner` is the single abstraction over `os/exec`. Everything else depends on it,
  which makes unit testing possible without real CLI tools.
- `pkg/github.GithubClient` is an interface; tests use `net/http/httptest` — no real GitHub calls.
- `pkg/onepassword.OnePasswordClient` wraps `op read` via `CommandRunner`; caches results in-memory.
- Integration tests are tagged `//go:build integration` and require a live `op` session.

## config.yaml format

```yaml
github:
  token_ref: "op://Personal/GitHub Housekeeping/token"

repositories:
  - owner: acme
    repo: frontend
  - owner: acme
    repo: backend

policy:
  merge:
    allow: [patch, minor]   # major bumps always held for human review
  ci_poll:
    timeout: 10m
    interval: 30s
```
