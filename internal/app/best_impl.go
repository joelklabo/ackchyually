package app

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/joelklabo/ackchyually/internal/contextkey"
	"github.com/joelklabo/ackchyually/internal/execx"
	"github.com/joelklabo/ackchyually/internal/redact"
	"github.com/joelklabo/ackchyually/internal/store"
)

type scored struct {
	Argv  []string
	Score float64
}

func bestImpl(tool, query string) int {
	ctxKey := contextkey.Detect()
	qTokens := tokenize(query)

	var cands []store.SuccessCandidate
	if err := store.WithDB(func(db *store.DB) error {
		var err error
		cands, err = db.ListSuccessCandidates(tool, ctxKey, 200)
		return err
	}); err != nil {
		fmt.Fprintln(os.Stderr, "ackchyually:", err)
		return 1
	}

	if len(cands) == 0 {
		fmt.Fprintln(os.Stderr, "ackchyually: no successful commands recorded yet for this tool/context")
		return 1
	}

	var scoredList []scored
	for _, c := range cands {
		cmdStr := strings.ToLower(execx.ShellJoin(c.Argv))
		match := countMatches(cmdStr, qTokens)

		if len(qTokens) > 0 && match == 0 {
			continue
		}

		score := 0.0
		score += math.Log1p(float64(c.Count)) * 100.0
		ageH := time.Since(c.Last).Hours()
		score += 150.0 / (1.0 + ageH/24.0)
		score += float64(match) * 250.0

		scoredList = append(scoredList, scored{Argv: c.Argv, Score: score})
	}

	if len(scoredList) == 0 {
		for _, c := range cands {
			score := math.Log1p(float64(c.Count))*100.0 + 150.0/(1.0+time.Since(c.Last).Hours()/24.0)
			scoredList = append(scoredList, scored{Argv: c.Argv, Score: score})
		}
	}

	sort.Slice(scoredList, func(i, j int) bool { return scoredList[i].Score > scoredList[j].Score })

	n := 5
	if len(scoredList) < n {
		n = len(scoredList)
	}
	for i := 0; i < n; i++ {
		fmt.Println(execx.ShellJoin(scoredList[i].Argv))
	}
	return 0
}

func tokenize(s string) []string {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" {
		return nil
	}
	return strings.Fields(s)
}

func countMatches(hay string, tokens []string) int {
	n := 0
	for _, t := range tokens {
		if strings.Contains(hay, t) {
			n++
		}
	}
	return n
}

func tagAdd(args []string) int {
	if len(args) < 3 {
		fmt.Fprintln(os.Stderr, `usage: ackchyually tag add "<tag>" -- <command...>`)
		return 2
	}
	tag := args[0]
	i := -1
	for idx, a := range args {
		if a == "--" {
			i = idx
			break
		}
	}
	if i == -1 || i+1 >= len(args) {
		fmt.Fprintln(os.Stderr, `usage: ackchyually tag add "<tag>" -- <command...>`)
		return 2
	}
	argv := args[i+1:]
	tool := argv[0]

	ctxKey := contextkey.Detect()
	err := store.WithDB(func(db *store.DB) error {
		return db.UpsertTag(store.Tag{
			ContextKey: ctxKey,
			Tag:        tag,
			Tool:       tool,
			ArgvJSON:   store.MustJSON(argv),
		})
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "ackchyually:", err)
		return 1
	}
	return 0
}

func tagRun(args []string) int {
	if len(args) != 1 {
		fmt.Fprintln(os.Stderr, `usage: ackchyually tag run "<tag>"`)
		return 2
	}
	tag := args[0]
	ctxKey := contextkey.Detect()

	var tr store.Tag
	err := store.WithDB(func(db *store.DB) error {
		got, err := db.GetTag(ctxKey, tag)
		if err != nil {
			return err
		}
		tr = got
		return nil
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "ackchyually: tag not found:", tag)
		return 1
	}

	var argv []string
	if err := json.Unmarshal([]byte(tr.ArgvJSON), &argv); err != nil {
		fmt.Fprintln(os.Stderr, "ackchyually: corrupt tag argv")
		return 1
	}
	if len(argv) == 0 {
		fmt.Fprintln(os.Stderr, "ackchyually: corrupt tag argv")
		return 1
	}

	return RunShim(argv[0], argv[1:])
}

func exportImpl(format, tool string) int {
	ctxKey := contextkey.Detect()
	r := redact.Default()
	home, err := os.UserHomeDir()
	if err != nil {
		home = os.Getenv("HOME")
	}
	repoRoot := exportRepoRoot(ctxKey)
	ctxKeyExport := exportNormalizeContextKey(ctxKey, home)

	err = store.WithDB(func(db *store.DB) error {
		tags, err := db.ListTags(ctxKey, tool)
		if err != nil {
			return err
		}

		switch format {
		case "json":
			type out struct {
				Context string      `json:"context"`
				Tags    []exportTag `json:"tags"`
			}

			var outTags []exportTag
			for _, tg := range tags {
				argv := exportDecodeArgv(tg.ArgvJSON)
				argv = exportNormalizeArgv(argv, home, repoRoot)
				argv = r.RedactArgs(argv)
				if len(argv) == 0 {
					continue
				}
				outTags = append(outTags, exportTag{Tag: tg.Tag, Tool: argv[0], Argv: argv})
			}

			o := out{Context: ctxKeyExport, Tags: outTags}
			b, err := json.MarshalIndent(o, "", "  ")
			if err != nil {
				return err
			}
			fmt.Println(string(b))
			return nil

		case "md":
			fmt.Printf("## ackchyually export\n\n")
			fmt.Printf("- Context: `%s`\n\n", ctxKeyExport)

			if len(tags) > 0 {
				fmt.Println("### Tags")
				for _, tg := range tags {
					argv := exportDecodeArgv(tg.ArgvJSON)
					argv = exportNormalizeArgv(argv, home, repoRoot)
					argv = r.RedactArgs(argv)
					if len(argv) == 0 {
						continue
					}
					fmt.Printf("- **%s**: `%s`\n", tg.Tag, execx.ShellJoin(argv))
				}
				fmt.Println()
			}

			fmt.Println("### Recent successful commands")
			if tool != "" {
				cmds, err := db.ListSuccessful(tool, ctxKey, 10)
				if err != nil {
					return err
				}
				for _, argv := range cmds {
					argv = exportNormalizeArgv(argv, home, repoRoot)
					fmt.Printf("- `%s`\n", execx.ShellJoin(r.RedactArgs(argv)))
				}
			} else {
				fmt.Println("_Tip: pass `--tool <tool>` to export successful commands._")
			}
			return nil
		default:
			return fmt.Errorf("export: unknown format: %s", format)
		}
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "ackchyually:", err)
		return 2
	}
	return 0
}

type exportTag struct {
	Tag  string   `json:"tag"`
	Tool string   `json:"tool"`
	Argv []string `json:"argv"`
}

func exportDecodeArgv(argvJSON string) []string {
	var argv []string
	if err := json.Unmarshal([]byte(argvJSON), &argv); err != nil {
		return nil
	}
	return argv
}

func exportRepoRoot(ctxKey string) string {
	prefix, path, ok := strings.Cut(ctxKey, ":")
	if !ok {
		return ""
	}
	if prefix != "git" {
		return ""
	}
	return filepath.Clean(path)
}

func exportNormalizeContextKey(ctxKey, home string) string {
	prefix, path, ok := strings.Cut(ctxKey, ":")
	if !ok {
		return ctxKey
	}
	path = exportSanitizeValue(path, home, "")
	return prefix + ":" + path
}

func exportNormalizeArgv(argv []string, home, repoRoot string) []string {
	if len(argv) == 0 {
		return argv
	}
	out := make([]string, 0, len(argv))
	for _, a := range argv {
		out = append(out, exportSanitizeArg(a, home, repoRoot))
	}
	return out
}

func exportNormalizePath(s, home, repoRoot string) string {
	sep := string(filepath.Separator)
	if repoRoot != "" {
		repoRoot = filepath.Clean(repoRoot)
		if s == repoRoot {
			return "."
		}
		if strings.HasPrefix(s, repoRoot+sep) {
			return "." + sep + strings.TrimPrefix(s, repoRoot+sep)
		}
	}

	if home != "" {
		home = filepath.Clean(home)
		if s == home {
			return "~"
		}
		if strings.HasPrefix(s, home+sep) {
			return "~" + sep + strings.TrimPrefix(s, home+sep)
		}
	}
	return s
}
