package commands

import "drift/cli/output"

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
	return output.CoverageResult{Report: report}, 0
}

// D! id=ccov range-end

func (c CoverageCommand) Meta() Meta {
	return Meta{
		Name:  "coverage",
		Short: "Report spec coverage (lines covered by markers)",
		Usage: "Usage: drift coverage\n\nRead-only report of how many lines marker ranges cover.\nFiles with markers are listed individually; marker-free subtrees are\ncollapsed to one row. Totals split code vs markdown, plus the size of\nthe spec layer itself. Always exits 0 — use --json for build tools.\n\nExamples:\n  drift coverage\n  drift coverage --json",
	}
}
