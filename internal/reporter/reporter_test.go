package reporter

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReporter(t *testing.T) {
	t.Run("should report merged PR", func(t *testing.T) {
		var buf bytes.Buffer
		r := New(&buf)
		r.RecordMerged("acme", "frontend", "dependabot/lodash-4.17.21", "patch")
		r.PrintSummary()

		out := buf.String()
		assert.Contains(t, out, "acme/frontend")
		assert.Contains(t, out, "merged")
		assert.Contains(t, out, "lodash")
		assert.Contains(t, out, "patch")
	})

	t.Run("should report held PR", func(t *testing.T) {
		var buf bytes.Buffer
		r := New(&buf)
		r.RecordHeld("acme", "frontend", "dependabot/go-1.23.0", "major")
		r.PrintSummary()

		out := buf.String()
		assert.Contains(t, out, "held")
		assert.Contains(t, out, "major")
	})

	t.Run("should report failed repo", func(t *testing.T) {
		var buf bytes.Buffer
		r := New(&buf)
		r.RecordFailed("acme", "backend", "main CI red after merge")
		r.PrintSummary()

		out := buf.String()
		assert.Contains(t, out, "failed")
		assert.Contains(t, out, "main CI red")
	})

	t.Run("should return exit code 0 when all merged", func(t *testing.T) {
		var buf bytes.Buffer
		r := New(&buf)
		r.RecordMerged("a", "b", "ref", "patch")

		assert.Equal(t, 0, r.ExitCode())
	})

	t.Run("should return exit code 1 when any held", func(t *testing.T) {
		var buf bytes.Buffer
		r := New(&buf)
		r.RecordHeld("a", "b", "ref", "major")

		assert.Equal(t, 1, r.ExitCode())
	})

	t.Run("should return exit code 1 when any failed", func(t *testing.T) {
		var buf bytes.Buffer
		r := New(&buf)
		r.RecordFailed("a", "b", "reason")

		assert.Equal(t, 1, r.ExitCode())
	})

	t.Run("summary counts should match records", func(t *testing.T) {
		var buf bytes.Buffer
		r := New(&buf)
		r.RecordMerged("a", "b", "ref1", "patch")
		r.RecordMerged("a", "b", "ref2", "minor")
		r.RecordHeld("a", "b", "ref3", "major")
		r.RecordFailed("a", "c", "ci failed")
		r.PrintSummary()

		out := buf.String()
		assert.True(t, strings.Contains(out, "2 merged"), "expected '2 merged' in: %s", out)
		assert.True(t, strings.Contains(out, "1 held"), "expected '1 held' in: %s", out)
		assert.True(t, strings.Contains(out, "1 failed"), "expected '1 failed' in: %s", out)
	})
}
