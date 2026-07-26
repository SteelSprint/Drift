package diff

import "strings"

// D! id=dunif range-start

// UnifiedDiff returns a unified-diff style string comparing old and new.
// Uses LCS-based algorithm with 3 lines of context. Returns empty string
// when inputs are identical (or both empty).
func UnifiedDiff(old, new string) string {
	oldLines := splitLines(old)
	newLines := splitLines(new)

	if linesEqual(oldLines, newLines) {
		return ""
	}

	hunks := buildHunks(oldLines, newLines)
	if len(hunks) == 0 {
		return ""
	}

	var sb strings.Builder
	for _, h := range hunks {
		sb.WriteString(h.header)
		sb.WriteString("\n")
		sb.WriteString(h.body)
	}
	return strings.TrimRight(sb.String(), "\n")
}

type hunk struct {
	header string
	body   string
}

type op struct {
	kind byte // ' ', '-', '+'
	line string
}

func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	s = strings.ReplaceAll(s, "\r\n", "\n")
	lines := strings.Split(s, "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}

func linesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// lcsTable computes the LCS DP table. Returns table of size (m+1) x (n+1).
func lcsTable(a, b []string) [][]int {
	m, n := len(a), len(b)
	dp := make([][]int, m+1)
	for i := range dp {
		dp[i] = make([]int, n+1)
	}
	for i := 1; i <= m; i++ {
		for j := 1; j <= n; j++ {
			if a[i-1] == b[j-1] {
				dp[i][j] = dp[i-1][j-1] + 1
			} else if dp[i-1][j] >= dp[i][j-1] {
				dp[i][j] = dp[i-1][j]
			} else {
				dp[i][j] = dp[i][j-1]
			}
		}
	}
	return dp
}

// diffOps walks the LCS table back to produce a sequence of ops.
func diffOps(a, b []string) []op {
	dp := lcsTable(a, b)
	var ops []op
	i, j := len(a), len(b)
	for i > 0 || j > 0 {
		switch {
		case i > 0 && j > 0 && a[i-1] == b[j-1]:
			ops = append([]op{{' ', a[i-1]}}, ops...)
			i--
			j--
		case j > 0 && (i == 0 || dp[i][j-1] >= dp[i-1][j]):
			ops = append([]op{{'+', b[j-1]}}, ops...)
			j--
		case i > 0 && (j == 0 || dp[i][j-1] < dp[i-1][j]):
			ops = append([]op{{'-', a[i-1]}}, ops...)
			i--
		}
	}
	return ops
}

const contextLines = 3

// gapThreshold is the maximum number of context lines between two change
// regions that still merge into a single hunk. Spec R4: "Two change regions
// separated by more than 2*3+1 context lines MUST produce separate hunks."
// So gap <= gapThreshold merges; gap > gapThreshold splits.
const gapThreshold = 2*contextLines + 1

func buildHunks(oldLines, newLines []string) []hunk {
	ops := diffOps(oldLines, newLines)

	// Assign old/new line numbers to each op.
	oldNo, newNo := 0, 0
	type numberedOp struct {
		op
		oldNo, newNo int
	}
	nops := make([]numberedOp, len(ops))
	for i, o := range ops {
		nops[i] = numberedOp{o, oldNo, newNo}
		switch o.kind {
		case ' ':
			oldNo++
			newNo++
		case '-':
			oldNo++
		case '+':
			newNo++
		}
	}

	// Find change indices (non-context ops).
	nextChange := func(from int) int {
		for k := from; k < len(nops); k++ {
			if nops[k].kind != ' ' {
				return k
			}
		}
		return -1
	}

	var hunks []hunk
	changeIdx := nextChange(0)

	for changeIdx != -1 {
		// Start of hunk: walk back contextLines for leading context.
		start := changeIdx
		for k := 0; k < contextLines && start > 0 && nops[start-1].kind == ' '; k++ {
			start--
		}

		// Find end of hunk: extend forward through changes and small gaps.
		// A gap (run of context ops) larger than gapThreshold splits the hunk.
		end := changeIdx
		for {
			// Advance past all contiguous changes.
			for end < len(nops) && nops[end].kind != ' ' {
				end++
			}
			// Measure the run of context ops after the last change.
			ctxRun := 0
			for k := end; k < len(nops) && nops[k].kind == ' '; k++ {
				ctxRun++
			}
			// If the gap exceeds the threshold, this hunk ends after
			// contextLines of trailing context (or fewer if the file
			// ends first).
			if ctxRun > gapThreshold {
				end += contextLines
				if end > len(nops) {
					end = len(nops)
				}
				break
			}
			// Gap is small enough to absorb. If there are more changes after
			// the gap, extend through the gap and continue. If no more
			// changes, the hunk extends to the trailing context or EOF.
			nextChangeIdx := nextChange(end)
			if nextChangeIdx == -1 {
				// No more changes. Include up to contextLines of trailing
				// context, then end.
				end += contextLines
				if end > len(nops) {
					end = len(nops)
				}
				break
			}
			// Absorb the gap and continue extending from the next change.
			end = nextChangeIdx
		}

		// Build hunk.
		hunkOldStart := nops[start].oldNo + 1
		hunkNewStart := nops[start].newNo + 1
		oldCount, newCount := 0, 0
		for k := start; k < end; k++ {
			switch nops[k].kind {
			case ' ', '-':
				oldCount++
			}
			switch nops[k].kind {
			case ' ', '+':
				newCount++
			}
		}

		header := formatHunkHeader(hunkOldStart, oldCount, hunkNewStart, newCount)
		var body strings.Builder
		for k := start; k < end; k++ {
			body.WriteByte(nops[k].kind)
			body.WriteString(nops[k].line)
			body.WriteString("\n")
		}

		hunks = append(hunks, hunk{header: header, body: body.String()})

		// Next hunk starts after the trailing context we already included.
		changeIdx = nextChange(end)
	}

	return hunks
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

// D! id=dunif range-end

// D! id=dfmt range-start

func formatHunkHeader(oldStart, oldCount, newStart, newCount int) string {
	return formatRange("@@ -", oldStart, oldCount) + formatRange(" +", newStart, newCount) + " @@"
}

func formatRange(prefix string, start, count int) string {
	if count == 0 {
		return prefix + itoa(start-1) + ",0"
	}
	if count == 1 {
		return prefix + itoa(start)
	}
	return prefix + itoa(start) + "," + itoa(count)
}

// D! id=dfmt range-end
