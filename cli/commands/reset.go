package commands

import (
	"fmt"

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
			Message: "usage:\n  drift reset <hash>     Sync the closure's seed events into baseline\n\nExample: drift reset a3f7b2c1",
			Exit:    1,
		}, 1
	}
	hash := args[1]
	// D! id=cnobulk range-start
	_, err := ctx.Orch.ResetClosure(hash)
	if err != nil {
		return output.ErrorResult{Command: "reset", Message: err.Error(), Exit: 1}, 1
	}
	return output.OkResult{
		Command: "reset",
		Message: fmt.Sprintf("Closure %s resolved. Baseline updated.", hash),
	}, 0
	// D! id=cnobulk range-end
}

// D! id=crfmt range-end
func (c ResetCommand) Meta() Meta {
	return Meta{
		Name:  "reset",
		Short: "Resolve a drift closure by syncing baseline to scan",
		Usage: "Usage:\n  drift reset <hash>     Resolve a closure by syncing its seed events into baseline\n\nThe hash is the 8-character closure ID printed by `drift todo`.\nClosures containing only broken-edge events are refused (fix the scan instead).\n\nExample:\n  drift reset a3f7b2c1",
	}
}
