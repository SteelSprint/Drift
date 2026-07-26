package diff_test

import (
	"strings"
	"testing"

	"drift/internal/diff"
)

func TestUnifiedDiffIdentity(t *testing.T) {
	got := diff.UnifiedDiff("hello\nworld\n", "hello\nworld\n")
	if got != "" {
		t.Fatalf("expected empty diff for identical input, got:\n%s", got)
	}
}

func TestUnifiedDiffEmpty(t *testing.T) {
	got := diff.UnifiedDiff("", "")
	if got != "" {
		t.Fatalf("expected empty diff for both empty, got:\n%s", got)
	}
}

func TestUnifiedDiffInsert(t *testing.T) {
	old := "a\nb\n"
	new := "a\nx\nb\n"
	got := diff.UnifiedDiff(old, new)
	if !strings.Contains(got, "\n+x") {
		t.Fatalf("expected +x line in diff:\n%s", got)
	}
	if hasRemovedLine(got) {
		t.Fatalf("expected no removed lines in insert-only diff:\n%s", got)
	}
}

func TestUnifiedDiffDelete(t *testing.T) {
	old := "a\nx\nb\n"
	new := "a\nb\n"
	got := diff.UnifiedDiff(old, new)
	if !strings.Contains(got, "\n-x") {
		t.Fatalf("expected -x line in diff:\n%s", got)
	}
	if hasAddedLine(got) {
		t.Fatalf("expected no added lines in delete-only diff:\n%s", got)
	}
}

func TestUnifiedDiffModify(t *testing.T) {
	old := "a\nx\nb\n"
	new := "a\ny\nb\n"
	got := diff.UnifiedDiff(old, new)
	if !strings.Contains(got, "\n-x") || !strings.Contains(got, "\n+y") {
		t.Fatalf("expected -x and +y in diff:\n%s", got)
	}
}

func TestUnifiedDiffReplaceBlock(t *testing.T) {
	old := "a\nx\ny\nb\n"
	new := "a\np\nq\nb\n"
	got := diff.UnifiedDiff(old, new)
	if !strings.Contains(got, "\n-x") || !strings.Contains(got, "\n-y") {
		t.Fatalf("expected -x -y in diff:\n%s", got)
	}
	if !strings.Contains(got, "\n+p") || !strings.Contains(got, "\n+q") {
		t.Fatalf("expected +p +q in diff:\n%s", got)
	}
}

func TestUnifiedDiffMultiHunk(t *testing.T) {
	old := "l1\nl2\nl3\nl4\nl5\nl6\nl7\nl8\nl9\nl10\nl11\nl12\n"
	new := "l1\nl2\nl3\nl4\nMOD\nl6\nl7\nl8\nl9\nl10\nl11\nl12\n"
	got := diff.UnifiedDiff(old, new)
	if !strings.Contains(got, "MOD") {
		t.Fatalf("expected MOD in diff:\n%s", got)
	}
	// single change in middle, small file → should still produce one hunk
	hunks := strings.Count(got, "@@ -")
	if hunks != 1 {
		t.Fatalf("expected one hunk, got %d:\n%s", hunks, got)
	}
}

func TestUnifiedDiffMultiHunkDisjoint(t *testing.T) {
	old := "l1\nl2\nl3\nl4\nl5\nl6\nl7\nl8\nl9\nl10\nl11\nl12\nl13\nl14\nl15\n"
	new := "CHANGED1\nl2\nl3\nl4\nl5\nl6\nl7\nl8\nl9\nl10\nl11\nl12\nl13\nl14\nCHANGED2\n"
	got := diff.UnifiedDiff(old, new)
	hunks := strings.Count(got, "@@ -")
	if hunks < 2 {
		t.Fatalf("expected at least 2 hunks for disjoint changes, got %d:\n%s", hunks, got)
	}
}

func TestUnifiedDiffSingleLine(t *testing.T) {
	got := diff.UnifiedDiff("a\n", "b\n")
	if !strings.Contains(got, "\n-a") || !strings.Contains(got, "\n+b") {
		t.Fatalf("expected -a +b in single-line diff:\n%s", got)
	}
}

func TestUnifiedDiffDeletionDrift(t *testing.T) {
	// Marker deleted → new content is empty.
	got := diff.UnifiedDiff("line1\nline2\nline3\n", "")
	if !strings.Contains(got, "\n-line1") || !strings.Contains(got, "\n-line2") {
		t.Fatalf("expected all-removed lines in deletion drift:\n%s", got)
	}
	if hasAddedLine(got) {
		t.Fatalf("expected no added lines in deletion drift:\n%s", got)
	}
}

func TestUnifiedDiffNewOnly(t *testing.T) {
	// Old empty, new has content (e.g. marker just added).
	got := diff.UnifiedDiff("", "line1\nline2\n")
	if !strings.Contains(got, "\n+line1") || !strings.Contains(got, "\n+line2") {
		t.Fatalf("expected all-added lines:\n%s", got)
	}
	if hasRemovedLine(got) {
		t.Fatalf("expected no removed lines:\n%s", got)
	}
}

func TestUnifiedDiffHunkHeader(t *testing.T) {
	got := diff.UnifiedDiff("a\nb\nc\nd\n", "a\nb\nNEW\nd\n")
	if !strings.HasPrefix(got, "@@ ") {
		t.Fatalf("expected diff to start with hunk header:\n%s", got)
	}
	if !strings.Contains(got, "@@ -1,4 +1,4 @@") && !strings.Contains(got, "@@ -2") {
		t.Fatalf("expected hunk header with line counts:\n%s", got)
	}
}

func hasRemovedLine(s string) bool {
	for _, line := range strings.Split(s, "\n") {
		if strings.HasPrefix(line, "-") {
			return true
		}
	}
	return false
}

func hasAddedLine(s string) bool {
	for _, line := range strings.Split(s, "\n") {
		if strings.HasPrefix(line, "+") {
			return true
		}
	}
	return false
}

// Hunk-splitting tests (R4): "Two change regions separated by more than
// 2*3+1 context lines MUST produce separate hunks." Gap = number of context
// lines between change regions. Gap <= 7 → one hunk. Gap > 7 → two hunks.
// Hunks MUST NOT overlap (no line appears in more than one hunk).

func countHunks(t *testing.T, s string) int {
	t.Helper()
	return strings.Count(s, "@@ -")
}

// hunkLines extracts the lines in each hunk (the body lines after the
// header). Returns a slice of slices for overlap checking.
func hunkLines(t *testing.T, s string) [][]string {
	t.Helper()
	var hunks [][]string
	var current []string
	inHunk := false
	for _, line := range strings.Split(s, "\n") {
		if strings.HasPrefix(line, "@@ ") {
			if inHunk {
				hunks = append(hunks, current)
			}
			current = nil
			inHunk = true
			continue
		}
		if inHunk {
			if line == "" {
				continue
			}
			// Strip the leading prefix character (' ', '-', '+')
			if len(line) > 0 {
				current = append(current, line[1:])
			}
		}
	}
	if inHunk {
		hunks = append(hunks, current)
	}
	return hunks
}

func TestHunkSplitGap4(t *testing.T) {
	// Two changes separated by 4 context lines. Gap <= 7 → one hunk.
	old := "A\nC1\nC2\nC3\nC4\nB\n"
	new := "A2\nC1\nC2\nC3\nC4\nB2\n"
	got := diff.UnifiedDiff(old, new)
	if c := countHunks(t, got); c != 1 {
		t.Errorf("gap=4: expected 1 hunk, got %d:\n%s", c, got)
	}
}

func TestHunkSplitGap7(t *testing.T) {
	// Two changes separated by 7 context lines. Gap <= 7 → one hunk.
	old := "A\nC1\nC2\nC3\nC4\nC5\nC6\nC7\nB\n"
	new := "A2\nC1\nC2\nC3\nC4\nC5\nC6\nC7\nB2\n"
	got := diff.UnifiedDiff(old, new)
	if c := countHunks(t, got); c != 1 {
		t.Errorf("gap=7: expected 1 hunk, got %d:\n%s", c, got)
	}
}

func TestHunkSplitGap8(t *testing.T) {
	// Two changes separated by 8 context lines. Gap > 7 → two hunks.
	old := "A\nC1\nC2\nC3\nC4\nC5\nC6\nC7\nC8\nB\n"
	new := "A2\nC1\nC2\nC3\nC4\nC5\nC6\nC7\nC8\nB2\n"
	got := diff.UnifiedDiff(old, new)
	if c := countHunks(t, got); c != 2 {
		t.Errorf("gap=8: expected 2 hunks, got %d:\n%s", c, got)
	}
}

func TestHunkSplitNoOverlap(t *testing.T) {
	// Hunks MUST NOT overlap — no content line appears in more than one
	// hunk. This catches the overlapping-hunk bug where trailing context
	// of hunk 1 becomes leading context of hunk 2.
	old := "A\nC1\nC2\nC3\nC4\nB\n"
	new := "A2\nC1\nC2\nC3\nC4\nB2\n"
	got := diff.UnifiedDiff(old, new)
	hunks := hunkLines(t, got)
	if len(hunks) > 1 {
		seen := make(map[string]bool)
		for _, hunk := range hunks {
			for _, line := range hunk {
				if seen[line] {
					t.Errorf("line %q appears in multiple hunks (overlap):\n%s", line, got)
				}
				seen[line] = true
			}
		}
	}
}

func TestHunkSplitMixedGaps(t *testing.T) {
	// Three changes: gap1=2 (merge A+B), gap2=10 (split B+C).
	// Expected: 2 hunks (A+B merged, C separate).
	old := "A\nC1\nC2\nB\nD1\nD2\nD3\nD4\nD5\nD6\nD7\nD8\nD9\nD10\nE\n"
	new := "A2\nC1\nC2\nB2\nD1\nD2\nD3\nD4\nD5\nD6\nD7\nD8\nD9\nD10\nE2\n"
	got := diff.UnifiedDiff(old, new)
	if c := countHunks(t, got); c != 2 {
		t.Errorf("mixed gaps (2,10): expected 2 hunks, got %d:\n%s", c, got)
	}
}

func TestHunkSplitConsecutiveChanges(t *testing.T) {
	// Two changes with no context between them (gap=0). Always one hunk.
	old := "A\nB\nC\n"
	new := "A2\nB2\nC\n"
	got := diff.UnifiedDiff(old, new)
	if c := countHunks(t, got); c != 1 {
		t.Errorf("gap=0: expected 1 hunk, got %d:\n%s", c, got)
	}
}

func TestHunkSplitExactly7(t *testing.T) {
	// Boundary: gap = exactly 7 → merge. Two changes at positions 1 and 9
	// (1-indexed), separated by lines 2-8 (7 context lines).
	old := "OLD\nK1\nK2\nK3\nK4\nK5\nK6\nK7\nOLD2\n"
	new := "NEW\nK1\nK2\nK3\nK4\nK5\nK6\nK7\nNEW2\n"
	got := diff.UnifiedDiff(old, new)
	if c := countHunks(t, got); c != 1 {
		t.Errorf("gap=exactly 7: expected 1 hunk, got %d:\n%s", c, got)
	}
}

func TestHunkSplitExactly8(t *testing.T) {
	// Boundary: gap = exactly 8 → split. Two changes at positions 1 and 10
	// (1-indexed), separated by lines 2-9 (8 context lines).
	old := "OLD\nK1\nK2\nK3\nK4\nK5\nK6\nK7\nK8\nOLD2\n"
	new := "NEW\nK1\nK2\nK3\nK4\nK5\nK6\nK7\nK8\nNEW2\n"
	got := diff.UnifiedDiff(old, new)
	if c := countHunks(t, got); c != 2 {
		t.Errorf("gap=exactly 8: expected 2 hunks, got %d:\n%s", c, got)
	}
}
