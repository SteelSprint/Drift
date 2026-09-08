package cli_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"drift/cli"
	"drift/cli/output"
	"drift/internal/testutil"
)

// setupCoverageProject builds a small drift project:
//   - main.drift.xml with three specs (two linked, one orphan)
//   - code.go with a linked marker (3 interior lines)
//   - doc.md with a linked marker (1 interior line)
//   - plain.go and a clean subtree (lib/) with no markers
// and baselines everything so todo is clean.
func setupCoverageProject(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	testutil.WriteSpecFile(t, dir, "main.drift.xml",
		`<module name="m">
<spec id="validate">Validate input.</spec>
<spec id="doc">Document the validation flow.</spec>
<spec id="orphan">Nobody wraps this one.</spec>
</module>`)
	testutil.WriteCodeFile(t, dir, "code.go",
		"package main\n"+
			"// D! id=cval range-start\n"+
			"func validate() {}\n"+
			"func helper() {}\n"+
			"func helper2() {}\n"+
			"// D! id=cval range-end\n"+
			"// tail\n")
	testutil.WriteCodeFile(t, dir, "plain.go", "package main\n\nfunc plain() {}\n")
	if err := os.Mkdir(filepath.Join(dir, "lib"), 0755); err != nil {
		t.Fatal(err)
	}
	testutil.WriteCodeFile(t, filepath.Join(dir, "lib"), "util.go", "package lib\n")
	testutil.WriteSpecFile(t, dir, "doc.md",
		"# Doc\n"+
			"<!-- D! id=cdoc range-start -->\n"+
			"prose line\n"+
			"<!-- D! id=cdoc range-end -->\n"+
			"tail\n")

	run := func(args ...string) (string, int) {
		return cli.RunWithRender(args, dir, output.PlainPresenter{})
	}
	if out, code := run("init"); code != 0 {
		t.Fatalf("init: code=%d out=%s", code, out)
	}
	if out, code := run("link", "cval", "m.validate"); code != 0 {
		t.Fatalf("link cval: code=%d out=%s", code, out)
	}
	if out, code := run("link", "cdoc", "m.doc"); code != 0 {
		t.Fatalf("link cdoc: code=%d out=%s", code, out)
	}
	out, code := run("todo")
	// Resolve the NODE_ADDED closures one at a time until clean. The fixture
	// keeps one deliberate orphan spec (m.orphan) which cannot be resolved —
	// it exists to exercise the unlinked-spec stats — so exit 1 remains.
	for code == 1 && strings.Contains(out, "Closure") {
		out, code = run("reset", firstClosureHash(t, out))
		if code != 0 {
			t.Fatalf("reset: code=%d out=%s", code, out)
		}
		out, code = run("todo")
	}
	if code == 1 && !strings.Contains(out, "orphan specs") {
		t.Fatalf("todo not clean after baseline: code=%d out=%s", code, out)
	}
	return dir
}

// TestCoverage_Command: Jest-style stat block, legend, per-file rows with
// symbols, closing line.
func TestCoverage_Command(t *testing.T) {
	dir := setupCoverageProject(t)

	out, code := cli.RunWithRender([]string{"coverage"}, dir, output.PlainPresenter{})
	if code != 0 {
		t.Fatalf("coverage: code=%d out=%s", code, out)
	}

	// Stat block with labels.
	for _, want := range []string{
		"Specs", "2 linked to a marker", "1 without a marker",
		"Markers", "Lines", "covered", "linked", "Files",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("stat block missing %q:\n%s", want, out)
		}
	}
	// Legend explaining the terms.
	for _, want := range []string{
		"what this means",
		"lines inside a marker range",
		"visible but unenforced",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("legend missing %q:\n%s", want, out)
		}
	}
	// Per-file rows: marker files with ✓, zero-marker file with ✗.
	if !strings.Contains(out, "✓") || !strings.Contains(out, "code.go") || !strings.Contains(out, "doc.md") {
		t.Fatalf("marker rows missing:\n%s", out)
	}
	if !strings.Contains(out, "✗") || !strings.Contains(out, "plain.go") {
		t.Fatalf("zero-marker row missing:\n%s", out)
	}
	// Collapsed subtree row for the clean lib/ directory.
	if !strings.Contains(out, "lib/*") {
		t.Fatalf("collapsed subtree row missing:\n%s", out)
	}
	// Closing interpretive line.
	if !strings.Contains(out, "Checked 4 files") {
		t.Fatalf("closing line missing:\n%s", out)
	}
	if !strings.Contains(out, "% of lines are under spec protection") {
		t.Fatalf("closing protection line missing:\n%s", out)
	}
}

// TestCoverage_ExitAlwaysZero: coverage never signals drift via exit code,
// even with heavy drift present.
func TestCoverage_ExitAlwaysZero(t *testing.T) {
	dir := setupCoverageProject(t)

	// Mutate a marker region to create drift (inside the cval range).
	mutated := "package main\n" +
		"// D! id=cval range-start\n" +
		"func validate() CHANGED {}\n" +
		"func helper() {}\n" +
		"func helper2() {}\n" +
		"// D! id=cval range-end\n" +
		"// tail\n"
	if err := os.WriteFile(filepath.Join(dir, "code.go"), []byte(mutated), 0644); err != nil {
		t.Fatal(err)
	}

	_, todoCode := cli.RunWithRender([]string{"todo"}, dir, output.PlainPresenter{})
	if todoCode != 1 {
		t.Fatalf("precondition: todo should exit 1 with drift, got %d", todoCode)
	}

	out, code := cli.RunWithRender([]string{"coverage"}, dir, output.PlainPresenter{})
	if code != 0 {
		t.Fatalf("coverage must always exit 0, got %d out=%s", code, out)
	}
}

// TestCoverage_NotInitialized: prescriptive error naming the fix surfaces
// from the state store (coverage adds no duplicate message).
func TestCoverage_NotInitialized(t *testing.T) {
	dir := t.TempDir()
	out, code := cli.RunWithRender([]string{"coverage"}, dir, output.PlainPresenter{})
	if code != 1 {
		t.Fatalf("not-initialized: code=%d out=%s", code, out)
	}
	if !strings.Contains(out, "not found") || !strings.Contains(out, "drift init") {
		t.Fatalf("expected prescriptive not-initialized error, got:\n%s", out)
	}
}

// TestCoverage_UnknownFlagRejected: coverage accepts no flags beyond globals.
func TestCoverage_UnknownFlagRejected(t *testing.T) {
	dir := setupCoverageProject(t)
	out, code := cli.RunWithRender([]string{"coverage", "--frobnicate"}, dir, output.PlainPresenter{})
	if code != 1 {
		t.Fatalf("unknown flag: code=%d out=%s", code, out)
	}
	if !strings.Contains(out, "unknown flag: --frobnicate") {
		t.Fatalf("expected unknown-flag error, got:\n%s", out)
	}
}

// TestCoverage_JSONShape: --json output parses and carries per-file entries
// (including zero-marker files) plus unified totals with spec counts. No
// file-type classification, no percentages, no presentation fields.
func TestCoverage_JSONShape(t *testing.T) {
	dir := setupCoverageProject(t)

	out, code := cli.RunWithRender([]string{"coverage"}, dir, output.JSONPresenter{})
	if code != 0 {
		t.Fatalf("coverage json: code=%d out=%s", code, out)
	}

	var doc struct {
		Files []struct {
			Path          string `json:"path"`
			TotalLines    int    `json:"totalLines"`
			CoveredAny    int    `json:"coveredAny"`
			CoveredLinked int    `json:"coveredLinked"`
			Markers       int    `json:"markers"`
			LinkedMarkers int    `json:"linkedMarkers"`
		} `json:"files"`
		Totals struct {
			FilesWalked       int `json:"filesWalked"`
			FilesWithMarkers  int `json:"filesWithMarkers"`
			TotalLines        int `json:"totalLines"`
			CoveredAny        int `json:"coveredAny"`
			CoveredLinked     int `json:"coveredLinked"`
			MarkersTotal      int `json:"markersTotal"`
			MarkersLinked     int `json:"markersLinked"`
			SpecLines         int `json:"specLines"`
			SpecsTotal        int `json:"specsTotal"`
			SpecsLinked       int `json:"specsLinked"`
		} `json:"totals"`
	}
	if err := json.Unmarshal([]byte(out), &doc); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out)
	}

	if len(doc.Files) != 4 {
		t.Fatalf("expected 4 files in JSON, got %d: %s", len(doc.Files), out)
	}
	byPath := map[string]int{}
	for _, f := range doc.Files {
		byPath[f.Path] = f.TotalLines
	}
	if byPath["code.go"] != 7 {
		t.Fatalf("code.go totalLines = %d, want 7", byPath["code.go"])
	}
	if _, ok := byPath["lib/util.go"]; !ok {
		t.Fatalf("lib/util.go missing from JSON files array (all walked files expected)")
	}
	if _, ok := byPath["main.drift.xml"]; ok {
		t.Fatalf(".drift.xml must not appear in files array")
	}
	for _, f := range doc.Files {
		if f.Path == "code.go" && (f.CoveredAny != 3 || f.LinkedMarkers != 1) {
			t.Fatalf("code.go numbers = %+v", f)
		}
	}

	tot := doc.Totals
	if tot.FilesWalked != 4 || tot.FilesWithMarkers != 2 {
		t.Fatalf("file counts = %+v, want walked 4 withMarkers 2", tot)
	}
	// code.go 7 + plain.go 3 + lib/util.go 1 + doc.md 5 = 16; covered 3+1=4.
	if tot.TotalLines != 16 || tot.CoveredAny != 4 || tot.CoveredLinked != 4 {
		t.Fatalf("line totals = %+v, want total 16 covered 4/4", tot)
	}
	if tot.SpecLines != 5 {
		t.Fatalf("specLines = %d, want 5", tot.SpecLines)
	}
	if tot.SpecsTotal != 3 || tot.SpecsLinked != 2 {
		t.Fatalf("spec counts = %d/%d, want 3/2", tot.SpecsTotal, tot.SpecsLinked)
	}
	if tot.MarkersTotal != 2 || tot.MarkersLinked != 2 {
		t.Fatalf("markers = %d/%d, want 2/2", tot.MarkersTotal, tot.MarkersLinked)
	}
}

// TestHelp_MentionsCoverage: the help text mentions the coverage command.
func TestHelp_MentionsCoverage(t *testing.T) {
	out, code := cli.RunWithRender(nil, t.TempDir(), output.PlainPresenter{})
	if code != 0 {
		t.Fatalf("help: code=%d", code)
	}
	if !strings.Contains(out, "coverage") {
		t.Fatalf("help text missing coverage command:\n%s", out)
	}
}
