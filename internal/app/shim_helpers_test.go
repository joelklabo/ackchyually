package app

import (
	"reflect"
	"testing"
)

func TestTokenVariants(t *testing.T) {
	tests := []struct {
		in   string
		want []string
	}{
		{"foo", []string{"foo"}},
		{"FOO", []string{"foo"}},
		{"-n5", []string{"-n5", "-n", "5"}},
		{"-n", []string{"-n"}},
		{"--foo=bar", []string{"--foo=bar", "--foo", "bar"}},
		{"--foo=", []string{"--foo=", "--foo"}},
		{"--foo", []string{"--foo"}},
	}

	for _, tt := range tests {
		got := tokenVariants(tt.in)
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("tokenVariants(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
}

func TestIsAttachedNumericShortFlag(t *testing.T) {
	tests := []struct {
		in   string
		want bool
	}{
		{"-n5", true},
		{"-n", false},
		{"--n5", false},
		{"-5", false}, // must be letter then digit?
		{"-a0", true},
		{"-z9", true},
		{"-A5", false},  // lowercase only?
		{"-n5a", false}, // digits only after char
	}

	for _, tt := range tests {
		got := isAttachedNumericShortFlag(tt.in)
		if got != tt.want {
			t.Errorf("isAttachedNumericShortFlag(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
}

func TestIsOneEditOrTransposition(t *testing.T) {
	tests := []struct {
		s1, s2 string
		want   bool
	}{
		{"abc", "abc", true},  // 0 edits
		{"abc", "ab", true},   // 1 deletion
		{"abc", "abcd", true}, // 1 insertion
		{"abc", "abd", true},  // 1 substitution
		{"abc", "acb", true},  // 1 transposition
		{"abc", "bac", true},  // 1 transposition
		{"abc", "def", false},
		{"abc", "abde", false},
	}
	for _, tt := range tests {
		if got := isOneEditOrTransposition(tt.s1, tt.s2); got != tt.want {
			t.Errorf("isOneEditOrTransposition(%q, %q) = %v, want %v", tt.s1, tt.s2, got, tt.want)
		}
	}
}

func TestIsFlagToken_EdgeCases(t *testing.T) {
	tests := []struct {
		in   string
		want bool
	}{
		{"", false},
		{"-", false},
		{"--", false},
		{"-a", true},
		{"--foo", true},
		{"-1", true},
		{"-!", false},     // invalid char
		{"--foo!", false}, // invalid char
	}
	for _, tt := range tests {
		if got := isFlagToken(tt.in); got != tt.want {
			t.Errorf("isFlagToken(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
}

func TestWantTokens_Duplicates(t *testing.T) {
	// wantTokens should deduplicate tokens
	argv := []string{"tool", "-v", "-v"}
	got := wantTokens(argv)
	// -v produces "-v"
	// so we expect just ["-v"]
	if len(got) != 1 || got[0] != "-v" {
		t.Errorf("wantTokens(%v) = %v, want [-v]", argv, got)
	}

	// Test empty token variant (if any)
	// tokenVariants returns empty string? No, it returns at least original.
	// But let's check if we can trigger the "continue" in wantTokens
}

func TestIsWordToken_EdgeCases(t *testing.T) {
	tests := []struct {
		in   string
		want bool
	}{
		{"", false},
		{"abc", true},
		{"123", true},
		{"a1", true},
		{"-", false},
		{"!", false},
		{"a!", false},
	}
	for _, tt := range tests {
		if got := isWordToken(tt.in); got != tt.want {
			t.Errorf("isWordToken(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
}

func TestCountArgMatches_Cap(t *testing.T) {
	// countArgMatches caps match score at 2 per arg
	// We need an arg that matches multiple tokens in 'want'
	// e.g. arg "-n5" produces tokens "-n5", "-n", "5"
	// If 'want' contains all of them, we get 3 matches, but it should be capped at 2.

	want := []string{"-n5", "-n", "5"}
	argv := []string{"tool", "-n5"}

	got := countArgMatches(want, argv)
	if got != 2 {
		t.Errorf("countArgMatches(%v, %v) = %d, want 2 (capped)", want, argv, got)
	}
}
