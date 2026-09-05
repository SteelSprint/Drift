package cli_test

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"drift/cli"
	"drift/cli/output"
	"drift/internal/testutil"
)

// TestSkill_ContainsDecisionTree asserts that `drift skill`'s output
// includes the Decision tree section referenced by cli.skill_decision_tree.
func TestSkill_ContainsDecisionTree(t *testing.T) {
	if !strings.Contains(cli.SkillContent, "## Decision tree") {
		t.Fatalf("skill.md missing '## Decision tree' section header")
	}
	// Spot-check that the table covers every event kind.
	for _, kind := range []string{
		"NODE_CHANGED", "NODE_ADDED", "NODE_REMOVED",
		"EDGE_ADDED", "EDGE_REMOVED", "EDGE_BROKEN",
	} {
		if !strings.Contains(cli.SkillContent, kind) {
			t.Fatalf("skill.md decision tree missing event kind %q", kind)
		}
	}
}

// TestSkill_ContainsMarkerPlacement asserts that `drift skill`'s output
// includes the Marker placement section.
func TestSkill_ContainsMarkerPlacement(t *testing.T) {
	if !strings.Contains(cli.SkillContent, "## Marker placement") {
		t.Fatalf("skill.md missing '## Marker placement' section header")
	}
}

// TestSkill_ContainsReportedClarifications asserts the guide sections added
// from field feedback: directed cycles (LLMs create them by default), the
// line-endings pinning note (content-addressed hashing + CRLF/LF churn), the
// scoped link contract (link never baselines spec content), and the
// review-terms batch-reset rule (no loops/scripts/pipes).
func TestSkill_ContainsReportedClarifications(t *testing.T) {
	for _, want := range []string{
		"## No directed cycles",
		"edge graph contains a directed cycle",
		"## Line endings",
		".gitattributes",
		"Link NEVER baselines spec content",
		"one closure per REVIEW",
		"Review your own closures",
	} {
		if !strings.Contains(cli.SkillContent, want) {
			t.Fatalf("skill.md missing expected content %q", want)
		}
	}
}

// TestCLI_ClosureWorkflow exercises the closure-driven UX end-to-end:
// init → link → drift → todo shows closures → reset by hash → clean.
func TestCLI_ClosureWorkflow(t *testing.T) {
	dir := t.TempDir()
	testutil.WriteSpecFile(t, dir, "main.drift.xml",
		`<module name="m">
<spec id="validate">Validate input.</spec>
</module>`)
	testutil.WriteCodeFile(t, dir, "code.go",
		"// D! id=cval range-start\npackage main\nfunc validate() {}\n// D! id=cval range-end\n")

	run := func(args ...string) (string, int) {
		return cli.RunWithRender(args, dir, output.PlainPresenter{})
	}

	// Init + link.
	if out, code := run("init"); code != 0 {
		t.Fatalf("init: code=%d out=%s", code, out)
	}
	if out, code := run("link", "cval", "m.validate"); code != 0 {
		t.Fatalf("link: code=%d out=%s", code, out)
	}

	// The new spec is pending review — link registers the marker only and
	// never baselines spec content (see orch.link). Review and reset the
	// NODE_ADDED closure to establish the baseline.
	out, code := run("todo")
	if code != 1 {
		t.Fatalf("new-spec todo: code=%d out=%s", code, out)
	}
	if out, code := run("reset", firstClosureHash(t, out)); code != 0 {
		t.Fatalf("baseline reset: code=%d out=%s", code, out)
	}

	// Baseline should be clean.
	out, code = run("todo")
	if code != 0 {
		t.Fatalf("clean todo: code=%d out=%s", code, out)
	}
	if !strings.Contains(out, "No changes detected") {
		t.Fatalf("clean todo: unexpected output: %s", out)
	}

	// Mutate the spec.
	testutil.WriteSpecFile(t, dir, "main.drift.xml",
		`<module name="m">
<spec id="validate">Validate input more strictly.</spec>
</module>`)

	// todo should show 1 closure.
	out, code = run("todo")
	if code != 1 {
		t.Fatalf("drifted todo: code=%d out=%s", code, out)
	}
	if !strings.Contains(out, "Closure") {
		t.Fatalf("expected closure output: %s", out)
	}

	// Reset by hash.
	out, code = run("reset", firstClosureHash(t, out))
	if code != 0 {
		t.Fatalf("reset: code=%d out=%s", code, out)
	}

	// todo should now be clean.
	out, code = run("todo")
	if code != 0 {
		t.Fatalf("post-reset todo: code=%d out=%s", code, out)
	}
	if !strings.Contains(out, "No changes detected") {
		t.Fatalf("post-reset todo should be clean: %s", out)
	}
}

// TestCLI_JSONTodo: JSON output structure for closures.
func TestCLI_JSONTodo(t *testing.T) {
	dir := t.TempDir()
	testutil.WriteSpecFile(t, dir, "main.drift.xml",
		`<module name="m">
<spec id="a">A spec.</spec>
</module>`)
	testutil.WriteCodeFile(t, dir, "code.go",
		"// D! id=ca range-start\npackage main\n// D! id=ca range-end\n")

	run := func(args ...string) (string, int) {
		return cli.RunWithRender(args, dir, output.JSONPresenter{})
	}
	if _, code := run("init"); code != 0 {
		t.Fatal("init failed")
	}
	if _, code := run("link", "ca", "m.a"); code != 0 {
		t.Fatal("link failed")
	}

	// Drift the spec.
	testutil.WriteSpecFile(t, dir, "main.drift.xml",
		`<module name="m">
<spec id="a">A spec that changed.</spec>
</module>`)

	out, code := run("todo")
	if code != 1 {
		t.Fatalf("todo: code=%d out=%s", code, out)
	}
	if !strings.Contains(out, `"closures"`) {
		t.Fatalf("JSON missing closures field: %s", out)
	}
	if !strings.Contains(out, `"hash"`) {
		t.Fatalf("JSON missing hash field: %s", out)
	}
}

// TestCLI_ListEdgesSorted verifies the list command emits edges in
// (From, To) lexicographic order — required for diff-stable output
// across runs.
func TestCLI_ListEdgesSorted(t *testing.T) {
	dir := t.TempDir()
	testutil.WriteSpecFile(t, dir, "main.drift.xml",
		`<module name="m">
<spec id="a">A.</spec>
<spec id="b">B.</spec>
<spec id="c">C.</spec>
</module>`)
	testutil.WriteCodeFile(t, dir, "code.go",
		"// D! id=za range-start\npackage main\n// D! id=za range-end\n"+
			"// D! id=aa range-start\nvar _ = 1\n// D! id=aa range-end\n"+
			"// D! id=ma range-start\nvar _ = 2\n// D! id=ma range-end\n")

	run := func(args ...string) (string, int) {
		return cli.RunWithRender(args, dir, output.PlainPresenter{})
	}
	if _, code := run("init"); code != 0 {
		t.Fatalf("init: code=%d", code)
	}
	// Link in non-alphabetical order to expose any storage-order dependence.
	for _, pair := range [][2]string{
		{"za", "m.c"}, // spec C, marker za
		{"aa", "m.a"}, // spec A, marker aa
		{"ma", "m.b"}, // spec B, marker ma
	} {
		if out, code := run("link", pair[0], pair[1]); code != 0 {
			t.Fatalf("link %v: code=%d\n%s", pair, code, out)
		}
	}

	out, code := run("list")
	if code != 0 {
		t.Fatalf("list: code=%d\n%s", code, out)
	}

	// Extract the Edges block.
	lines := strings.Split(out, "\n")
	var edgeLines []string
	inEdges := false
	for _, l := range lines {
		if strings.HasPrefix(l, "Edges (") {
			inEdges = true
			continue
		}
		if inEdges {
			if l == "" || !strings.HasPrefix(l, "  ") {
				break
			}
			edgeLines = append(edgeLines, strings.TrimSpace(l))
		}
	}
	if len(edgeLines) != 3 {
		t.Fatalf("expected 3 edge lines, got %d: %v", len(edgeLines), edgeLines)
	}
	// Expected order (by From, To):
	//   aa → m.a
	//   ma → m.b
	//   za → m.c
	want := []string{"aa", "ma", "za"}
	for i, w := range want {
		if !strings.HasPrefix(edgeLines[i], w+" ") {
			t.Fatalf("edge %d = %q, expected prefix %q\nfull edge block:\n%s",
				i, edgeLines[i], w, strings.Join(edgeLines, "\n"))
		}
	}
}

// TestCLI_DiffSeedLabel verifies that drift diff annotates each node with
// [SEED] or [citer] based on whether it originated the closure.
func TestCLI_DiffSeedLabel(t *testing.T) {
	dir := t.TempDir()
	testutil.WriteSpecFile(t, dir, "main.drift.xml",
		`<module name="m">
<spec id="a">A spec.</spec>
</module>`)
	testutil.WriteCodeFile(t, dir, "code.go",
		"// D! id=ca range-start\npackage main\n// D! id=ca range-end\n")
	run := func(args ...string) (string, int) {
		return cli.RunWithRender(args, dir, output.PlainPresenter{})
	}
	if _, code := run("init"); code != 0 {
		t.Fatal("init failed")
	}
	if _, code := run("link", "ca", "m.a"); code != 0 {
		t.Fatal("link failed")
	}

	// Drift the spec. Marker ca is a citer of m.a (via the link edge),
	// so when m.a drifts, m.a is the SEED and ca is the citer.
	testutil.WriteSpecFile(t, dir, "main.drift.xml",
		`<module name="m">
<spec id="a">A spec that changed.</spec>
</module>`)

	out, code := run("todo")
	if code != 1 {
		t.Fatalf("todo: code=%d out=%s", code, out)
	}
	hashLine := ""
	for _, l := range strings.Split(out, "\n") {
		if strings.Contains(l, "Closure ") {
			hashLine = l
			break
		}
	}
	parts := strings.Fields(hashLine)
	if len(parts) < 2 {
		t.Fatalf("could not parse closure hash from: %q", hashLine)
	}
	hash := parts[1]

	out, code = run("diff", hash)
	if code != 0 {
		t.Fatalf("diff: code=%d out=%s", code, out)
	}
	// Expect marker ca labeled as [citer] (it cites spec m.a, was pulled in).
	// Expect spec m.a labeled as [SEED] (it's the seed of the closure).
	if !strings.Contains(out, `Spec: m.a [SEED]`) {
		t.Fatalf("expected 'Spec: m.a [SEED]' in diff output:\n%s", out)
	}
	if !strings.Contains(out, `Marker: ca [citer]`) {
		t.Fatalf("expected 'Marker: ca [citer]' in diff output:\n%s", out)
	}
}

// TestCLI_ResetDryRun verifies:
// - Output contains the change-summary lines (preview).
// - state.xml is NOT modified — next todo still reports drift.
// - Exit code is 3 (special dry-run code).
func TestCLI_ResetDryRun(t *testing.T) {
	dir := t.TempDir()
	testutil.WriteSpecFile(t, dir, "main.drift.xml",
		`<module name="m">
<spec id="a">A spec.</spec>
</module>`)
	testutil.WriteCodeFile(t, dir, "code.go",
		"// D! id=ca range-start\npackage main\n// D! id=ca range-end\n")
	run := func(args ...string) (string, int) {
		return cli.RunWithRender(args, dir, output.PlainPresenter{})
	}
	if _, code := run("init"); code != 0 {
		t.Fatal("init failed")
	}
	if _, code := run("link", "ca", "m.a"); code != 0 {
		t.Fatal("link failed")
	}
	// Drift the spec.
	testutil.WriteSpecFile(t, dir, "main.drift.xml",
		`<module name="m">
<spec id="a">A spec that changed.</spec>
</module>`)
	out, code := run("todo")
	if code != 1 {
		t.Fatalf("todo: code=%d", code)
	}
	hashLine := ""
	for _, l := range strings.Split(out, "\n") {
		if strings.Contains(l, "Closure ") {
			hashLine = l
			break
		}
	}
	parts := strings.Fields(hashLine)
	if len(parts) < 2 {
		t.Fatalf("could not parse hash from: %q", hashLine)
	}
	hash := parts[1]

	// Snapshot state.xml to detect mutation.
	statePath := dir + "/.drift/state.xml"
	beforeBytes, err := os.ReadFile(statePath)
	if err != nil {
		t.Fatal(err)
	}

	out, code = run("reset", "--dry-run", hash)
	if code != 3 {
		t.Fatalf("dry-run reset: expected exit 3, got %d\n%s", code, out)
	}
	if !strings.Contains(out, "Preview — no changes written") {
		t.Fatalf("missing preview banner:\n%s", out)
	}
	if !strings.Contains(out, hash) {
		t.Fatalf("missing closure hash in preview:\n%s", out)
	}
	if !strings.Contains(out, "m.a") {
		t.Fatalf("missing spec id in preview:\n%s", out)
	}
	if !strings.Contains(out, "changed") {
		t.Fatalf("missing 'changed' kind in preview:\n%s", out)
	}

	// state.xml must be unchanged.
	afterBytes, err := os.ReadFile(statePath)
	if err != nil {
		t.Fatal(err)
	}
	if string(beforeBytes) != string(afterBytes) {
		t.Fatalf("dry-run mutated state.xml (before=%d bytes, after=%d bytes)", len(beforeBytes), len(afterBytes))
	}

	// Next todo still reports drift.
	out, code = run("todo")
	if code != 1 {
		t.Fatalf("after dry-run: expected todo to still drift (code=%d)\n%s", code, out)
	}
}

// TestCLI_ResetSummary verifies that reset prints the change-summary lines
// after applying (same shape as the dry-run preview).
func TestCLI_ResetSummary(t *testing.T) {
	dir := t.TempDir()
	testutil.WriteSpecFile(t, dir, "main.drift.xml",
		`<module name="m">
<spec id="a">A spec.</spec>
</module>`)
	testutil.WriteCodeFile(t, dir, "code.go",
		"// D! id=ca range-start\npackage main\n// D! id=ca range-end\n")
	run := func(args ...string) (string, int) {
		return cli.RunWithRender(args, dir, output.PlainPresenter{})
	}
	run("init")
	run("link", "ca", "m.a")
	testutil.WriteSpecFile(t, dir, "main.drift.xml",
		`<module name="m">
<spec id="a">A spec that changed.</spec>
</module>`)
	out, _ := run("todo")
	hashLine := ""
	for _, l := range strings.Split(out, "\n") {
		if strings.Contains(l, "Closure ") {
			hashLine = l
			break
		}
	}
	parts := strings.Fields(hashLine)
	hash := parts[1]

	out, code := run("reset", hash)
	if code != 0 {
		t.Fatalf("reset: code=%d\n%s", code, out)
	}
	if !strings.Contains(out, "Closure "+hash+" resolved") {
		t.Fatalf("missing 'Closure HASH resolved' line:\n%s", out)
	}
	if !strings.Contains(out, "m.a") {
		t.Fatalf("missing spec id in post-reset summary:\n%s", out)
	}
	if !strings.Contains(out, "changed") {
		t.Fatalf("missing 'changed' kind in post-reset summary:\n%s", out)
	}

	// Verify baseline actually updated.
	out, code = run("todo")
	if code != 0 {
		t.Fatalf("post-reset todo should be clean: code=%d\n%s", code, out)
	}
}

// TestCLI_LinkDryRun verifies link --dry-run previews, doesn't save, exits 3.
func TestCLI_LinkDryRun(t *testing.T) {
	dir := t.TempDir()
	testutil.WriteSpecFile(t, dir, "main.drift.xml",
		`<module name="m">
<spec id="a">A.</spec>
</module>`)
	testutil.WriteCodeFile(t, dir, "code.go",
		"// D! id=ca range-start\npackage main\n// D! id=ca range-end\n")
	run := func(args ...string) (string, int) {
		return cli.RunWithRender(args, dir, output.PlainPresenter{})
	}
	run("init")

	statePath := dir + "/.drift/state.xml"
	before, _ := os.ReadFile(statePath)

	out, code := run("link", "--dry-run", "ca", "m.a")
	if code != 3 {
		t.Fatalf("dry-run link: expected exit 3, got %d\n%s", code, out)
	}
	if !strings.Contains(out, "Preview — no changes written") {
		t.Fatalf("missing preview banner:\n%s", out)
	}
	if !strings.Contains(out, "ca") || !strings.Contains(out, "m.a") {
		t.Fatalf("missing edge info in preview:\n%s", out)
	}

	after, _ := os.ReadFile(statePath)
	if string(before) != string(after) {
		t.Fatalf("dry-run link mutated state.xml")
	}

	// Real link still works.
	_, code = run("link", "ca", "m.a")
	if code != 0 {
		t.Fatalf("real link after dry-run: code=%d", code)
	}
}

// TestCLI_UnlinkDryRun verifies unlink --dry-run previews, doesn't save, exits 3.
func TestCLI_UnlinkDryRun(t *testing.T) {
	dir := t.TempDir()
	testutil.WriteSpecFile(t, dir, "main.drift.xml",
		`<module name="m">
<spec id="a">A.</spec>
</module>`)
	testutil.WriteCodeFile(t, dir, "code.go",
		"// D! id=ca range-start\npackage main\n// D! id=ca range-end\n")
	run := func(args ...string) (string, int) {
		return cli.RunWithRender(args, dir, output.PlainPresenter{})
	}
	run("init")
	run("link", "ca", "m.a")

	statePath := dir + "/.drift/state.xml"
	before, _ := os.ReadFile(statePath)

	out, code := run("unlink", "--dry-run", "ca", "m.a")
	if code != 3 {
		t.Fatalf("dry-run unlink: expected exit 3, got %d\n%s", code, out)
	}
	if !strings.Contains(out, "Preview — no changes written") {
		t.Fatalf("missing preview banner:\n%s", out)
	}

	after, _ := os.ReadFile(statePath)
	if string(before) != string(after) {
		t.Fatalf("dry-run unlink mutated state.xml")
	}
}

// TestCLI_LinkPreservesPendingSpecDrift reproduces the reported defect
// end-to-end: establish a clean baseline, edit a spec, confirm `drift todo`
// flags it, then create a NEW marker and link it to that spec. The link must
// not absorb the pending edit — the NODE_CHANGED closure must survive (todo
// still exits 1). Pre-fix, link baselined the spec's content and todo
// reported clean, exit 0.
func TestCLI_LinkPreservesPendingSpecDrift(t *testing.T) {
	dir := t.TempDir()
	testutil.WriteSpecFile(t, dir, "main.drift.xml",
		`<module name="m">
<spec id="x">X behavior.</spec>
</module>`)
	testutil.WriteCodeFile(t, dir, "code.go",
		"// D! id=cx range-start\npackage main\n// D! id=cx range-end\n")

	run := func(args ...string) (string, int) {
		return cli.RunWithRender(args, dir, output.PlainPresenter{})
	}
	runJSON := func(args ...string) (string, int) {
		return cli.RunWithRender(args, dir, output.JSONPresenter{})
	}

	if _, code := run("init"); code != 0 {
		t.Fatal("init failed")
	}
	if _, code := run("link", "cx", "m.x"); code != 0 {
		t.Fatal("link failed")
	}

	// Establish a reviewed baseline. Post-fix, a newly linked spec stays
	// pending (NODE_ADDED) until reviewed — review and reset it. Pre-fix the
	// link already baselined the spec, so todo is clean here and this is a
	// no-op.
	out, code := runJSON("todo")
	if code != 0 && code != 1 {
		t.Fatalf("todo: code=%d\n%s", code, out)
	}
	if code == 1 {
		var parsed struct {
			Closures []struct {
				Hash string `json:"hash"`
			} `json:"closures"`
		}
		if err := json.Unmarshal([]byte(out), &parsed); err != nil {
			t.Fatalf("json parse: %v\n%s", err, out)
		}
		for _, c := range parsed.Closures {
			if out, code := run("reset", c.Hash); code != 0 {
				t.Fatalf("reset %s: code=%d\n%s", c.Hash, code, out)
			}
		}
	}
	if out, code := run("todo"); code != 0 {
		t.Fatalf("baseline todo not clean: code=%d\n%s", code, out)
	}

	// Edit the spec — pending, never reviewed.
	testutil.WriteSpecFile(t, dir, "main.drift.xml",
		`<module name="m">
<spec id="x">X behavior. Additional sentence.</spec>
</module>`)
	out, code = run("todo")
	if code != 1 {
		t.Fatalf("pending edit not flagged: code=%d\n%s", code, out)
	}

	// Create a NEW marker in a new file (the report's step 4)...
	testutil.WriteCodeFile(t, dir, "code2.go",
		"// D! id=cm range-start\npackage main\n// D! id=cm range-end\n")

	// ...and link it to the spec carrying the pending edit (step 5).
	out, code = run("link", "cm", "m.x")
	if code != 0 {
		t.Fatalf("link: code=%d\n%s", code, out)
	}

	// The pending edit must survive the link (step 6).
	out, code = run("todo")
	if code != 1 {
		t.Fatalf("link absorbed the pending spec edit (todo exit %d, want 1):\n%s", code, out)
	}
	if !strings.Contains(out, "NODE-CHANGED") || !strings.Contains(out, "m.x") {
		t.Fatalf("expected surviving NODE_CHANGED closure for m.x:\n%s", out)
	}
}

// firstClosureHash extracts the first closure hash from plain `drift todo`
// output ("Closure abc12345  (...)"). Fails the test when no hash is found.
func firstClosureHash(t *testing.T, out string) string {
	t.Helper()
	for _, l := range strings.Split(out, "\n") {
		if strings.Contains(l, "Closure ") {
			parts := strings.SplitN(l, " ", 3)
			if len(parts) >= 2 {
				return parts[1]
			}
		}
	}
	t.Fatalf("could not extract closure hash from output:\n%s", out)
	return ""
}

// TestCLI_LinkPreservesPendingRefEdge reproduces the report's "wider blast
// radius": a pending, never-reviewed spec-spec ref (EDGE_ADDED closure) must
// survive a link whose target spec is clean. Pre-fix, link merged ALL
// scanned spec-spec edges into baseline and the pending ref was never
// reviewed.
func TestCLI_LinkPreservesPendingRefEdge(t *testing.T) {
	dir := t.TempDir()
	testutil.WriteSpecFile(t, dir, "main.drift.xml",
		`<module name="m">
<spec id="a">A behavior.</spec>
<spec id="b">B behavior.</spec>
</module>`)
	testutil.WriteCodeFile(t, dir, "code.go",
		"// D! id=ca range-start\npackage main\n// D! id=ca range-end\n")
	testutil.WriteCodeFile(t, dir, "code2.go",
		"// D! id=cb range-start\npackage main\n// D! id=cb range-end\n")

	run := func(args ...string) (string, int) {
		return cli.RunWithRender(args, dir, output.PlainPresenter{})
	}
	runJSON := func(args ...string) (string, int) {
		return cli.RunWithRender(args, dir, output.JSONPresenter{})
	}

	if _, code := run("init"); code != 0 {
		t.Fatal("init failed")
	}
	if _, code := run("link", "ca", "m.a"); code != 0 {
		t.Fatal("link ca failed")
	}
	if _, code := run("link", "cb", "m.b"); code != 0 {
		t.Fatal("link cb failed")
	}

	// Establish a reviewed baseline (reset the NODE_ADDED closures for both
	// newly linked specs).
	out, code := runJSON("todo")
	if code != 0 && code != 1 {
		t.Fatalf("todo: code=%d\n%s", code, out)
	}
	if code == 1 {
		var parsed struct {
			Closures []struct {
				Hash string `json:"hash"`
			} `json:"closures"`
		}
		if err := json.Unmarshal([]byte(out), &parsed); err != nil {
			t.Fatalf("json parse: %v\n%s", err, out)
		}
		for _, c := range parsed.Closures {
			if out, code := run("reset", c.Hash); code != 0 {
				t.Fatalf("reset %s: code=%d\n%s", c.Hash, code, out)
			}
		}
	}
	if out, code := run("todo"); code != 0 {
		t.Fatalf("baseline todo not clean: code=%d\n%s", code, out)
	}

	// Add a ref in m.b citing m.a — a pending, unreviewed EDGE_ADDED. The
	// self-closing form is stripped entirely from the canonical hash, so the
	// ref edge is the ONLY pending drift.
	testutil.WriteSpecFile(t, dir, "main.drift.xml",
		`<module name="m">
<spec id="a">A behavior.</spec>
<spec id="b">B behavior.<ref spec="m.a" /></spec>
</module>`)
	out, code = run("todo")
	if code != 1 {
		t.Fatalf("pending ref not flagged: code=%d\n%s", code, out)
	}

	// Create a NEW marker and link it to the CLEAN spec m.a.
	testutil.WriteCodeFile(t, dir, "code3.go",
		"// D! id=cn range-start\npackage main\n// D! id=cn range-end\n")
	out, code = run("link", "cn", "m.a")
	if code != 0 {
		t.Fatalf("link: code=%d\n%s", code, out)
	}

	// The pending ref closure must survive the link.
	out, code = run("todo")
	if code != 1 {
		t.Fatalf("link absorbed the pending ref edge (todo exit %d, want 1):\n%s", code, out)
	}
	if !strings.Contains(out, "EDGE-ADDED") || !strings.Contains(out, "m.b") {
		t.Fatalf("expected surviving EDGE_ADDED closure for m.b:\n%s", out)
	}
}
