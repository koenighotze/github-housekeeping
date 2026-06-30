package reporter

import (
	"fmt"
	"io"
	"time"
)

type entry struct {
	owner, repo, ref, detail, kind string
}

type Reporter struct {
	w       io.Writer
	merged  []entry
	held    []entry
	failed  []entry
	repoMap map[string]bool
}

func New(w io.Writer) *Reporter {
	return &Reporter{w: w, repoMap: make(map[string]bool)}
}

func (r *Reporter) RecordMerged(owner, repo, ref, bumpKind string) {
	r.merged = append(r.merged, entry{owner: owner, repo: repo, ref: ref, kind: bumpKind})
	r.repoMap[owner+"/"+repo] = true
}

func (r *Reporter) RecordHeld(owner, repo, ref, reason string) {
	r.held = append(r.held, entry{owner: owner, repo: repo, ref: ref, detail: reason})
	r.repoMap[owner+"/"+repo] = true
}

func (r *Reporter) RecordFailed(owner, repo, reason string) {
	r.failed = append(r.failed, entry{owner: owner, repo: repo, detail: reason})
	r.repoMap[owner+"/"+repo] = true
}

func (r *Reporter) ExitCode() int {
	if len(r.held) > 0 || len(r.failed) > 0 {
		return 1
	}
	return 0
}

func (r *Reporter) PrintSummary() {
	_, _ = fmt.Fprintf(r.w, "\ngithub-housekeeping run — %s\n\n", time.Now().UTC().Format(time.RFC3339))

	for repoKey := range r.repoMap {
		_, _ = fmt.Fprintf(r.w, "%s\n", repoKey)

		for _, e := range r.merged {
			if e.owner+"/"+e.repo == repoKey {
				_, _ = fmt.Fprintf(r.w, "  ✓ merged   %s  (%s)\n", e.ref, e.kind)
			}
		}
		for _, e := range r.held {
			if e.owner+"/"+e.repo == repoKey {
				_, _ = fmt.Fprintf(r.w, "  ✗ held     %s  (%s — human review required)\n", e.ref, e.detail)
			}
		}
		for _, e := range r.failed {
			if e.owner+"/"+e.repo == repoKey {
				_, _ = fmt.Fprintf(r.w, "  ✗ failed   %s\n", e.detail)
			}
		}
		_, _ = fmt.Fprintln(r.w)
	}

	_, _ = fmt.Fprintf(r.w, "%d merged · %d held · %d failed\n", len(r.merged), len(r.held), len(r.failed))
}
