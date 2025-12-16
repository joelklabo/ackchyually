package app

import (
	"strings"
	"testing"
	"time"
)

func TestSuggestKnownGood(t *testing.T) {
	ctxKey := setTempHomeAndCWD(t)
	now := time.Now()

	// Seed data
	seedInvocation(t, ctxKey, "git", []string{"git", "status"}, now.Add(-time.Minute), 0)
	seedInvocation(t, ctxKey, "git", []string{"git", "commit"}, now.Add(-time.Hour), 0)

	// Use a typo that is long enough for fuzzy matching (>= 3 chars)
	code, _, errOut := captureStdoutStderr(t, func() int {
		suggestKnownGood("git", ctxKey, []string{"git", "statu"})
		return 0
	})

	if code != 0 {
		t.Errorf("suggestKnownGood returned %d, want 0", code)
	}
	if !strings.Contains(errOut, "ackchyually: suggestion") {
		t.Errorf("output missing suggestion header, got:\n%s", errOut)
	}
	if !strings.Contains(errOut, "git status") {
		t.Errorf("output missing 'git status', got:\n%s", errOut)
	}
}

func TestSuggestKnownGood_NoGoodMatch(t *testing.T) {
	ctxKey := setTempHomeAndCWD(t)
	t.Setenv("ACKCHYUALLY_TEST_FORCE_TTY", "true")
	now := time.Now()

	// Seed a candidate that is very different
	seedInvocation(t, ctxKey, "git", []string{"git", "commit"}, now, 0)

	// Call with "git status" (very different from commit)
	code, _, errOut := captureStdoutStderr(t, func() int {
		suggestKnownGood("git", ctxKey, []string{"git", "status"})
		return 0
	})

	if code != 0 {
		t.Errorf("suggestKnownGood returned %d want 0", code)
	}
	// Should call suggestNoKnownGood
	if !strings.Contains(errOut, "no known-good git command saved") {
		t.Errorf("expected no known-good message, got:\n%s", errOut)
	}
}
