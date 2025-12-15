package app

import (
	"testing"
	"time"

	"github.com/joelklabo/ackchyually/internal/store"
)

func TestPickKnownGood_PrefersSpecificFlagMatchOverPlain(t *testing.T) {
	now := time.Date(2025, 12, 15, 0, 0, 0, 0, time.UTC)

	argvBad := []string{"bd", "list", "--jsn"} //nolint:misspell // intentional typo scenario
	cands := []store.SuccessCandidate{
		{Argv: []string{"bd", "list"}, Count: 100, Last: now},
		{Argv: []string{"bd", "list", "--json"}, Count: 1, Last: now.Add(-time.Hour)},
	}

	got := pickKnownGood(cands, argvBad)
	want := []string{"bd", "list", "--json"}

	if !slicesEqual(got, want) {
		t.Fatalf("pickKnownGood=%q want %q", got, want)
	}
}

func TestPickKnownGood_MatchesAttachedNumericShortFlag(t *testing.T) {
	now := time.Date(2025, 12, 15, 0, 0, 0, 0, time.UTC)

	argvBad := []string{"attachttool", "run", "-n", "1", "--badflag"}
	cands := []store.SuccessCandidate{
		{Argv: []string{"attachttool", "run"}, Count: 50, Last: now},
		{Argv: []string{"attachttool", "run", "-n1"}, Count: 1, Last: now.Add(-time.Hour)},
	}

	got := pickKnownGood(cands, argvBad)
	want := []string{"attachttool", "run", "-n1"}

	if !slicesEqual(got, want) {
		t.Fatalf("pickKnownGood=%q want %q", got, want)
	}
}
