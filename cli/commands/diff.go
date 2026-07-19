package commands

import (
	"drift/cli/output"
)

// DiffCommand implements `drift diff`:
//   - drift diff <hash>     Show diff for all nodes in a closure
//   - drift diff --all      Show diffs for ALL closures
type DiffCommand struct{}

// D! id=cdiff range-start
func (c DiffCommand) Run(ctx Context) (output.Result, int) {
	args := ctx.Args
	if len(args) < 2 {
		return output.ErrorResult{
			Command: "diff",
			Message: "usage:\n  drift diff <hash>\n  drift diff --all\n\nExample: drift diff a3f7b2c1\n         drift diff --all",
			Exit:    1,
		}, 1
	}
	if args[1] == "--all" {
		closures, state, err := ctx.Orch.DiffAll()
		if err != nil {
			return output.ErrorResult{Command: "diff", Message: err.Error(), Exit: 1}, 1
		}
		return output.DiffAllResult{State: state, Closures: closures}, 0
	}
	hash := args[1]
	diffs, err := ctx.Orch.DiffClosure(hash)
	if err != nil {
		return output.ErrorResult{Command: "diff", Message: err.Error(), Exit: 1}, 1
	}
	return output.DiffClosureResult{Hash: hash, Diffs: diffs}, 0
}

// D! id=cdiff range-end
func (c DiffCommand) Meta() Meta {
	return Meta{
		Name:  "diff",
		Short: "Show what changed in a closure (or all closures)",
		Usage: "Usage:\n  drift diff <hash>     Show unified diff for every node in the closure\n  drift diff --all      Show diffs for ALL closures at once\n\nThe hash is the 8-character closure ID printed by `drift todo`.\n\nExamples:\n  drift diff a3f7b2c1\n  drift diff --all",
		Flags: []string{"--all"},
	}
}
