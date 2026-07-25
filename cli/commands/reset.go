package commands

import (
	"fmt"
	"os"

	"drift/cli/output"
)

// ResetCommand implements `drift reset <hash>`: sync the closure's seed
// events into baseline. Closures containing only broken-edge events are
// refused (require scan fix).
type ResetCommand struct{}

// D! id=crfmt range-start
func (c ResetCommand) Run(ctx Context) (output.Result, int) {
	args := ctx.Args
	if len(args) < 2 {
		return output.ErrorResult{
			Command: "reset",
			Message: "usage:\n  drift reset <hash>     Sync the closure's seed events into baseline\n  drift reset --dry-run <hash>   Preview the change summary without writing\n  drift reset --dangerously-override-friction <hash>   Bypass the rate-limit block\n\nExample: drift reset a3f7b2c1",
			Exit:    1,
		}, 1
	}
	var hash string
	var dryRun bool
	var overrideFriction bool
	switch args[1] {
	case "--dry-run":
		if len(args) < 3 {
			return output.ErrorResult{
				Command: "reset",
				Message: "usage: drift reset --dry-run <hash>",
				Exit:    1,
			}, 1
		}
		dryRun = true
		hash = args[2]
	case "--dangerously-override-friction":
		if len(args) < 3 {
			return output.ErrorResult{
				Command: "reset",
				Message: "usage: drift reset --dangerously-override-friction <hash>",
				Exit:    1,
			}, 1
		}
		overrideFriction = true
		hash = args[2]
	default:
		hash = args[1]
	}
	// D! id=cnobulk range-start
	if dryRun {
		summary, err := ctx.Orch.PreviewResetClosure(ctx.Sess, hash)
		if err != nil {
			return output.ErrorResult{Command: "reset", Message: err.Error(), Exit: 1}, 1
		}
		return output.ChangeSummaryResult{
			Summary: summary,
			Preview: true,
		}, 3
	}
	if !overrideFriction {
		if err := checkFriction(ctx.Sess); err != nil {
			return output.ErrorResult{Command: "reset", Message: err.Error(), Exit: 2}, 2
		}
	} else {
		// Override squawk: stderr-only line so JSON output stays a clean
		// machine-consumable structure. The warning surfaces in JSON via
		// ChangeSummaryResult.Warning.
		fmt.Fprint(os.Stderr, frictionOverrideSquawk)
	}
	evaluated, summary, err := ctx.Orch.ResetClosureWithSummary(ctx.Sess, hash)
	if err != nil {
		return output.ErrorResult{Command: "reset", Message: err.Error(), Exit: 1}, 1
	}
	_ = evaluated
	if err := recordReset(ctx.Sess); err != nil {
		// Recording failure MUST NOT fail the reset — the baseline mutation
		// already succeeded. Log to stderr and continue.
		fmt.Fprintf(os.Stderr, "warning: failed to record friction telemetry: %s\n", err.Error())
	}
	result := output.ChangeSummaryResult{
		Summary:  summary,
		Preview:  false,
		Message:  fmt.Sprintf("Closure %s resolved.", hash),
	}
	if overrideFriction {
		result.Warning = frictionWarningField
	}
	return result, 0
	// D! id=cnobulk range-end
}

// D! id=crfmt range-end
func (c ResetCommand) Meta() Meta {
	return Meta{
		Name:  "reset",
		Short: "Resolve a drift closure by syncing baseline to scan",
		Usage: "Usage:\n  drift reset <hash>           Resolve a closure by syncing its seed events into baseline (exit 0)\n  drift reset --dry-run <hash>  Preview the change summary without writing; exits 3 (LLM signal: no change applied)\n  drift reset --dangerously-override-friction <hash>  Bypass the rate-limit block (see cli.reset_friction_block); intended for tests and CI\n\nThe hash is the 8-character closure ID printed by `drift todo`.\nClosures containing only broken-edge events are refused (fix the scan instead).\n\nRate limit: drift blocks the 4th reset within any 30-second window. The block enforces per-closure review (the friction principle); the override exists for tests and CI but is not advertised in error output.\n\nExamples:\n  drift reset a3f7b2c1\n  drift reset --dry-run a3f7b2c1",
		Flags: []string{"--dry-run", "--dangerously-override-friction"},
	}
}
