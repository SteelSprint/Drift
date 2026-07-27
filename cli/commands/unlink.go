package commands

import (
	"fmt"

	"drift/cli/output"
)

// UnlinkCommand implements `drift unlink <marker> <module.spec>`.
type UnlinkCommand struct{}

// D! id=cunlnk range-start
func (c UnlinkCommand) Run(ctx Context) (output.Result, int) {
	if len(ctx.Args) < 3 {
		return output.ErrorResult{
			Command: "unlink",
			Message: "usage:\n  drift unlink <marker> <module.spec>\n  drift unlink --dry-run <marker> <module.spec>\n\nExample: drift unlink validate_input core.validate_input",
			Exit:    1,
		}, 1
	}
	var dryRun bool
	var positional []string
	for _, arg := range ctx.Args[1:] {
		if arg == "--dry-run" {
			dryRun = true
		} else {
			positional = append(positional, arg)
		}
	}
	if len(positional) < 2 {
		return output.ErrorResult{
			Command: "unlink",
			Message: "usage:\n  drift unlink <marker> <module.spec>\n  drift unlink --dry-run <marker> <module.spec>\n\nExample: drift unlink validate_input core.validate_input",
			Exit:    1,
		}, 1
	}
	markerID, specID := positional[0], positional[1]

	if dryRun {
		summary, err := ctx.Orch.PreviewUnlink(ctx.Sess, markerID, specID)
		if err != nil {
			return output.ErrorResult{Command: "unlink", Message: err.Error(), Exit: 1}, 1
		}
		return output.ChangeSummaryResult{Summary: summary, Preview: true}, 3
	}
	summary, err := ctx.Orch.UnlinkWithSummary(ctx.Sess, markerID, specID)
	if err != nil {
		return output.ErrorResult{Command: "unlink", Message: err.Error(), Exit: 1}, 1
	}
	return output.ChangeSummaryResult{
		Summary: summary,
		Preview: false,
		Message: fmt.Sprintf("Unlinked marker %q from spec %q", markerID, specID),
	}, 0
}

// D! id=cunlnk range-end
func (c UnlinkCommand) Meta() Meta {
	return Meta{
		Name:  "unlink",
		Short: "Remove a link between a marker and a spec",
		Usage: "Usage:\n  drift unlink <marker> <module.spec>               Remove the link (exit 0)\n  drift unlink --dry-run <marker> <module.spec>  Preview the change summary without writing (exit 3)\n\nRemove a link between a marker and a spec.\n\nExamples:\n  drift unlink validate_input core.validate_input\n  drift unlink --dry-run validate_input core.validate_input",
		Flags: []string{"--dry-run"},
	}
}
