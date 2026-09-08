package orchestrator

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"drift/core"
	"drift/internal/fileio"
)

// D! id=ocov range-start

// CoverageFile reports coverage for one walked file. CoveredAny counts the
// union of all marker interiors; CoveredLinked counts the union of interiors
// of markers linked to a spec in the baseline. There is no file-type
// classification — drift is language-agnostic (see principles.language_agnostic).
// Percentages are a presenter concern; this layer carries raw counts only.
type CoverageFile struct {
	Path          string
	TotalLines    int
	CoveredAny    int
	CoveredLinked int
	Markers       int
	LinkedMarkers int
}

// CoverageTotals aggregates the per-file numbers plus the spec layer.
// SpecLines is the raw line count of the unique *.drift.xml files in the
// import chain — the size of the spec layer, reported separately from the
// walked files. SpecsTotal/SpecsLinked count non-deleted scan specs and
// those named by a baseline link-style edge (spec is enforced); their
// difference is the count of visible-but-unenforced specs.
type CoverageTotals struct {
	FilesWalked       int
	FilesWithMarkers  int
	SpecLines         int
	TotalLines        int
	CoveredAny        int
	CoveredLinked     int
	MarkersTotal      int
	MarkersLinked     int
	SpecsTotal        int
	SpecsLinked       int
}

// CoverageReport is the full coverage picture: one entry per walked file
// (sorted by path) plus totals.
type CoverageReport struct {
	Files  []CoverageFile
	Totals CoverageTotals
}

// Coverage reports how many lines marker ranges cover. It loads the baseline
// (for link edges), scans (reusing the scanner walk, so drift.ignore and
// text-file detection apply), and computes per-file unions of marker
// interiors. Read-only — it never mutates state.
//
// Covered lines are the interior lines between range-start and range-end
// (end − start − 1), matching the scanner's half-open hashing semantics.
// Nested and overlapping markers are unioned per file, never double-counted.
// A marker counts as linked when a baseline link-style edge (marker → spec)
// names it; unlinked markers count toward CoveredAny only. A spec counts as
// linked when a baseline link-style edge names it on the To side.
func (o *Orchestrator) Coverage(sess *fileio.Session) (CoverageReport, error) {
	state, err := o.stateStore.Load(sess)
	if err != nil {
		return CoverageReport{}, err
	}

	scanResult, err := o.scanner.Scan()
	if err != nil {
		return CoverageReport{}, err
	}

	// Baseline link-style edges (From is a marker — no dot) define which
	// markers and specs are enforced. Ref edges are spec→spec and never
	// contribute to linkage.
	linkedMarkers := map[string]bool{}
	linkedSpecs := map[string]bool{}
	for _, e := range state.Edges {
		if !isSpecIDOrch(e.From) && isSpecIDOrch(e.To) {
			linkedMarkers[e.From] = true
			linkedSpecs[e.To] = true
		}
	}

	// Group scan markers by file.
	markersByFile := map[string][]core.Marker{}
	for _, m := range scanResult.Markers {
		markersByFile[m.Filepath] = append(markersByFile[m.Filepath], m)
	}

	report := CoverageReport{
		Totals: CoverageTotals{
			FilesWalked: len(scanResult.FilesWalked),
			SpecsTotal:  len(scanResult.Specs),
		},
	}
	seen := map[string]bool{}
	for _, rel := range scanResult.FilesWalked {
		if seen[rel] {
			continue
		}
		seen[rel] = true
		total, err := countLines(filepath.Join(o.scanner.Dir(), rel))
		if err != nil {
			return CoverageReport{}, err
		}
		fileMarkers := markersByFile[rel]
		cf := CoverageFile{Path: rel, TotalLines: total}
		var anyIvs, linkedIvs []lineInterval
		for _, m := range fileMarkers {
			iv := lineInterval{start: m.LineNumber, end: m.EndLineNumber}
			anyIvs = append(anyIvs, iv)
			cf.Markers++
			report.Totals.MarkersTotal++
			if linkedMarkers[m.ID] {
				linkedIvs = append(linkedIvs, iv)
				cf.LinkedMarkers++
				report.Totals.MarkersLinked++
			}
		}
		if cf.Markers > 0 {
			report.Totals.FilesWithMarkers++
		}
		cf.CoveredAny = coveredLines(anyIvs)
		cf.CoveredLinked = coveredLines(linkedIvs)

		report.Files = append(report.Files, cf)
		report.Totals.TotalLines += cf.TotalLines
		report.Totals.CoveredAny += cf.CoveredAny
		report.Totals.CoveredLinked += cf.CoveredLinked
	}

	for _, s := range scanResult.Specs {
		if linkedSpecs[s.ID] {
			report.Totals.SpecsLinked++
		}
	}

	report.Totals.SpecLines, err = specLineCount(o.scanner.Dir(), scanResult.Specs)
	if err != nil {
		return CoverageReport{}, err
	}
	return report, nil
}

// lineInterval is one marker span; covered lines are its interior
// (start+1 .. end-1).
type lineInterval struct{ start, end int }

// coveredLines sums the union of interval interiors. Intervals are merged
// before summing so nested and overlapping markers never double-count.
// Two interiors intersect or touch exactly when the second interval's start
// line is strictly before the current end line.
func coveredLines(ivs []lineInterval) int {
	if len(ivs) == 0 {
		return 0
	}
	sort.Slice(ivs, func(i, j int) bool {
		if ivs[i].start != ivs[j].start {
			return ivs[i].start < ivs[j].start
		}
		return ivs[i].end < ivs[j].end
	})
	total := 0
	cur := ivs[0]
	for _, iv := range ivs[1:] {
		if iv.start < cur.end {
			if iv.end > cur.end {
				cur.end = iv.end
			}
		} else {
			total += maxInt(0, cur.end-cur.start-1)
			cur = iv
		}
	}
	total += maxInt(0, cur.end-cur.start-1)
	return total
}

// countLines returns the number of lines in a file using the same convention
// as the scanner's line splitting: lines are \n-separated; a trailing partial
// line (no final newline) counts; an empty file has 0 lines.
func countLines(path string) (int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	if len(data) == 0 {
		return 0, nil
	}
	n := strings.Count(string(data), "\n")
	if data[len(data)-1] != '\n' {
		n++
	}
	return n, nil
}

// specLineCount sums the raw line counts of the unique *.drift.xml files in
// the import chain (the spec layer's size).
func specLineCount(dir string, specs []core.Spec) (int, error) {
	seen := map[string]bool{}
	total := 0
	for _, s := range specs {
		if seen[s.Filepath] {
			continue
		}
		seen[s.Filepath] = true
		n, err := countLines(filepath.Join(dir, s.Filepath))
		if err != nil {
			return 0, err
		}
		total += n
	}
	return total, nil
}

// isSpecIDOrch reports whether an ID carries the module-qualified spec shape
// (a dot separating module from local part). The predicate is duplicated here
// (as in cli/commands) to keep the orchestrator free of cross-package
// helpers; marker shortcodes never contain a dot.
func isSpecIDOrch(id string) bool {
	first := strings.Index(id, ".")
	if first < 0 {
		return false
	}
	return strings.Index(id[first+1:], ".") < 0
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// D! id=ocov range-end
