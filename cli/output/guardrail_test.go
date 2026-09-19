package output

import (
	"regexp"
	"testing"

	"drift/orchestrator"
)

// stripANSI removes SGR escape sequences (ESC [ ... m). The guardrail
// property (output.guardrail_property) says stripping Color output must
// equal Plain output byte-for-byte — this helper is the enforcement tool
// the spec names. See AUDIT_FINDINGS.md A1: the tests promised by
// output_impl.guardrail_test never existed; this file adds them.
var ansiRe = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func stripANSI(s string) string { return ansiRe.ReplaceAllString(s, "") }

// assertGuardrail is the spot-check: Plain and stripped-Color must agree.
// It uses Render — the same dispatch the production path uses — so the
// spot-checks cannot drift from reality.
func assertGuardrail(t *testing.T, name string, r Result) {
	t.Helper()
	plain := Render(PlainPresenter{}, r)
	color := Render(ColorPresenter{Theme: DefaultTheme}, r)
	if plain != stripANSI(color) {
		t.Errorf("guardrail violated for %s:\n plain: %q\n color: %q (stripped)", name, plain, stripANSI(color))
	}
}

// TestGuardrail_ChangeSummary: node + edge change summaries (all three node
// kinds) must strip-equal across presenters. Regression: Plain padded kinds
// to width 8 (%-8s), Color did not — stripped outputs differed by spaces
// (AUDIT_FINDINGS.md A1).
func TestGuardrail_ChangeSummary(t *testing.T) {
	r := ChangeSummaryResult{
		Message: "Closure a1b2c3d4 resolved.",
		Summary: orchestrator.ChangeSummary{
			Operation: "resolve closure a1b2c3d4",
			NodeChanges: []orchestrator.NodeChange{
				{Kind: "changed", ID: "m.spec", OldHash: "aaaa1111", NewHash: "bbbb2222"},
				{Kind: "added", ID: "m.new", OldHash: "", NewHash: "cccc3333"},
				{Kind: "removed", ID: "m.gone", OldHash: "dddd4444", NewHash: ""},
			},
			EdgeChanges: []orchestrator.EdgeChange{
				{Kind: "added", From: "m.a", To: "m.b"},
				{Kind: "removed", From: "m.c", To: "m.d"},
			},
		},
	}
	assertGuardrail(t, "ChangeSummaryResult", r)
}

// TestGuardrail_CoverageCheck: the coverage verdict line + per-path failure
// lines (added in the coverage-policy feature) must strip-equal.
func TestGuardrail_CoverageCheck(t *testing.T) {
	report := orchestrator.CoverageReport{}
	r := CoverageResult{
		Report: report,
		Check: &CoverageCheckResult{
			Target: 80,
			Failures: []CoverageFailure{
				{Group: "(repo)", Actual: 63, Target: 80},
				{Group: "internal/legacy/", Actual: 12, Target: 40},
			},
		},
	}
	assertGuardrail(t, "CoverageResult+Check", r)

	r.Check = nil
	assertGuardrail(t, "CoverageResult", r)
}

// TestGuardrail_SimpleResults: the text-shaped results.
func TestGuardrail_SimpleResults(t *testing.T) {
	assertGuardrail(t, "OkResult", OkResult{Command: "init", Message: "Initialized .drift/"})
	assertGuardrail(t, "ErrorResult", ErrorResult{Command: "todo", Message: "project not initialized here. Run 'drift init' first.", Exit: 2})
	assertGuardrail(t, "TextResult", TextResult{Text: "some guide text"})
	assertGuardrail(t, "VersionResult", VersionResult{Version: "1.5.0"})
}
