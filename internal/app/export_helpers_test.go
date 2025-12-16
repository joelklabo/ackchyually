package app

import (
	"path/filepath"
	"testing"
)

func TestExportNormalizePath(t *testing.T) {
	home := "/Users/user"
	repo := "/Users/user/code/repo"

	tests := []struct {
		name     string
		path     string
		home     string
		repoRoot string
		want     string
	}{
		{
			name:     "empty",
			path:     "",
			home:     home,
			repoRoot: repo,
			want:     "",
		},
		{
			name:     "repo root",
			path:     repo,
			home:     home,
			repoRoot: repo,
			want:     ".",
		},
		{
			name:     "repo file",
			path:     filepath.Join(repo, "main.go"),
			home:     home,
			repoRoot: repo,
			want:     "./main.go",
		},
		{
			name:     "home dir",
			path:     home,
			home:     home,
			repoRoot: repo,
			want:     "~",
		},
		{
			name:     "home file",
			path:     filepath.Join(home, "config"),
			home:     home,
			repoRoot: repo,
			want:     "~/config",
		},
		{
			name:     "outside",
			path:     "/tmp/file",
			home:     home,
			repoRoot: repo,
			want:     "/tmp/file",
		},
		{
			name:     "no home or repo",
			path:     "/tmp/file",
			home:     "",
			repoRoot: "",
			want:     "/tmp/file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := exportNormalizePath(tt.path, tt.home, tt.repoRoot)
			if got != tt.want {
				t.Errorf("exportNormalizePath(%q, %q, %q) = %q, want %q", tt.path, tt.home, tt.repoRoot, got, tt.want)
			}
		})
	}
}
