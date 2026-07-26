package scanner

import (
	"os"
	"path/filepath"
	"testing"
)

// TestReadMarkerContent_R1SpecContract verifies ReadMarkerContent against the
// literal spec contract in scanner.marker_content_read R1:
//   startLine = "the line number (1-indexed) immediately after the range-start marker"
//   endLine   = "the line number (1-indexed) of the range-end marker line itself"
//   R1: return the half-open range [startLine, endLine-1], 1-indexed.
//
// The caller (orchestrator.writeBaseline) passes m.LineNumber (the range-start
// marker's line) as startLine, NOT "the line immediately after". This test
// exercises the spec's literal defined terms, not the caller's workaround.
func TestReadMarkerContent_R1SpecContract(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.go")

	// Line 1: package
	// Line 2: (blank)
	// Line 3: // D! id=foo range-start
	// Line 4: content A
	// Line 5: // D! id=foo range-end
	content := "package main\n\n// D! id=foo range-start\ncontent A\n// D! id=foo range-end\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	// Per spec defined terms:
	//   startLine = line immediately after range-start = 4
	//   endLine   = line of range-end = 5
	// R1: return [4, 4] = "content A\n"
	got, err := ReadMarkerContent(path, 4, 5)
	if err != nil {
		t.Fatal(err)
	}
	want := "content A\n"
	if got != want {
		t.Errorf("ReadMarkerContent(path, 4, 5) = %q, want %q\n"+
			"Per spec R1: startLine=4 (line after range-start), endLine=5 (range-end line),\n"+
			"half-open [4,4] should return line 4 only.", got, want)
	}
}

// TestReadMarkerContent_R1MultiLine verifies the half-open range with multiple
// content lines, per the spec's defined terms.
func TestReadMarkerContent_R1MultiLine(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.go")

	// Line 1: // D! id=bar range-start
	// Line 2: alpha
	// Line 3: beta
	// Line 4: gamma
	// Line 5: // D! id=bar range-end
	content := "// D! id=bar range-start\nalpha\nbeta\ngamma\n// D! id=bar range-end\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	// startLine = 2 (line after range-start), endLine = 5 (range-end line)
	// R1: [2, 4] = "alpha\nbeta\ngamma\n"
	got, err := ReadMarkerContent(path, 2, 5)
	if err != nil {
		t.Fatal(err)
	}
	want := "alpha\nbeta\ngamma\n"
	if got != want {
		t.Errorf("ReadMarkerContent(path, 2, 5) = %q, want %q", got, want)
	}
}

// TestReadMarkerContent_R2NestedBlanking verifies nested marker declarations
// are blanked (R2), per the spec's defined terms for startLine/endLine.
func TestReadMarkerContent_R2NestedBlanking(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.go")

	// Line 1: // D! id=outer range-start
	// Line 2: before
	// Line 3: // D! id=inner range-start
	// Line 4: inner-content
	// Line 5: // D! id=inner range-end
	// Line 6: after
	// Line 7: // D! id=outer range-end
	content := "// D! id=outer range-start\nbefore\n// D! id=inner range-start\ninner-content\n// D! id=inner range-end\nafter\n// D! id=outer range-end\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	// startLine = 2, endLine = 7
	// Lines [2, 6] = before, (blanked inner range-start), inner-content, (blanked inner range-end), after
	got, err := ReadMarkerContent(path, 2, 7)
	if err != nil {
		t.Fatal(err)
	}
	// Nested marker declarations stripped from "D!" to end of line; comment prefix preserved.
	// The "// " prefix before "D!" is preserved.
	want := "before\n// \ninner-content\n// \nafter\n"
	if got != want {
		t.Errorf("ReadMarkerContent nested blanking = %q, want %q", got, want)
	}
}

// TestReadMarkerContent_R3PastEOF verifies R3: range past end of file returns
// available lines without error.
func TestReadMarkerContent_R3PastEOF(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.go")

	// Line 1: // D! id=z range-start
	// Line 2: only
	// (no range-end in file — but caller may pass an endLine beyond EOF)
	content := "// D! id=z range-start\nonly\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	// startLine = 2, endLine = 10 (past EOF)
	got, err := ReadMarkerContent(path, 2, 10)
	if err != nil {
		t.Fatalf("R3: unexpected error: %v", err)
	}
	want := "only\n"
	if got != want {
		t.Errorf("R3: ReadMarkerContent(path, 2, 10) = %q, want %q", got, want)
	}
}

// TestReadMarkerContent_R4EmptyReturnsEmpty verifies R4: empty result is the
// empty string (no trailing newline).
func TestReadMarkerContent_R4EmptyReturnsEmpty(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.go")

	// Line 1: // D! id=e range-start
	// Line 2: // D! id=e range-end
	// startLine = 2, endLine = 2 → [2, 1] is empty
	content := "// D! id=e range-start\n// D! id=e range-end\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := ReadMarkerContent(path, 2, 2)
	if err != nil {
		t.Fatal(err)
	}
	if got != "" {
		t.Errorf("R4: empty range returned %q, want %q", got, "")
	}
}

// TestReadMarkerContent_CallerConvention verifies the caller's convention
// (startLine = m.LineNumber+1, the line after the range-start marker).
// This matches the spec's defined terms and the orchestrator/build callers.
func TestReadMarkerContent_CallerConvention(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.go")

	// Line 1: // D! id=foo range-start
	// Line 2: content A
	// Line 3: // D! id=foo range-end
	content := "// D! id=foo range-start\ncontent A\n// D! id=foo range-end\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	// Caller passes m.LineNumber+1=2 (line after range-start), m.EndLineNumber=3 (range-end line).
	got, err := ReadMarkerContent(path, 2, 3)
	if err != nil {
		t.Fatal(err)
	}
	want := "content A\n"
	if got != want {
		t.Errorf("caller convention ReadMarkerContent(path, 2, 3) = %q, want %q", got, want)
	}
}
