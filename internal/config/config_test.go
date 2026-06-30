package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeTemp(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "config-*.yaml")
	require.NoError(t, err)
	_, err = f.WriteString(content)
	require.NoError(t, err)
	require.NoError(t, f.Close())
	return f.Name()
}

func TestLoad(t *testing.T) {
	t.Run("should load a valid config", func(t *testing.T) {
		path := writeTemp(t, `
github:
  token_ref: "op://Personal/GH/token"
repositories:
  - owner: acme
    repo: frontend
policy:
  merge:
    allow: [patch, minor]
  ci_poll:
    timeout: 10m
    interval: 30s
`)
		cfg, err := Load(path)

		require.NoError(t, err)
		assert.Equal(t, "op://Personal/GH/token", cfg.GitHub.TokenRef)
		assert.Len(t, cfg.Repositories, 1)
		assert.Equal(t, "acme", cfg.Repositories[0].Owner)
		assert.Equal(t, "frontend", cfg.Repositories[0].Repo)
		assert.Equal(t, []string{"patch", "minor"}, cfg.Policy.Merge.Allow)
	})

	t.Run("should return error for missing token_ref", func(t *testing.T) {
		path := writeTemp(t, `
github:
  token_ref: ""
repositories:
  - owner: acme
    repo: frontend
policy:
  merge:
    allow: [patch, minor]
  ci_poll:
    timeout: 10m
    interval: 30s
`)
		_, err := Load(path)

		assert.ErrorContains(t, err, "token_ref")
	})

	t.Run("should return error for empty repositories list", func(t *testing.T) {
		path := writeTemp(t, `
github:
  token_ref: "op://Personal/GH/token"
repositories: []
policy:
  merge:
    allow: [patch, minor]
  ci_poll:
    timeout: 10m
    interval: 30s
`)
		_, err := Load(path)

		assert.ErrorContains(t, err, "repositories")
	})

	t.Run("should return error for invalid allow value", func(t *testing.T) {
		path := writeTemp(t, `
github:
  token_ref: "op://Personal/GH/token"
repositories:
  - owner: acme
    repo: frontend
policy:
  merge:
    allow: [patch, major]
  ci_poll:
    timeout: 10m
    interval: 30s
`)
		_, err := Load(path)

		assert.ErrorContains(t, err, "major")
	})

	t.Run("should return error when file does not exist", func(t *testing.T) {
		_, err := Load(filepath.Join(t.TempDir(), "nonexistent.yaml"))

		assert.Error(t, err)
	})
}
