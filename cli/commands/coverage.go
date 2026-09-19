package commands

import (
	"fmt"
	"os"
	"strings"

	"drift/cli/output"
)

// CoverageCommand implements `drift coverage` — a read-only report of how
// many lines marker ranges cover. It never signals drift via exit code
// (always 0 on success); drift status remains `drift todo`'s job.
type CoverageCommand struct{}

// D! id=ccov range-start
func (c CoverageCommand) Run(ctx Context) (output.Result, int) {
	report, err := ctx.Orch.Coverage(ctx.Sess)
	if err != nil {
		return output.ErrorResult{Command: "coverage", Message: err.Error(), Exit: 1}, 1
	}

	// Coverage policy (.drift/coverage.xml): optional, committed. Missing
	// file means today's behavior. Present file means the verdict renders
	// on every run; --check turns a shortfall into exit 1 (unfinished work).
	checkRequested := hasFlag(ctx.Args, "--check")
	data, readErr := ctx.Sess.Read(coverageFileName)
	hasConfig := readErr == nil
	if readErr != nil && !os.IsNotExist(readErr) {
		return output.ErrorResult{Command: "coverage", Message: readErr.Error(), Exit: 1}, 1
	}

	var check *output.CoverageCheckResult
	if hasConfig {
		cfg, parseErr := ParseCoverageConfig(data)
		if parseErr != nil {
			return output.ErrorResult{Command: "coverage", Message: parseErr.Error(), Exit: 1}, 1
		}
		failures := EvaluateCoverageCheck(cfg, report)
		check = &output.CoverageCheckResult{Target: cfg.Target, Failures: failures}
	} else if checkRequested {
		return output.ErrorResult{
			Command: "coverage",
			Message: "coverage --check requires .drift/coverage.xml (not present).\n\nCreate it with a target, e.g.:\n\n  <coverage>\n    <target lines=\"80\"/>\n  </coverage>\n\nThen drift coverage --check exits 1 while coverage is below target.",
			Exit:    1,
		}, 1
	}

	if checkRequested && check != nil && !check.Passed() {
		var parts []string
		for _, f := range check.Failures {
			parts = append(parts, fmt.Sprintf("  %s: %d%% vs target %d%%", f.Group, f.Actual, f.Target))
		}
		return output.ErrorResult{
			Command: "coverage",
			Message: fmt.Sprintf("coverage check FAILED (target %d%%):\n%s", check.Target, strings.Join(parts, "\n")),
			Exit:    1,
		}, 1
	}

	return output.CoverageResult{Report: report, Check: check}, 0
}

// D! id=ccov range-end

func (c CoverageCommand) Meta() Meta {
	return Meta{
		Lock:  LockRequire,
		Name:  "coverage",
		Short: "Report spec coverage (lines covered by markers)",
		Flags: []string{"--check"},
		Usage: "Usage: drift coverage [--check]\n\nRead-only report of how many lines marker ranges cover.\nFiles with markers are listed individually; marker-free subtrees are\ncollapsed to one row. Totals split code vs markdown, plus the size of\nthe spec layer itself. Informational runs always exit 0.\n\nCoverage policy: if .drift/coverage.xml exists (committed), a verdict\nline shows the target and pass/fail on every run. --check exits 1\nwhile coverage is below target — a build gate alongside drift todo.\n\nSettings file (optional, committed):\n  <coverage>\n    <target lines=\"80\"/>\n    <path pattern=\"cmd/\" target=\"90\"/>\n  </coverage>\n\nPer-path targets apply to files under the pattern prefix; the most\nspecific pattern wins; unmatched files use the global target.\n\nExamples:\n  drift coverage\n  drift coverage --check\n  drift coverage --json",
	}
}
