package app

import (
	"fmt"
	"os"
	"sort"
	"strings"
)

func printUnknownCommand(got string, candidates []string) {
	got = strings.TrimSpace(got)
	if got == "" {
		fmt.Fprintln(os.Stderr, "ackchyually: missing command")
		return
	}
	fmt.Fprintf(os.Stderr, "ackchyually: unknown command: %s\n", got)
	printSuggestion([]string{"ackchyually"}, got, candidates)
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
