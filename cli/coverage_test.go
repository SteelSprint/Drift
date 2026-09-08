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
//   - main.drift.xml with one spec (m.validate)
//   - code.go with a linked marker (3 interior lines)
//   - doc.md with a linked marker (1 interior line)
//   - plain.go with no markers
// and baselines everything so todo is clean.
func setupCoverageProject(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	testutil.WriteSpecFile(t, dir, "main.drift.xml",
		`<module name="m">
<spec id="validate">Validate input.</spec>
<spec id="doc">Document the validation flow.</spec>
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
	if code != 1 {
		t.Fatalf("todo after link: code=%d out=%s", code, out)
	}
	// Resolve the NODE_ADDED closures one at a time until clean.
	for code == 1 && strings.Contains(out, "Closure") {
		out, code = run("reset", firstClosureHash(t, out))
		if code != 0 {
			t.Fatalf("reset: code=%d out=%s", code, out)
		}
		out, code = run("todo")
	}
	if code != 0 {
		t.Fatalf("todo not clean after baseline: code=%d out=%s", code, out)
	}
	return dir
}

// TestCoverage_Command: read-only report, exit 0, per-file rows and totals.
func TestCoverage_Command(t *testing.T) {
	dir := setupCoverageProject(t)

	out, code := cli.RunWithRender([]string{"coverage"}, dir, output.PlainPresenter{})
	if code != 0 {
		t.Fatalf("coverage: code=%d out=%s", code, out)
	}

	// Per-file rows for marker-bearing files.
	if !strings.Contains(out, "code.go") || !strings.Contains(out, "doc.md") {
		t.Fatalf("coverage output missing marker files:\n%s", out)
	}
	// Zero-marker files appear (all walked files are in the report).
	if !strings.Contains(out, "plain.go") {
		t.Fatalf("coverage output missing zero-marker file plain.go:\n%s", out)
	}
	// Markdown and code totals both present.
	if !strings.Contains(out, "markdown") || !strings.Contains(out, "code") {
		t.Fatalf("coverage output missing markdown/code totals:\n%s", out)
	}
	// Spec lines reported (the .drift.xml layer).
	if !strings.Contains(out, "spec layer") {
		t.Fatalf("coverage output missing spec layer total:\n%s", out)
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
// (including zero-marker files) plus totals.
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
			IsMarkdown    bool   `json:"isMarkdown"`
		} `json:"files"`
		Totals struct {
			FilesWalked       int `json:"filesWalked"`
			SpecLines         int `json:"specLines"`
			CodeTotal         int `json:"codeTotal"`
			CodeCoveredAny    int `json:"codeCoveredAny"`
			CodeCoveredLinked int `json:"codeCoveredLinked"`
			MdTotal           int `json:"mdTotal"`
			MdCoveredAny      int `json:"mdCoveredAny"`
			MdCoveredLinked   int `json:"mdCoveredLinked"`
			MarkersTotal      int `json:"markersTotal"`
			MarkersLinked     int `json:"markersLinked"`
		} `json:"totals"`
	}
	if err := json.Unmarshal([]byte(out), &doc); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out)
	}

	if len(doc.Files) != 3 {
		t.Fatalf("expected 3 files in JSON, got %d: %s", len(doc.Files), out)
	}
	byPath := map[string]int{}
	for _, f := range doc.Files {
		byPath[f.Path] = f.TotalLines
	}
	if byPath["code.go"] != 7 {
		t.Fatalf("code.go totalLines = %d, want 7", byPath["code.go"])
	}
	if _, ok := byPath["plain.go"]; !ok {
		t.Fatalf("plain.go missing from JSON files array")
	}
	if _, ok := byPath["main.drift.xml"]; ok {
		t.Fatalf(".drift.xml must not appear in files array")
	}

	tot := doc.Totals
	if tot.FilesWalked != 3 {
		t.Fatalf("filesWalked = %d, want 3", tot.FilesWalked)
	}
	// code.go: 7 lines, marker interior = lines 3-5 = 3 covered
	if tot.CodeTotal != 10 || tot.CodeCoveredAny != 3 || tot.CodeCoveredLinked != 3 {
		t.Fatalf("code totals = %+v, want total 10 covered 3/3", tot)
	}
	// doc.md: 5 lines, marker interior = 1
	if tot.MdTotal != 5 || tot.MdCoveredAny != 1 || tot.MdCoveredLinked != 1 {
		t.Fatalf("md totals = %+v, want total 5 covered 1/1", tot)
	}
	if tot.SpecLines != 4 {
		t.Fatalf("specLines = %d, want 4", tot.SpecLines)
	}
	if tot.MarkersTotal != 2 || tot.MarkersLinked != 2 {
		t.Fatalf("markers = %d/%d, want 2/2", tot.MarkersTotal, tot.MarkersLinked)
	}
}

// TestHelpListsCoverage: the help text mentions the coverage command.
func TestHelp_MentionsCoverage(t *testing.T) {
	out, code := cli.RunWithRender(nil, t.TempDir(), output.PlainPresenter{})
	if code != 0 {
		t.Fatalf("help: code=%d", code)
	}
	if !strings.Contains(out, "coverage") {
		t.Fatalf("help text missing coverage command:\n%s", out)
	}
}
