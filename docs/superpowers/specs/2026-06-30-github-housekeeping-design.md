# github-housekeeping — Design

**Date:** 2026-06-30

## Problem

Engineering teams waste time chasing Dependabot notifications across many repositories. Routine patch and minor dependency bumps are safe to merge automatically, but doing it by hand is slow and error-prone.

## Solution

A one-shot Go CLI that scans configured repositories, merges safe Dependabot PRs when CI is green, holds major bumps for human review, posts PR comments explaining skipped cases, and prints a structured summary to stdout.

---

## Stack

| Concern | Decision |
|---|---|
| Language | Go 1.25 |
| Auth | 1Password PAT via `op read` |
| Invocation | One-shot CLI (`--config`, `--dry-run`) |
| Repo config | YAML config file |
| Merge gate | Semver: patch + minor only; major always held |
| Escalation | Stdout summary + PR comment with sentinel |
| Post-merge CI | Poll main branch until green before next merge |
| Distribution | Docker image (`OP_SERVICE_ACCOUNT_TOKEN` env var) |

---

## Architecture

```
config.yaml
  └─ internal/config → Config{}
       └─ pkg/onepassword → PAT via "op read <token_ref>"
            └─ pkg/github → Client (REST API)
                 └─ internal/pipeline → for each repo:
                      ├─ internal/semver → ClassifyBump(pr.Title)
                      ├─ internal/merger → MergePR + poll main CI
                      └─ internal/reporter → stdout summary + PR comments
```

### Key design choices

- `pkg/cli.CommandRunner` is the single abstraction over `os/exec`. All external process calls go through it, making unit testing possible without real CLI tools.
- `pkg/github.Client` is an interface. Production code uses the REST implementation; tests use `net/http/httptest` — no real GitHub calls.
- `pkg/onepassword.OnePasswordClient` wraps `op read` via `CommandRunner`. Results are cached in-memory per run.
- When main CI fails after a merge, the pipeline records the failure in the reporter and stops processing remaining PRs for that repo via a sentinel `errStopRepo` — so `Run` returns nil and the reporter's exit code signals the problem.
- A `<!-- github-housekeeping -->` HTML comment sentinel prevents double-posting on re-runs.

---

## Config schema

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
    allow: [patch, minor]
  ci_poll:
    timeout: 10m
    interval: 30s
```

---

## Pipeline (sequential)

```
for each repo:
  list open PRs by dependabot[bot]
  for each PR:
    classify semver bump from PR title
    if major → post PR comment (once) + record held + skip
    if PR CI not all green → post PR comment (once) + record held + skip
    merge PR
    poll main CI until green or timeout
    if main CI fails → record failed + stop this repo
record merged + print summary
exit 0 if nothing failed/held, exit 1 otherwise
```

---

## Testing

- **Unit**: table-driven, no network, no processes (`semver`, `config`, `reporter`, `cli`, `onepassword`).
- **Integration**: `//go:build integration` tag, `net/http/httptest` for GitHub API, stubbed `op` via `CommandRunner` mock.
- **No e2e**: real GitHub + real 1Password in CI is too fragile.
