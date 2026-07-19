package cli_test

import (
	"strings"
	"testing"

	"drift/cli"
	"drift/cli/output"
	"drift/internal/testutil"
)

// TestCLI_ClosureWorkflow exercises the closure-driven UX end-to-end:
// init → link → drift → todo shows closures → reset by hash → clean.
func TestCLI_ClosureWorkflow(t *testing.T) {
	dir := t.TempDir()
	testutil.WriteSpecFile(t, dir, "main.drift.xml",
		`<module name="m">
<spec id="validate">Validate input.</spec>
</module>`)
	testutil.WriteCodeFile(t, dir, "code.go",
		"// D! id=cval range-start\npackage main\nfunc validate() {}\n// D! id=cval range-end\n")

	run := func(args ...string) (string, int) {
		return cli.RunWithRender(args, dir, output.PlainPresenter{})
	}

	// Init + link.
	if out, code := run("init"); code != 0 {
		t.Fatalf("init: code=%d out=%s", code, out)
	}
	if out, code := run("link", "cval", "m.validate"); code != 0 {
		t.Fatalf("link: code=%d out=%s", code, out)
	}

	// Baseline should be clean.
	out, code := run("todo")
	if code != 0 {
		t.Fatalf("clean todo: code=%d out=%s", code, out)
	}
	if !strings.Contains(out, "No changes detected") {
		t.Fatalf("clean todo: unexpected output: %s", out)
	}

	// Mutate the spec.
	testutil.WriteSpecFile(t, dir, "main.drift.xml",
		`<module name="m">
<spec id="validate">Validate input more strictly.</spec>
</module>`)

	// todo should show 1 closure.
	out, code = run("todo")
	if code != 1 {
		t.Fatalf("drifted todo: code=%d out=%s", code, out)
	}
	if !strings.Contains(out, "Closure") {
		t.Fatalf("expected closure output: %s", out)
	}

	// Extract hash from output.
	lines := strings.Split(out, "\n")
	var hash string
	for _, l := range lines {
		if strings.Contains(l, "Closure ") {
			// "Closure abc12345  (...)"
			parts := strings.SplitN(l, " ", 3)
			if len(parts) >= 2 {
				hash = parts[1]
			}
			break
		}
	}
	if hash == "" {
		t.Fatalf("could not extract closure hash from output:\n%s", out)
	}

	// Reset by hash.
	out, code = run("reset", hash)
	if code != 0 {
		t.Fatalf("reset: code=%d out=%s", code, out)
	}

	// todo should now be clean.
	out, code = run("todo")
	if code != 0 {
		t.Fatalf("post-reset todo: code=%d out=%s", code, out)
	}
	if !strings.Contains(out, "No changes detected") {
		t.Fatalf("post-reset todo should be clean: %s", out)
	}
}

// TestCLI_JSONTodo: JSON output structure for closures.
func TestCLI_JSONTodo(t *testing.T) {
	dir := t.TempDir()
	testutil.WriteSpecFile(t, dir, "main.drift.xml",
		`<module name="m">
<spec id="a">A spec.</spec>
</module>`)
	testutil.WriteCodeFile(t, dir, "code.go",
		"// D! id=ca range-start\npackage main\n// D! id=ca range-end\n")

	run := func(args ...string) (string, int) {
		return cli.RunWithRender(args, dir, output.JSONPresenter{})
	}
	if _, code := run("init"); code != 0 {
		t.Fatal("init failed")
	}
	if _, code := run("link", "ca", "m.a"); code != 0 {
		t.Fatal("link failed")
	}

	// Drift the spec.
	testutil.WriteSpecFile(t, dir, "main.drift.xml",
		`<module name="m">
<spec id="a">A spec that changed.</spec>
</module>`)

	out, code := run("todo")
	if code != 1 {
		t.Fatalf("todo: code=%d out=%s", code, out)
	}
	if !strings.Contains(out, `"closures"`) {
		t.Fatalf("JSON missing closures field: %s", out)
	}
	if !strings.Contains(out, `"hash"`) {
		t.Fatalf("JSON missing hash field: %s", out)
	}
}
