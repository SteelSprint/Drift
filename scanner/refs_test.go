package scanner

import (
	"strings"
	"testing"
)

// TestProcessSpecContent_SelfClosingRef: <ref spec="X" /> is parsed and
// removed from canonical content (no label).
func TestProcessSpecContent_SelfClosingRef(t *testing.T) {
	input := `Overview: see <ref spec="m.other" /> for details.`
	canonical, targets, err := processSpecContent(input)
	if err != nil {
		t.Fatal(err)
	}
	if len(targets) != 1 || targets[0] != "m.other" {
		t.Fatalf("targets = %v, want [m.other]", targets)
	}
	if strings.Contains(canonical, "<ref") {
		t.Fatalf("canonical should have ref stripped: %q", canonical)
	}
}

// TestProcessSpecContent_PairedRef: <ref spec="X">label</ref> is parsed and
// replaced by its label text in canonical content.
func TestProcessSpecContent_PairedRef(t *testing.T) {
	input := `Overview: see <ref spec="m.other">the other spec</ref> for details.`
	canonical, targets, err := processSpecContent(input)
	if err != nil {
		t.Fatal(err)
	}
	if len(targets) != 1 || targets[0] != "m.other" {
		t.Fatalf("targets = %v, want [m.other]", targets)
	}
	if !strings.Contains(canonical, "the other spec") {
		t.Fatalf("canonical should contain label text: %q", canonical)
	}
	if strings.Contains(canonical, "<ref") {
		t.Fatalf("canonical should have ref tag stripped: %q", canonical)
	}
}

// TestProcessSpecContent_SourceOrder: multiple refs appear in targets in
// source order.
func TestProcessSpecContent_SourceOrder(t *testing.T) {
	input := `See <ref spec="m.a" /> then <ref spec="m.b" /> then <ref spec="m.c" />.`
	_, targets, err := processSpecContent(input)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"m.a", "m.b", "m.c"}
	if len(targets) != 3 {
		t.Fatalf("targets = %v, want %v", targets, want)
	}
	for i, w := range want {
		if targets[i] != w {
			t.Fatalf("targets[%d] = %q, want %q", i, targets[i], w)
		}
	}
}

// TestProcessSpecContent_DuplicateRefsDeduped: duplicate (from, to) pairs
// within the same spec are deduped at the scanner level. This tests
// processSpecContent's targets output; the actual dedup to edges happens in
// scanner.go's loop, but processSpecContent should preserve both occurrences
// in targets so the scanner can dedup. (The spec says dedup happens at
// scanner level, not processSpecContent — so targets may contain duplicates.)
func TestProcessSpecContent_DuplicateRefsDeduped(t *testing.T) {
	input := `See <ref spec="m.a" /> and <ref spec="m.a" /> again.`
	_, targets, err := processSpecContent(input)
	if err != nil {
		t.Fatal(err)
	}
	// processSpecContent returns all refs in source order; the scanner loop
	// dedups before emitting edges. Verify both are returned here.
	if len(targets) != 2 || targets[0] != "m.a" || targets[1] != "m.a" {
		t.Fatalf("targets = %v, want [m.a m.a] (dedup happens at scanner level)", targets)
	}
}

// TestProcessSpecContent_MalformedRefTagErrors: a <ref-looking tag that fails
// the strict grammar (e.g. <ref sprc="X"> with a typo in the attribute name)
// causes a scan error.
func TestProcessSpecContent_MalformedRefTagErrors(t *testing.T) {
	input := `See <ref sprc="m.a">typo</ref> here.`
	_, _, err := processSpecContent(input)
	if err == nil {
		t.Fatal("expected error for malformed <ref sprc=...> tag (attribute typo)")
	}
}

// TestProcessSpecContent_MissingSpecAttrErrors: <ref> without spec attribute
// causes a scan error.
func TestProcessSpecContent_MissingSpecAttrErrors(t *testing.T) {
	input := `See <ref>no spec</ref> here.`
	_, _, err := processSpecContent(input)
	if err == nil {
		t.Fatal("expected error for <ref> without spec attribute")
	}
}

// TestProcessSpecContent_SingleQuoteRejected: single-quoted spec attribute is
// rejected by the strict grammar.
func TestProcessSpecContent_SingleQuoteRejected(t *testing.T) {
	input := `See <ref spec='m.a'>single</ref> here.`
	_, _, err := processSpecContent(input)
	if err == nil {
		t.Fatal("expected error for single-quoted spec attribute (strict grammar requires double quotes)")
	}
}

// TestProcessSpecContent_ReorderedAttrRejected: <ref foo="bar" spec="X"> is
// rejected (spec must be the first attribute).
func TestProcessSpecContent_ReorderedAttrRejected(t *testing.T) {
	input := `See <ref foo="bar" spec="m.a">reordered</ref> here.`
	_, _, err := processSpecContent(input)
	if err == nil {
		t.Fatal("expected error for reordered attributes (spec must be first)")
	}
}

// TestProcessSpecContent_NoRefs: content without refs returns trimmed content
// and empty targets.
func TestProcessSpecContent_NoRefs(t *testing.T) {
	input := `  Just plain text, no refs here.  `
	canonical, targets, err := processSpecContent(input)
	if err != nil {
		t.Fatal(err)
	}
	if len(targets) != 0 {
		t.Fatalf("targets = %v, want empty", targets)
	}
	if canonical != strings.TrimSpace(input) {
		t.Fatalf("canonical = %q, want %q", canonical, strings.TrimSpace(input))
	}
}

// TestProcessSpecContent_NonexistentRefNoError: refs to nonexistent spec IDs
// do NOT error at scan time — they surface as broken-edge drift items at todo
// time.
func TestProcessSpecContent_NonexistentRefNoError(t *testing.T) {
	input := `See <ref spec="m.doesnotexist" /> here.`
	_, targets, err := processSpecContent(input)
	if err != nil {
		t.Fatalf("nonexistent ref should not error at scan time: %v", err)
	}
	if len(targets) != 1 || targets[0] != "m.doesnotexist" {
		t.Fatalf("targets = %v, want [m.doesnotexist]", targets)
	}
}
