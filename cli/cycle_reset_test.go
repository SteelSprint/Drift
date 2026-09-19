package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"drift/cli/output"
	"drift/internal/testutil"
)

// TestCycleFormingReset_Refused: a reset whose EDGE_ADDED events would create
// a directed cycle in the baseline edge graph MUST be refused — baseline
// unchanged, clear cycle error, and todo still functional afterwards.
//
// Regression background: reset used to validate the post-reset edge graph
// after writing it. A cycle-forming edge landed in .drift/state.xml; every
// subsequent command (todo, reset, list) validated the baseline graph, hit
// the cycle, and refused. Since closures are derived from the baseline
// graph, no EDGE_REMOVED closure could ever be derived to repair it — the
// only escape was hand-editing state.xml. See principles.red_before_green
// and the core.reset_action spec.
func TestCycleFormingReset_Refused(t *testing.T) {
	dir := t.TempDir()

	testutil.WriteSpecFile(t, dir, "main.drift.xml",
		`<main>
  <import path="alpha.drift.xml" />
  <import path="beta.drift.xml" />
</main>`)
	testutil.WriteSpecFile(t, dir, "alpha.drift.xml",
		`<module name="alpha">
<spec id="spec_a">Alpha describes the base rule.</spec>
</module>`)
	testutil.WriteSpecFile(t, dir, "beta.drift.xml",
		`<module name="beta">
<spec id="spec_b">Beta is cited by alpha.</spec>
</module>`)

	run := func(args ...string) (string, int) {
		return RunWithRender(args, dir, output.PlainPresenter{})
	}

	firstHash := func(out string) string {
		t.Helper()
		i := strings.Index(out, "Closure ")
		if i < 0 {
			t.Fatalf("no closure in output:\n%s", out)
		}
		return out[i+len("Closure "):][:8]
	}

	if out, code := run("init"); code != 0 {
		t.Fatalf("init: code=%d out=%s", code, out)
	}

	// Baseline the two specs (NODE_ADDED orphans).
	out, _ := run("todo")
	if out, code := run("reset", "--dangerously-override-friction", firstHash(out)); code != 0 {
		t.Fatalf("reset alpha: code=%d out=%s", code, out)
	}
	out, _ = run("todo")
	if out, code := run("reset", "--dangerously-override-friction", firstHash(out)); code != 0 {
		t.Fatalf("reset beta: code=%d out=%s", code, out)
	}

	// alpha cites beta → EDGE_ADDED → reset → baseline edge alpha→beta.
	testutil.WriteSpecFile(t, dir, "alpha.drift.xml",
		`<module name="alpha">
<spec id="spec_a">Alpha describes the base rule. See <ref spec="beta.spec_b">beta</ref>.</spec>
</module>`)
	out, _ = run("todo")
	if out, code := run("reset", "--dangerously-override-friction", firstHash(out)); code != 0 {
		t.Fatalf("reset alpha-cites-beta: code=%d out=%s", code, out)
	}
	if out, code := run("todo"); code == 2 || strings.Contains(out, "closure(s) with drift") {
		t.Fatalf("setup: unexpected todo state, code=%d out=%s", code, out)
	}

	statePath := filepath.Join(dir, ".drift", "state.xml")
	baselineBefore, err := os.ReadFile(statePath)
	if err != nil {
		t.Fatal(err)
	}

	// THE BUG: beta cites alpha, closing the cycle alpha→beta→alpha.
	// todo derives the closure; reset must REFUSE to write it.
	testutil.WriteSpecFile(t, dir, "beta.drift.xml",
		`<module name="beta">
<spec id="spec_b">Beta cites alpha. See <ref spec="alpha.spec_a">alpha</ref>.</spec>
</module>`)
	out, code := run("todo")
	if code != 1 {
		t.Fatalf("pre-reset todo: code=%d out=%s", code, out)
	}
	cycleHash := firstHash(out)

	out, code = run("reset", "--dangerously-override-friction", cycleHash)
	if code == 0 {
		t.Fatalf("cycle-forming reset must be refused, got exit 0:\n%s", out)
	}
	if !strings.Contains(out, "cycle") {
		t.Fatalf("refusal must name the cycle, got:\n%s", out)
	}

	// Baseline must be unchanged — the cycle edge must NOT be persisted.
	baselineAfter, err := os.ReadFile(statePath)
	if err != nil {
		t.Fatal(err)
	}
	if string(baselineAfter) != string(baselineBefore) {
		t.Fatalf("cycle-forming reset mutated state.xml")
	}

	// todo must still work: no deadlock. The closure stays pending; the
	// user fixes the spec instead.
	out, code = run("todo")
	if code != 1 {
		t.Fatalf("todo after refused reset: code=%d out=%s", code, out)
	}

	// Repair path: remove the ref. Because the refused reset never wrote
	// the edge, the scan and baseline are already back in sync — no closure,
	// no deadlock, no hand-editing of state.xml.
	testutil.WriteSpecFile(t, dir, "beta.drift.xml",
		`<module name="beta">
<spec id="spec_b">Beta is cited by alpha.</spec>
</module>`)
	out, code = run("todo")
	if code == 2 || strings.Contains(out, "closure(s) with drift") {
		t.Fatalf("post-repair todo should have no closures, code=%d out=%s", code, out)
	}
}
