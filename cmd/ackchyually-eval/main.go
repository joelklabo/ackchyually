package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/joelklabo/ackchyually/internal/eval/helpcount"
)

func main() {
	os.Exit(run())
}

func run() int {
	var (
		mode           = flag.String("mode", string(helpcount.ModeCompare), "baseline|memory|compare")
		scenarioFilter = flag.String("scenario", "", "run scenarios whose name/description contains this substring")
		jsonOut        = flag.Bool("json", false, "output JSON")
	)
	flag.Parse()

	repoRoot, err := repoRoot()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}

	r, err := helpcount.NewRunner(repoRoot)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	defer r.Close()

	report, err := r.Run(helpcount.Options{
		Mode:           helpcount.Mode(*mode),
		ScenarioFilter: *scenarioFilter,
		JSON:           *jsonOut,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(report); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 2
		}
		return 0
	}

	printReport(report)
	return 0
}

func printReport(r helpcount.Report) {
	fmt.Printf("ackchyually eval (helpcount)\n")
	fmt.Printf("started: %s\n", r.StartedAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("ended:   %s\n\n", r.EndedAt.Format("2006-01-02 15:04:05"))

	for _, s := range r.Results {
		fmt.Printf("- %s: %s\n", s.Name, s.Description)
		if s.Baseline != nil {
			fmt.Printf("  baseline: success=%v steps=%d help=%d suggestion=%v\n",
				s.Baseline.Success, s.Baseline.Steps, s.Baseline.HelpInvocations, s.Baseline.SuggestionPrinted)
			if s.Baseline.Error != "" {
				fmt.Printf("    error: %s\n", s.Baseline.Error)
			}
		}
		if s.Memory != nil {
			fmt.Printf("  memory:   success=%v steps=%d help=%d suggestion=%v\n",
				s.Memory.Success, s.Memory.Steps, s.Memory.HelpInvocations, s.Memory.SuggestionPrinted)
			if s.Memory.Error != "" {
				fmt.Printf("    error: %s\n", s.Memory.Error)
			}
		}
	}
}

func repoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("could not find repo root (go.mod)")
		}
		dir = parent
	}
}
