package orchestrator_test

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"drift/core"
	"drift/internal/testutil"
	"drift/orchestrator"
	"drift/scanner"
	"drift/statestore"
)

// coverageScanner is a fake scanner that reports a fixed result from a real
// directory, so Coverage can read walked files to count lines.
type coverageScanner struct {
	result scanner.ScanResult
	dir    string
}

func (f *coverageScanner) Scan() (scanner.ScanResult, error) { return f.result, nil }
func (f *coverageScanner) Dir() string                       { return f.dir }

// coverageFixture builds a temp project dir and an orchestrator whose scanner
// reports the given markers and walked files; the walked files are written to
// disk so Coverage can count lines.
func newCoverageFixture(t *testing.T, edges []core.Edge, markers []core.Marker, files map[string]string) *orchestrator.Orchestrator {
	t.Helper()
	dir := t.TempDir()
	for name, content := range files {
		full := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	walked := make([]string, 0, len(files))
	for name := range files {
		walked = append(walked, name)
	}
	sort.Strings(walked)
	store := &fakeStateStore{state: statestore.State{Edges: edges}}
	sc := &coverageScanner{dir: dir, result: scanner.ScanResult{
		Markers:     markers,
		FilesWalked: walked,
	}}
	return orchestrator.NewOrchestrator(store, sc, nil)
}

// markerAt builds a scan marker with explicit start/end lines.
func markerAt(id, file string, start, end int) core.Marker {
	return core.Marker{ID: id, Hash: "h-" + id, Filepath: file, LineNumber: start, EndLineNumber: end}
}

func findCoverageFile(t *testing.T, files []orchestrator.CoverageFile, path string) orchestrator.CoverageFile {
	t.Helper()
	for _, f := range files {
		if f.Path == path {
			return f
		}
	}
	t.Fatalf("coverage file %q not found in report", path)
	return orchestrator.CoverageFile{}
}

// TestOrchestrator_Coverage_SingleMarker: interior lines = end - start - 1.
func TestOrchestrator_Coverage_SingleMarker(t *testing.T) {
	// 8 lines total; marker spans lines 2-7 → 4 interior lines (3,4,5,6)
	content := "line1\n" +
		"// D! id=cval range-start\n" +
		"a\nb\nc\nd\n" +
		"// D! id=cval range-end\n" +
		"line8\n"
	orch := newCoverageFixture(t,
		[]core.Edge{testutil.NewLink("m.a", "cval")},
		[]core.Marker{markerAt("cval", "main.go", 2, 7)},
		map[string]string{"main.go": content},
	)
	report, err := orch.Coverage(nil)
	testutil.AssertNoError(t, err)

	cf := findCoverageFile(t, report.Files, "main.go")
	if cf.TotalLines != 8 {
		t.Fatalf("TotalLines = %d, want 8", cf.TotalLines)
	}
	if cf.CoveredAny != 4 {
		t.Fatalf("CoveredAny = %d, want 4 (interior of lines 2-6)", cf.CoveredAny)
	}
	if cf.CoveredLinked != 4 {
		t.Fatalf("CoveredLinked = %d, want 4 (marker linked)", cf.CoveredLinked)
	}
	if cf.Markers != 1 || cf.LinkedMarkers != 1 {
		t.Fatalf("Markers=%d LinkedMarkers=%d, want 1/1", cf.Markers, cf.LinkedMarkers)
	}
	if report.Totals.CodeTotal != 8 || report.Totals.CodeCoveredAny != 4 || report.Totals.CodeCoveredLinked != 4 {
		t.Fatalf("code totals wrong: %+v", report.Totals)
	}
}

// TestOrchestrator_Coverage_NestedUnion: overlapping/nested markers union,
// no double counting.
func TestOrchestrator_Coverage_NestedUnion(t *testing.T) {
	// outer wraps lines 2-8 (interior 3..7 = 5 lines), inner wraps 4-6
	// (interior 5 = 1 line) — union is 5 lines.
	content := "l1\n" +
		"// D! id=couter range-start\n" +
		"x\n" +
		"// D! id=cinner range-start\n" +
		"mid\n" +
		"// D! id=cinner range-end\n" +
		"y\n" +
		"// D! id=couter range-end\n" +
		"l9\n"
	orch := newCoverageFixture(t,
		[]core.Edge{
			testutil.NewLink("m.a", "couter"),
			testutil.NewLink("m.b", "cinner"),
		},
		[]core.Marker{
			markerAt("couter", "nested.go", 2, 8),
			markerAt("cinner", "nested.go", 4, 6),
		},
		map[string]string{"nested.go": content},
	)
	report, err := orch.Coverage(nil)
	testutil.AssertNoError(t, err)
	cf := findCoverageFile(t, report.Files, "nested.go")
	if cf.TotalLines != 9 {
		t.Fatalf("TotalLines = %d, want 9", cf.TotalLines)
	}
	if cf.CoveredAny != 5 {
		t.Fatalf("CoveredAny = %d, want 5 (union of 3-7 and 5)", cf.CoveredAny)
	}
	if cf.CoveredLinked != 5 {
		t.Fatalf("CoveredLinked = %d, want 5 (both linked)", cf.CoveredLinked)
	}
}

// TestOrchestrator_Coverage_LinkedVsUnlinked: only baseline-linked markers
// count toward CoveredLinked.
func TestOrchestrator_Coverage_LinkedVsUnlinked(t *testing.T) {
	content := "l1\n" +
		"// D! id=clink range-start\nx\ny\n// D! id=clink range-end\n" +
		"l5\n" +
		"// D! id=cfree range-start\nz\n// D! id=cfree range-end\n" +
		"l8\n"
	orch := newCoverageFixture(t,
		[]core.Edge{testutil.NewLink("m.a", "clink")},
		[]core.Marker{
			markerAt("clink", "split.go", 2, 5),
			markerAt("cfree", "split.go", 7, 9),
		},
		map[string]string{"split.go": content},
	)
	report, err := orch.Coverage(nil)
	testutil.AssertNoError(t, err)
	cf := findCoverageFile(t, report.Files, "split.go")
	if cf.TotalLines != 10 {
		t.Fatalf("TotalLines = %d, want 10", cf.TotalLines)
	}
	if cf.CoveredAny != 3 {
		t.Fatalf("CoveredAny = %d, want 3 (2+1)", cf.CoveredAny)
	}
	if cf.CoveredLinked != 2 {
		t.Fatalf("CoveredLinked = %d, want 2 (only clink)", cf.CoveredLinked)
	}
	if cf.Markers != 2 || cf.LinkedMarkers != 1 {
		t.Fatalf("Markers=%d LinkedMarkers=%d, want 2/1", cf.Markers, cf.LinkedMarkers)
	}
	if report.Totals.MarkersLinked != 1 || report.Totals.MarkersTotal != 2 {
		t.Fatalf("totals markers wrong: %+v", report.Totals)
	}
}

// TestOrchestrator_Coverage_MarkdownSplit: .md/.mdx files are reported in the
// markdown buckets, everything else in code buckets.
func TestOrchestrator_Coverage_MarkdownSplit(t *testing.T) {
	mdContent := "# Doc\n" +
		"<!-- D! id=cdoc range-start -->\nprose\n<!-- D! id=cdoc range-end -->\n" +
		"tail\n"
	orch := newCoverageFixture(t,
		[]core.Edge{
			testutil.NewLink("m.a", "cdoc"),
			testutil.NewLink("m.b", "cguide"),
		},
		[]core.Marker{
			markerAt("cdoc", "doc.md", 2, 4),
			markerAt("cguide", "guide.mdx", 2, 4),
		},
		map[string]string{
			"doc.md":    mdContent,
			"guide.mdx": strings.ReplaceAll(mdContent, "cdoc", "cguide"),
			"code.go":   "p\nq\nr\n",
		},
	)
	report, err := orch.Coverage(nil)
	testutil.AssertNoError(t, err)
	if report.Totals.MdTotal != 10 {
		t.Fatalf("MdTotal = %d, want 10 (5+5)", report.Totals.MdTotal)
	}
	if report.Totals.MdCoveredAny != 2 {
		t.Fatalf("MdCoveredAny = %d, want 2", report.Totals.MdCoveredAny)
	}
	if report.Totals.CodeTotal != 3 || report.Totals.CodeCoveredAny != 0 {
		t.Fatalf("code totals wrong: %+v", report.Totals)
	}
}

// TestOrchestrator_Coverage_SpecLines: unique .drift.xml files in the import
// chain contribute their raw line count to SpecLines, and do NOT appear in
// the per-file list.
func TestOrchestrator_Coverage_SpecLines(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "main.drift.xml"), []byte("<main>\n<spec id=\"a\">x</spec>\n</main>\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("a\nb\n"), 0644); err != nil {
		t.Fatal(err)
	}
	store := &fakeStateStore{}
	sc := &coverageScanner{dir: dir, result: scanner.ScanResult{
		Specs:       []core.Spec{testutil.NewSpecWithLocation("m.a", "h", "main.drift.xml", 2)},
		FilesWalked: []string{"main.go"},
	}}
	orch := orchestrator.NewOrchestrator(store, sc, nil)
	report, err := orch.Coverage(nil)
	testutil.AssertNoError(t, err)
	if report.Totals.SpecLines != 3 {
		t.Fatalf("SpecLines = %d, want 3", report.Totals.SpecLines)
	}
	for _, cf := range report.Files {
		if cf.Path == "main.drift.xml" {
			t.Fatal(".drift.xml files must not appear in Files list")
		}
	}
}

// TestOrchestrator_Coverage_EmptyAndZeroInterior: adjacent markers (no
// interior lines) contribute 0; walked files with zero lines are reported.
func TestOrchestrator_Coverage_EmptyAndZeroInterior(t *testing.T) {
	content := "// D! id=cadj range-start\n// D! id=cadj range-end\n"
	orch := newCoverageFixture(t,
		[]core.Edge{testutil.NewLink("m.a", "cadj")},
		[]core.Marker{markerAt("cadj", "adj.go", 1, 2)},
		map[string]string{
			"adj.go":   content,
			"empty.go": "",
		},
	)
	report, err := orch.Coverage(nil)
	testutil.AssertNoError(t, err)
	cf := findCoverageFile(t, report.Files, "adj.go")
	if cf.CoveredAny != 0 || cf.TotalLines != 2 {
		t.Fatalf("adj.go: covered=%d total=%d, want 0/2", cf.CoveredAny, cf.TotalLines)
	}
	ef := findCoverageFile(t, report.Files, "empty.go")
	if ef.TotalLines != 0 {
		t.Fatalf("empty.go TotalLines = %d, want 0", ef.TotalLines)
	}
	if report.Totals.FilesWalked != 2 {
		t.Fatalf("FilesWalked = %d, want 2", report.Totals.FilesWalked)
	}
}

// TestOrchestrator_Coverage_SortedDeterministic: Files sorted by path.
func TestOrchestrator_Coverage_SortedDeterministic(t *testing.T) {
	orch := newCoverageFixture(t, nil, nil,
		map[string]string{
			"zeta.go":  "a\n",
			"sub/b.go": "a\n",
			"alpha.go": "a\n",
		})
	report, err := orch.Coverage(nil)
	testutil.AssertNoError(t, err)
	if len(report.Files) != 3 {
		t.Fatalf("expected 3 files, got %d", len(report.Files))
	}
	want := []string{"alpha.go", "sub/b.go", "zeta.go"}
	for i, w := range want {
		if report.Files[i].Path != w {
			t.Fatalf("Files[%d]=%q, want %q", i, report.Files[i].Path, w)
		}
	}
}
