package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"drift/cli/output"
	"drift/internal/testutil"
)

// setupCoverageConfigProject builds a deterministic fixture:
//   - code.go: 10 lines, 5 covered by a linked marker
//   - legacy/old.go: 10 lines, unmarked
//   - overall: 5 of 20 walked lines (25%), legacy group 0 of 10 (0%)
func setupCoverageConfigProject(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	testutil.WriteSpecFile(t, dir, "main.drift.xml",
		`<module name="m">
<spec id="s">Validate input.</spec>
</module>`)
	testutil.WriteCodeFile(t, dir, "code.go",
		"line1\nline2\n// D! id=cval range-start\nline4\nline5\nline6\nline7\n// D! id=cval range-end\nline9\nline10\n")
	os.MkdirAll(filepath.Join(dir, "legacy"), 0755)
	testutil.WriteCodeFile(t, dir, "legacy/old.go",
		"l1\nl2\nl3\nl4\nl5\nl6\nl7\nl8\nl9\nl10\n")

	if out, code := RunWithRender([]string{"init"}, dir, output.PlainPresenter{}); code != 0 {
		t.Fatalf("init: code=%d out=%s", code, out)
	}
	if out, code := RunWithRender([]string{"link", "cval", "m.s"}, dir, output.PlainPresenter{}); code != 0 {
		t.Fatalf("link: code=%d out=%s", code, out)
	}
	return dir
}

func writeCoverageConfig(t *testing.T, dir, xml string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(dir, ".drift"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".drift", "coverage.xml"), []byte(xml), 0644); err != nil {
		t.Fatal(err)
	}
}

func writeRawCoverageConfig(t *testing.T, dir, xml string) {
	writeCoverageConfig(t, dir, xml)
}

// TestCoverageCheck_NoConfig: --check without .drift/coverage.xml is a usage
// error whose message documents the config file. Plain coverage stays exit 0.
func TestCoverageCheck_NoConfig(t *testing.T) {
	dir := setupCoverageConfigProject(t)

	out, code := RunWithRender([]string{"coverage", "--check"}, dir, output.PlainPresenter{})
	if code != 1 {
		t.Fatalf("--check without config: code=%d out=%s", code, out)
	}
	if !strings.Contains(out, "coverage.xml") {
		t.Fatalf("--check without config must name the config file, got:\n%s", out)
	}

	// Plain coverage remains informational: exit 0, no target line.
	out, code = RunWithRender([]string{"coverage"}, dir, output.PlainPresenter{})
	if code != 0 {
		t.Fatalf("plain coverage: code=%d out=%s", code, out)
	}
	if strings.Contains(out, "Target:") {
		t.Fatalf("plain coverage without config must not print a target line:\n%s", out)
	}
}

// TestCoverageCheck_GlobalTarget: overall covered/total vs the global target.
func TestCoverageCheck_GlobalTarget(t *testing.T) {
	dir := setupCoverageConfigProject(t)
	writeCoverageConfig(t, dir, `<coverage><target lines="80"/></coverage>`)

	// Plain coverage stays exit 0 but must show the target verdict.
	out, code := RunWithRender([]string{"coverage"}, dir, output.PlainPresenter{})
	if code != 0 {
		t.Fatalf("plain coverage with config: code=%d out=%s", code, out)
	}
	if !strings.Contains(out, "Target: 80%") || !strings.Contains(out, "FAIL") {
		t.Fatalf("plain coverage must show target and verdict:\n%s", out)
	}

	// --check fails below target and names the shortfall.
	out, code = RunWithRender([]string{"coverage", "--check"}, dir, output.PlainPresenter{})
	if code != 1 {
		t.Fatalf("--check below target: code=%d out=%s", code, out)
	}
	if !strings.Contains(out, "20%") || !strings.Contains(out, "80%") {
		t.Fatalf("--check must name current vs target:\n%s", out)
	}

	// A lenient target passes.
	writeCoverageConfig(t, dir, `<coverage><target lines="20"/></coverage>`)
	if _, code := RunWithRender([]string{"coverage", "--check"}, dir, output.PlainPresenter{}); code != 0 {
		out, code := RunWithRender([]string{"coverage", "--check"}, dir, output.PlainPresenter{})
		t.Fatalf("--check above target must pass: code=%d out=%s", code, out)
	}
}

// TestCoverageCheck_PathTarget: per-path overrides apply to the most
// specific matching prefix; unmatched files fall back to the global target.
func TestCoverageCheck_PathTarget(t *testing.T) {
	dir := setupCoverageConfigProject(t)
	writeCoverageConfig(t, dir,
		`<coverage><target lines="20"/><path pattern="legacy/" target="10"/></coverage>`)

	// Overall 25% >= 20 passes; legacy group 0% < 10 fails → check fails.
	out, code := RunWithRender([]string{"coverage", "--check"}, dir, output.PlainPresenter{})
	if code != 1 {
		t.Fatalf("per-path shortfall must fail the check: code=%d out=%s", code, out)
	}
	if !strings.Contains(out, "legacy/") {
		t.Fatalf("failure must name the failing path:\n%s", out)
	}

	// A generous per-path target lets everything pass.
	writeCoverageConfig(t, dir,
		`<coverage><target lines="20"/><path pattern="legacy/" target="0"/></coverage>`)
	if _, code := RunWithRender([]string{"coverage", "--check"}, dir, output.PlainPresenter{}); code != 0 {
		t.Fatalf("satisfied path targets must pass")
	}
}

// TestCoverageCheck_InvalidConfig: malformed settings fail loud, naming the
// problem — never a silent fallback to "no config".
func TestCoverageCheck_InvalidConfig(t *testing.T) {
	for _, bad := range []string{
		`<coverage><target lines="abc"/></coverage>`,
		`<coverage><target lines="140"/></coverage>`,
		`<coverage><target/></coverage>`,
		`<coverage><path pattern="x/" target="50"/></coverage>`,
	} {
		dir := setupCoverageConfigProject(t)
		writeRawCoverageConfig(t, dir, bad)
		out, code := RunWithRender([]string{"coverage", "--check"}, dir, output.PlainPresenter{})
		if code == 0 {
			t.Fatalf("invalid config %q must not pass", bad)
		}
		if strings.Contains(out, "PASS") {
			t.Fatalf("invalid config must not PASS:\n%s", out)
		}
	}
}
