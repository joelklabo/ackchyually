package helpcount

import "testing"

func TestIsHelpInvocation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		argv []string
		want bool
	}{
		{
			name: "subcommand_help_counts",
			argv: []string{"git", "help", "log"},
			want: true,
		},
		{
			name: "flag_help_counts",
			argv: []string{"git", "log", "-h"},
			want: true,
		},
		{
			name: "flag_double_dash_help_counts",
			argv: []string{"git", "log", "--help"},
			want: true,
		},
		{
			name: "help_word_in_arg_does_not_count",
			argv: []string{"git", "commit", "-m", "help"},
			want: false,
		},
		{
			name: "help_word_not_first_subcommand_does_not_count",
			argv: []string{"git", "commit", "help"},
			want: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := isHelpInvocation(tt.argv); got != tt.want {
				t.Fatalf("isHelpInvocation(%q)=%v want %v", tt.argv, got, tt.want)
			}
		})
	}
}

func TestMatchesHelpInvocation(t *testing.T) {
	t.Parallel()

	tool := "git"
	helpArgs := []string{"log", "-h"}

	if !matchesHelpInvocation(tool, helpArgs, []string{"git", "log", "-h"}) {
		t.Fatalf("expected exact help argv to match")
	}
	if matchesHelpInvocation(tool, helpArgs, []string{"git", "help", "log"}) {
		t.Fatalf("did not expect alternative help form to match when helpArgs are specified")
	}
	if !matchesHelpInvocation(tool, nil, []string{"git", "help", "log"}) {
		t.Fatalf("expected any-help matcher to match git help")
	}
}
