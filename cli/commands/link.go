package commands

import (
	"fmt"

	"drift/cli/output"
)

// LinkCommand implements `drift link <marker> <module.spec>`.
type LinkCommand struct{}

// D! id=clfmt range-start
func (c LinkCommand) Run(ctx Context) (output.Result, int) {
	if len(ctx.Args) < 3 {
		return output.ErrorResult{
			Command: "link",
			Message: "usage:\n  drift link <marker> <module.spec>\n  drift link --dry-run <marker> <module.spec>\n\nExample: drift link validate_input core.validate_input",
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
			Command: "link",
			Message: "usage:\n  drift link <marker> <module.spec>\n  drift link --dry-run <marker> <module.spec>\n\nExample: drift link validate_input core.validate_input",
			Exit:    1,
		}, 1
	}
	markerID, specID := positional[0], positional[1]

	if dryRun {
		summary, err := ctx.Orch.PreviewLink(ctx.Sess, markerID, specID)
		if err != nil {
			return output.ErrorResult{Command: "link", Message: err.Error(), Exit: 1}, 1
		}
		return output.ChangeSummaryResult{Summary: summary, Preview: true}, 3
	}
	summary, err := ctx.Orch.LinkWithSummary(ctx.Sess, markerID, specID)
	if err != nil {
		return output.ErrorResult{Command: "link", Message: err.Error(), Exit: 1}, 1
	}
	return output.ChangeSummaryResult{
		Summary: summary,
		Preview: false,
		Message: fmt.Sprintf("Linked marker %q to spec %q", markerID, specID),
	}, 0
}

// D! id=clfmt range-end
func (c LinkCommand) Meta() Meta {
	return Meta{
		Name:  "link",
		Short: "Connect a marker to a spec",
		Usage: "Usage:\n  drift link <marker> <module.spec>                 Create the link (exit 0)\n  drift link --dry-run <marker> <module.spec>    Preview the change summary without writing (exit 3)\n\nConnect a marker to a spec. Both must exist on disk.\n\nExamples:\n  drift link validate_input core.validate_input\n  drift link --dry-run validate_input core.validate_input",
		Flags: []string{"--dry-run"},
	}
}
