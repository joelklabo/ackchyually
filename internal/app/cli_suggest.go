package app

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/joelklabo/ackchyually/internal/contextkey"
	"github.com/joelklabo/ackchyually/internal/execx"
	"github.com/joelklabo/ackchyually/internal/redact"
	"github.com/joelklabo/ackchyually/internal/store"
)

func printUnknownCommand(got string, candidates []string) {
	got = strings.TrimSpace(got)
	if got == "" {
		fmt.Fprintln(os.Stderr, "ackchyually: missing command")
		return
	}
	fmt.Fprintf(os.Stderr, "ackchyually: unknown command: %s\n", got)
	printSuggestion([]string{"ackchyually"}, got, candidates)
	printLastSuccessfulAckchyually()
	printAvailable("commands", candidates)
}

func printUnknownSubcommand(parent, got string, candidates []string) {
	got = strings.TrimSpace(got)
	if got == "" {
		fmt.Fprintf(os.Stderr, "ackchyually: missing %s subcommand\n", parent)
		return
	}
	fmt.Fprintf(os.Stderr, "ackchyually: unknown %s subcommand: %s\n", parent, got)
	printSuggestion([]string{"ackchyually", parent}, got, candidates)
	printLastSuccessfulAckchyually()
	printAvailable(parent+" subcommands", candidates)
}

func printSuggestion(prefix []string, got string, candidates []string) {
	s, ok := bestCommandMatch(got, candidates)
	if !ok {
		return
	}
	fmt.Fprintln(os.Stderr, "ackchyually: suggestion:")
	fmt.Fprintf(os.Stderr, "  %s\n", strings.Join(append(prefix, s), " "))
}

func printAvailable(label string, candidates []string) {
	if len(candidates) == 0 {
		return
	}
	sorted := append([]string{}, candidates...)
	sort.Strings(sorted)
	fmt.Fprintf(os.Stderr, "ackchyually: available %s: %s\n", label, strings.Join(sorted, ", "))
}

func bestCommandMatch(got string, candidates []string) (string, bool) {
	g := strings.ToLower(strings.TrimSpace(got))
	if g == "" {
		return "", false
	}

	best := ""
	bestScore := 0
	bestLen := 0

	for _, c := range candidates {
		cl := strings.ToLower(c)
		score := 0
		switch {
		case strings.HasPrefix(cl, g):
			score = 3
		case strings.HasPrefix(g, cl):
			score = 2
		case fuzzyTokenMatch(g, cl):
			score = 2
		}

		if score == 0 {
			continue
		}
		if score > bestScore || (score == bestScore && (best == "" || len(c) < bestLen)) {
			best = c
			bestScore = score
			bestLen = len(c)
		}
	}

	return best, bestScore > 0
}

func printLastSuccessfulAckchyually() {
	ctxKey := contextkey.Detect()
	r := redact.Default()

	if err := store.WithDB(func(db *store.DB) error {
		cmds, err := db.ListSuccessful("ackchyually", ctxKey, 1)
		if err != nil {
			return err
		}
		if len(cmds) == 0 {
			return nil
		}
		argv := r.RedactArgs(cmds[0])
		if len(argv) == 0 {
			return nil
		}
		fmt.Fprintln(os.Stderr, "ackchyually: last success here:")
		fmt.Fprintln(os.Stderr, "  "+execx.ShellJoin(argv))
		return nil
	}); err != nil {
		_ = err // best-effort
	}
}
