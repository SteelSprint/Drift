package output

import (
	"testing"

	"drift/orchestrator"
)

// coverageFiles is a small helper for building report file entries in table
// tests.
func covFile(path string, total, covered, markers int) orchestrator.CoverageFile {
	return orchestrator.CoverageFile{
		Path:       path,
		TotalLines: total,
		CoveredAny: covered,
		Markers:    markers,
	}
}

func rowLabels(rows []coverageRow) []string {
	out := make([]string, len(rows))
	for i, r := range rows {
		out[i] = r.label
	}
	return out
}

func findRow(t *testing.T, rows []coverageRow, label string) coverageRow {
	t.Helper()
	for _, r := range rows {
		if r.label == label {
			return r
		}
	}
	t.Fatalf("row %q not found; rows = %v", label, rowLabels(rows))
	return coverageRow{}
}

// TestCoverageRows_FullyCleanSubtreeCollapses: a directory whose entire
// subtree has no marker-bearing file collapses to one "dir/*" row.
func TestCoverageRows_FullyCleanSubtree(t *testing.T) {
	with, without := buildCoverageRows([]orchestrator.CoverageFile{
		covFile("main.go", 10, 5, 1),
		covFile("docs/a.md", 20, 0, 0),
		covFile("docs/sub/b.md", 30, 0, 0),
	})
	if len(with) != 1 || with[0].label != "main.go" {
		t.Fatalf("withMarkers = %v, want only the marker file", rowLabels(with))
	}
	if len(without) != 1 || without[0].label != "docs/*" {
		t.Fatalf("without = %v, want collapsed docs/*", rowLabels(without))
	}
	if without[0].total != 50 {
		t.Fatalf("collapsed total = %d, want 50 (20+30)", without[0].total)
	}
	if !without[0].collapsed {
		t.Fatal("collapsed row must carry collapsed=true")
	}
}

// TestCoverageRows_MixedDirectoryListsFiles: when a directory contains
// marker files elsewhere, its zero-marker files appear individually.
func TestCoverageRows_MixedDirectoryListsFiles(t *testing.T) {
	with, without := buildCoverageRows([]orchestrator.CoverageFile{
		covFile("cmd/main.go", 100, 40, 2),
		covFile("cmd/plain.go", 5, 0, 0),
	})
	if len(with) != 1 || with[0].label != "cmd/main.go" {
		t.Fatalf("withMarkers = %v", rowLabels(with))
	}
	if len(without) != 1 || without[0].label != "cmd/plain.go" || without[0].collapsed {
		t.Fatalf("without = %v, want individual cmd/plain.go", rowLabels(without))
	}
}

// TestCoverageRows_ShallowestCleanAncestor: a clean pocket nested inside a
// marker-bearing tree collapses at the SHALLOWEST fully-clean ancestor, not
// the file's own directory.
func TestCoverageRows_ShallowestCleanAncestor(t *testing.T) {
	with, without := buildCoverageRows([]orchestrator.CoverageFile{
		covFile("a/root.go", 10, 2, 1),   // a/ has a marker → a/ not clean
		covFile("a/b/inner.go", 3, 0, 0), // a/b/ and a/b/c/ are clean
		covFile("a/b/c/leaf.go", 4, 0, 0),
	})
	if len(with) != 1 || with[0].label != "a/root.go" {
		t.Fatalf("withMarkers = %v", rowLabels(with))
	}
	if len(without) != 1 || without[0].label != "a/b/*" {
		t.Fatalf("without = %v, want collapsed a/b/* (shallowest clean ancestor)", rowLabels(without))
	}
	if without[0].total != 7 {
		t.Fatalf("collapsed total = %d, want 7 (3+4)", without[0].total)
	}
}

// TestCoverageRows_DeepCleanUnderPartial: a deep clean subtree collapses even
// when a sibling branch of the same directory carries markers.
func TestCoverageRows_DeepCleanUnderPartial(t *testing.T) {
	_, without := buildCoverageRows([]orchestrator.CoverageFile{
		covFile("a/one.go", 10, 5, 1),
		covFile("a/clean/c1.go", 3, 0, 0),
		covFile("a/clean/c2.go", 4, 0, 0),
	})
	if len(without) != 1 || without[0].label != "a/clean/*" || without[0].total != 7 {
		t.Fatalf("without = %v, want a/clean/* with 7 lines", rowLabels(without))
	}
}

// TestCoverageRows_SiblingCleanDirsSeparateRows: two clean subtrees produce
// two rows, not one merged glob.
func TestCoverageRows_SiblingCleanDirsSeparateRows(t *testing.T) {
	_, without := buildCoverageRows([]orchestrator.CoverageFile{
		covFile("x/f.go", 10, 5, 1),
		covFile("p/f.go", 2, 0, 0),
		covFile("q/f.go", 3, 0, 0),
	})
	if len(without) != 2 {
		t.Fatalf("without = %v, want 2 collapsed rows", rowLabels(without))
	}
	if without[0].label != "p/*" || without[1].label != "q/*" {
		t.Fatalf("labels = %v, want p/* then q/* (sorted)", rowLabels(without))
	}
}

// TestCoverageRows_RootLevelCleanFiles: zero-marker files at the root (no
// clean ancestor above them) are listed individually.
func TestCoverageRows_RootLevelCleanFiles(t *testing.T) {
	_, without := buildCoverageRows([]orchestrator.CoverageFile{
		covFile("main.go", 10, 5, 1),
		covFile("README.md", 8, 0, 0),
		covFile("LICENSE", 20, 0, 0),
	})
	if len(without) != 2 || without[0].label != "LICENSE" || without[1].label != "README.md" {
		t.Fatalf("without = %v, want LICENSE and README.md individually", rowLabels(without))
	}
	for _, r := range without {
		if r.collapsed {
			t.Fatalf("root-level row %q must not be collapsed", r.label)
		}
	}
}

// TestCoverageRows_SortedStably: rows sort by label within each section.
func TestCoverageRows_SortedStably(t *testing.T) {
	with, without := buildCoverageRows([]orchestrator.CoverageFile{
		covFile("z/z.go", 1, 0, 1),
		covFile("a/a.go", 1, 0, 1),
		covFile("m/m.go", 1, 0, 1),
		covFile("z/clean.go", 2, 0, 0),
		covFile("c/clean.go", 2, 0, 0),
	})
	if rowLabels(with)[0] != "a/a.go" || rowLabels(with)[1] != "m/m.go" || rowLabels(with)[2] != "z/z.go" {
		t.Fatalf("withMarkers order = %v", rowLabels(with))
	}
	if rowLabels(without)[0] != "c/*" || rowLabels(without)[1] != "z/clean.go" {
		t.Fatalf("without order = %v, want c/* (c/ fully clean) then z/clean.go (z/ mixed → individual)", rowLabels(without))
	}
}

// TestCoverageRows_EmptyReport: no files → no rows.
func TestCoverageRows_EmptyReport(t *testing.T) {
	with, without := buildCoverageRows(nil)
	if len(with) != 0 || len(without) != 0 {
		t.Fatalf("expected no rows, got %v / %v", rowLabels(with), rowLabels(without))
	}
}

// TestCoverageRows_AllFilesHaveMarkers: a fully covered tree leaves the
// without section empty.
func TestCoverageRows_AllFilesHaveMarkers(t *testing.T) {
	with, without := buildCoverageRows([]orchestrator.CoverageFile{
		covFile("a/x.go", 10, 5, 1),
		covFile("b/y.go", 10, 5, 1),
	})
	if len(with) != 2 || len(without) != 0 {
		t.Fatalf("with=%v without=%v", rowLabels(with), rowLabels(without))
	}
}

// TestCoverageRows_ZeroLineFiles: empty walked files count 0 lines wherever
// they land (collapsed or individual).
func TestCoverageRows_ZeroLineFiles(t *testing.T) {
	_, without := buildCoverageRows([]orchestrator.CoverageFile{
		covFile("main.go", 10, 5, 1),
		covFile(".gitkeep", 0, 0, 0),
	})
	if len(without) != 1 || without[0].label != ".gitkeep" || without[0].total != 0 {
		t.Fatalf("without = %v, want .gitkeep with 0 lines", rowLabels(without))
	}
}
