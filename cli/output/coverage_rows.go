package output

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"drift/orchestrator"
)

// D! id=ocovrows range-start

// coverageRow is one plain/color output row: either a marker-bearing file
// (individual, with coverage numbers) or a marker-free subtree collapsed to
// a single label ("dir/*").
type coverageRow struct {
	label      string
	total      int
	coveredAny int
	linked     int
	markers    int
	collapsed  bool
}

// buildCoverageRows partitions a coverage report's files into rows for the
// text presenters. Marker-bearing files are listed individually. A directory
// whose entire subtree contains no marker-bearing file collapses into one
// row labeled "dir/*" (total lines only). Zero-marker files inside a
// directory that does contain marker files elsewhere are listed individually,
// so nothing hides inside a partially covered directory.
func buildCoverageRows(files []orchestrator.CoverageFile) (withMarkers, without []coverageRow) {
	// Directories (as prefix paths, root = "") that contain at least one
	// marker-bearing file anywhere in their subtree.
	clean := map[string]bool{} // dir → subtree is marker-free
	hasMarker := map[string]bool{}
	for _, f := range files {
		if f.Markers > 0 {
			hasMarker[f.Path] = true
		}
	}
	// subtreeHasMarkers(dir) via memoized walk up the path segments.
	var subtreeHasMarkers func(dir string) bool
	subtreeHasMarkers = func(dir string) bool {
		v, ok := clean[dir]
		if ok {
			return v
		}
		found := false
		for _, f := range files {
			if hasMarker[f.Path] && dirContains(dir, f.Path) {
				found = true
				break
			}
		}
		clean[dir] = found
		return found
	}

	// Collapse point for a marker-free file: the SHALLOWEST ancestor
	// directory (including its own) whose subtree is marker-free. The walk
	// continues upward while ancestors are also clean and keeps the shallowest
	// one; "" means no clean ancestor exists (root is mixed) → individual row.
	collapseDir := func(rel string) string {
		shallowest := ""
		dir := filepath.ToSlash(filepath.Dir(rel))
		for dir != "." && dir != "/" && dir != "" {
			if !subtreeHasMarkers(dir) {
				shallowest = dir
			} else {
				break
			}
			dir = filepath.ToSlash(filepath.Dir(dir))
		}
		return shallowest
	}

	collapsedTotals := map[string]int{}
	dirsCollapsed := map[string]bool{}
	for _, f := range files {
		if f.Markers > 0 {
			withMarkers = append(withMarkers, coverageRow{
				label:      f.Path,
				total:      f.TotalLines,
				coveredAny: f.CoveredAny,
				linked:     f.CoveredLinked,
				markers:    f.Markers,
			})
			continue
		}
		cdir := collapseDir(f.Path)
		if cdir == "" {
			without = append(without, coverageRow{
				label: f.Path,
				total: f.TotalLines,
			})
			continue
		}
		collapsedTotals[cdir] += f.TotalLines
		dirsCollapsed[cdir] = true
	}
	for dir, total := range collapsedTotals {
		if !dirsCollapsed[dir] {
			continue
		}
		without = append(without, coverageRow{
			label:     dir + "/*",
			total:     total,
			collapsed: true,
		})
	}

	sortCoverageRows(withMarkers)
	sortCoverageRows(without)
	return withMarkers, without
}

// dirContains reports whether rel is inside dir (dir == "" is the root).
func dirContains(dir, rel string) bool {
	dir = strings.TrimPrefix(filepath.ToSlash(dir), "./")
	if dir == "" || dir == "." {
		return true
	}
	rel = filepath.ToSlash(rel)
	return strings.HasPrefix(rel, dir+"/")
}

func sortCoverageRows(rows []coverageRow) {
	sort.Slice(rows, func(i, j int) bool { return rows[i].label < rows[j].label })
}

// D! id=ocovrows range-end

// formatInt renders an integer with thousands separators (12,345) for the
// human-readable stat block. JSON keeps raw ints.
func formatInt(n int) string {
	s := fmt.Sprintf("%d", n)
	if len(s) <= 3 {
		return s
	}
	var parts []string
	for len(s) > 3 {
		parts = append([]string{s[len(s)-3:]}, parts...)
		s = s[:len(s)-3]
	}
	parts = append([]string{s}, parts...)
	return strings.Join(parts, ",")
}

// plural returns "s" for n != 1.
func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}
