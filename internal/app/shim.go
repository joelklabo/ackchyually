package app

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/joelklabo/ackchyually/internal/contextkey"
	"github.com/joelklabo/ackchyually/internal/execx"
	"github.com/joelklabo/ackchyually/internal/redact"
	"github.com/joelklabo/ackchyually/internal/store"
	"github.com/joelklabo/ackchyually/internal/toolid"
)

func RunShim(tool string, args []string) int {
	return runShim(tool, args, true)
}

func runShim(tool string, args []string, allowAutoExec bool) int {
	exe, err := execx.WhichSkippingShims(tool)
	if err != nil {
		fmt.Fprintln(os.Stderr, "ackchyually:", err)
		return 127
	}

	ctxKey := contextkey.Detect()
	ti, err := toolid.Identify(exe)
	if err != nil {
		ti = toolid.ToolIdentity{}
	}

	start := time.Now()
	res, err := execx.Run(exe, args)
	if err != nil {
		var ee *exec.ExitError
		if !errors.As(err, &ee) {
			fmt.Fprintln(os.Stderr, "ackchyually:", err)
		}
	}
	dur := time.Since(start)

	// redact argv before writing
	r := redact.Default()
	argvSafe := r.RedactArgs(append([]string{tool}, args...))
	stdoutTailSafe := r.RedactText(res.StdoutTail)
	stderrTailSafe := r.RedactText(res.StderrTail)
	combinedTailSafe := r.RedactText(res.CombinedTail)

	// best-effort logging
	if err := store.WithDB(func(db *store.DB) error {
		return db.InsertInvocation(store.Invocation{
			At:           start,
			DurationMS:   dur.Milliseconds(),
			ContextKey:   ctxKey,
			Tool:         tool,
			ExePath:      exe,
			ToolID:       ti.ID,
			ArgvJSON:     store.MustJSON(argvSafe),
			ExitCode:     res.ExitCode,
			Mode:         res.Mode,
			StdoutTail:   stdoutTailSafe,
			StderrTail:   stderrTailSafe,
			CombinedTail: combinedTailSafe,
		})
	}); err != nil {
		_ = err // best-effort
	}

	if isUsageish(res.ExitCode, res) {
		if allowAutoExec && autoExecKnownSuccessEnabled() && execx.IsTTY() {
			if code, ok := autoExecKnownSuccess(tool, ctxKey, argvSafe); ok {
				return code
			}
		}
		suggestKnownGood(tool, ctxKey, argvSafe)
	}

	return res.ExitCode
}

func isUsageish(code int, res execx.Result) bool {
	if code == 0 {
		return false
	}
	if code == 64 {
		return true
	}
	t := res.StdoutTail + res.StderrTail + res.CombinedTail
	return execx.ContainsFold(t, "usage:") ||
		execx.ContainsFold(t, "usage of") ||
		execx.ContainsFold(t, "flag provided but not defined") ||
		execx.ContainsFold(t, "unknown flag") ||
		execx.ContainsFold(t, "unknown shorthand flag") ||
		execx.ContainsFold(t, "unknown command") ||
		execx.ContainsFold(t, "unknown subcommand") ||
		execx.ContainsFold(t, "for usage") ||
		(execx.ContainsFold(t, "try '") && execx.ContainsFold(t, "--help")) ||
		(execx.ContainsFold(t, "is not a") && execx.ContainsFold(t, "command") && execx.ContainsFold(t, "--help")) ||
		(execx.ContainsFold(t, "option") && execx.ContainsFold(t, "is unknown")) ||
		execx.ContainsFold(t, "requires a value") ||
		execx.ContainsFold(t, "requires an argument") ||
		execx.ContainsFold(t, "requires parameter") ||
		execx.ContainsFold(t, "requires at least") ||
		execx.ContainsFold(t, "unknown option") ||
		execx.ContainsFold(t, "unrecognized option") ||
		execx.ContainsFold(t, "unrecognized argument") ||
		execx.ContainsFold(t, "invalid option") ||
		(execx.ContainsFold(t, "invalid --") && execx.ContainsFold(t, " option")) ||
		execx.ContainsFold(t, "invalid regexp") ||
		execx.ContainsFold(t, "not an integer") ||
		execx.ContainsFold(t, "key=value") ||
		execx.ContainsFold(t, "ambiguous argument") ||
		(execx.ContainsFold(t, "pathspec") && execx.ContainsFold(t, "did not match")) ||
		execx.ContainsFold(t, "wrong number of arguments") ||
		execx.ContainsFold(t, "name required") ||
		execx.ContainsFold(t, "unknown revision") ||
		execx.ContainsFold(t, "unknown date format") ||
		(execx.ContainsFold(t, "option") && execx.ContainsFold(t, "expects")) ||
		execx.ContainsFold(t, "needed a single revision") ||
		execx.ContainsFold(t, "missing required")
}

func pickKnownGood(cands []store.SuccessCandidate, argvSafe []string) []string {
	if len(argvSafe) == 0 {
		return nil
	}

	want := wantTokens(argvSafe)

	var best []string
	bestScore := 0
	bestLast := time.Time{}

	for _, c := range cands {
		if len(c.Argv) == 0 {
			continue
		}
		if slicesEqual(c.Argv, argvSafe) {
			continue
		}
		if containsRedacted(c.Argv) {
			continue
		}

		match := countArgMatches(want, c.Argv)
		if match == 0 {
			continue
		}
		prefix := commonPrefixLen(argvSafe, c.Argv)
		score := match*1000 + prefix*10 + minInt(c.Count, 50)

		if score > bestScore || (score == bestScore && c.Last.After(bestLast)) {
			best = c.Argv
			bestScore = score
			bestLast = c.Last
		}
	}

	return best
}

func wantTokens(argvSafe []string) []string {
	seen := make(map[string]struct{}, len(argvSafe))
	out := make([]string, 0, len(argvSafe))
	for _, a := range argvSafe[1:] { // exclude tool name
		for _, t := range tokenVariants(a) {
			if t == "" {
				continue
			}
			if _, ok := seen[t]; ok {
				continue
			}
			seen[t] = struct{}{}
			out = append(out, t)
		}
	}
	return out
}

func countArgMatches(want []string, argv []string) int {
	n := 0
	for _, a := range argv[1:] { // exclude tool name
		m := 0
		for _, t := range tokenVariants(a) {
			if matchesAnyToken(want, t) {
				m++
			}
		}
		// Avoid overweighting a single arg that matches in multiple ways.
		if m > 2 {
			m = 2
		}
		n += m
	}
	return n
}

func matchesAnyToken(tokens []string, argLower string) bool {
	for _, t := range tokens {
		if argLower == t || fuzzyTokenMatch(argLower, t) {
			return true
		}
	}
	return false
}

func fuzzyTokenMatch(a, b string) bool {
	na, oka := normalizeFuzzyToken(a)
	nb, okb := normalizeFuzzyToken(b)
	if !oka || !okb {
		return false
	}
	// Avoid overly-broad fuzzy matches for very short tokens (-m vs -n, etc.).
	if len(na) < 3 || len(nb) < 3 {
		return false
	}
	return isOneEditOrTransposition(na, nb)
}

func tokenVariants(arg string) []string {
	a := strings.ToLower(arg)
	out := []string{a}
	if i := strings.IndexByte(a, '='); i > 0 && i < len(a)-1 {
		out = append(out, a[:i], a[i+1:])
	} else if i := strings.IndexByte(a, '='); i > 0 {
		out = append(out, a[:i])
	}
	return out
}

func normalizeFuzzyToken(s string) (string, bool) {
	if isWordToken(s) {
		return s, true
	}
	if isFlagToken(s) {
		n := strings.TrimLeft(s, "-")
		n = strings.ReplaceAll(n, "-", "")
		if isWordToken(n) {
			return n, true
		}
	}
	return "", false
}

func isWordToken(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			continue
		}
		return false
	}
	return true
}

func isFlagToken(s string) bool {
	if s == "" || s == "-" {
		return false
	}
	if s[0] != '-' {
		return false
	}
	n := strings.TrimLeft(s, "-")
	if n == "" {
		return false
	}
	for _, r := range n {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			continue
		}
		return false
	}
	return true
}

func isOneEditOrTransposition(a, b string) bool {
	if a == b {
		return true
	}
	la, lb := len(a), len(b)
	if la == lb {
		// Single replacement or adjacent transposition.
		mismatch := make([]int, 0, 2)
		for i := 0; i < la; i++ {
			if a[i] != b[i] {
				mismatch = append(mismatch, i)
				if len(mismatch) > 2 {
					return false
				}
			}
		}
		switch len(mismatch) {
		case 1:
			return true
		case 2:
			i, j := mismatch[0], mismatch[1]
			return j == i+1 && a[i] == b[j] && a[j] == b[i]
		default:
			return false
		}
	}

	// Single insertion/deletion.
	if la+1 == lb {
		return isOneInsertAway(a, b) // a shorter
	}
	if lb+1 == la {
		return isOneInsertAway(b, a) // b shorter
	}
	return false
}

func isOneInsertAway(shorter, longer string) bool {
	i, j := 0, 0
	used := false
	for i < len(shorter) && j < len(longer) {
		if shorter[i] == longer[j] {
			i++
			j++
			continue
		}
		if used {
			return false
		}
		used = true
		j++ // skip one char in longer
	}
	return true
}

func commonPrefixLen(a, b []string) int {
	n := 0
	for n < len(a) && n < len(b) {
		if a[n] != b[n] {
			return n
		}
		n++
	}
	return n
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func suggestKnownGood(tool, ctxKey string, argvSafe []string) {
	if err := store.WithDB(func(db *store.DB) error {
		cands, err := db.ListSuccessCandidates(tool, ctxKey, 200)
		if err != nil {
			return err
		}
		argv := pickKnownGood(cands, argvSafe)
		if len(argv) == 0 {
			return nil
		}
		fmt.Fprintln(os.Stderr, "ackchyually: this worked before here:")
		fmt.Fprintln(os.Stderr, "  "+execx.ShellJoin(argv))
		return nil
	}); err != nil {
		_ = err // best-effort
	}
}

func autoExecKnownSuccessEnabled() bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv("ACKCHYUALLY_AUTO_EXEC")))
	return v == "known_success"
}

func autoExecKnownSuccess(tool, ctxKey string, argvSafe []string) (int, bool) {
	var cmd []string
	if err := store.WithDB(func(db *store.DB) error {
		cands, err := db.ListSuccessCandidates(tool, ctxKey, 200)
		if err != nil {
			return err
		}
		cmd = pickKnownGood(cands, argvSafe)
		return nil
	}); err != nil {
		return 0, false
	}

	if len(cmd) == 0 {
		return 0, false
	}
	if containsRedacted(cmd) {
		return 0, false
	}
	if slicesEqual(cmd, argvSafe) {
		return 0, false
	}

	fmt.Fprintln(os.Stderr, "ackchyually: auto-exec (known_success):")
	fmt.Fprintln(os.Stderr, "  "+execx.ShellJoin(cmd))
	return runShim(cmd[0], cmd[1:], false), true
}

func containsRedacted(argv []string) bool {
	for _, a := range argv {
		if strings.Contains(a, "<redacted>") {
			return true
		}
	}
	return false
}

func slicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
