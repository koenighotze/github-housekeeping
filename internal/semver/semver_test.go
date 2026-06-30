package semver

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClassifyBump(t *testing.T) {
	tests := []struct {
		title   string
		want    BumpKind
		wantErr bool
	}{
		{
			title: "Bump golang.org/x/net from 0.25.0 to 0.26.0",
			want:  Minor,
		},
		{
			title: "Bump github.com/pkg/errors from 0.9.1 to 0.9.2",
			want:  Patch,
		},
		{
			title: "Bump go from 1.22 to 1.23",
			want:  Minor,
		},
		{
			title: "Bump github.com/spf13/cobra from 1.8.1 to 2.0.0",
			want:  Major,
		},
		{
			title: "Bump lodash from 4.17.20 to 4.17.21",
			want:  Patch,
		},
		{
			title: "Bump react from 17.0.2 to 18.0.0",
			want:  Major,
		},
		{
			title: "chore(deps): bump golang.org/x/net from 0.25.0 to 0.26.0",
			want:  Minor,
		},
		{
			title:   "fix: unrelated commit message",
			wantErr: true,
		},
		{
			title:   "",
			wantErr: true,
		},
		{
			title:   "Bump foo from notasemver to 1.0.0",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.title, func(t *testing.T) {
			got, err := ClassifyBump(tt.title)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestBumpKindString(t *testing.T) {
	assert.Equal(t, "patch", Patch.String())
	assert.Equal(t, "minor", Minor.String())
	assert.Equal(t, "major", Major.String())
}
