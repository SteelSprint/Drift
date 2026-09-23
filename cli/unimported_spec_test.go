package cli_test

import (
	"encoding/json"
	"strings"
	"testing"

	"drift/cli"
	"drift/cli/output"
	"drift/internal/testutil"
)

// Integration tests for the unimported-spec-file warning (cli.todo_unimported_warning):
// a *.drift.xml file on disk that is not reachable from main.drift.xml via
// <import> is silently invisible to every scan. todo and list must warn.

func runPlain(dir string, args ...string) (string, int) {
	return cli.RunWithRender(args, dir, output.PlainPresenter{})
}

func TestTodoWarnsOnUnimportedSpecFile(t *testing.T) {
	dir := t.TempDir()
	testutil.WriteSpecFile(t, dir, "main.drift.xml",
		`<main><spec id="a">Main spec.</spec></main>`)
	testutil.WriteSpecFile(t, dir, "requirements.drift.xml",
		`<module name="requirements"><spec id="readme">The README must exist.</spec></module>`)

	if _, code := runPlain(dir, "init"); code != 0 {
		t.Fatalf("init failed")
	}
	out, code := runPlain(dir, "todo")
	if code != 1 && code != 0 {
		t.Fatalf("todo: unexpected code %d\n%s", code, out)
	}
	if !strings.Contains(out, "warning") || !strings.Contains(out, "requirements.drift.xml") {
		t.Fatalf("todo output missing unimported-file warning:\n%s", out)
	}
	if !strings.Contains(out, "import") {
		t.Fatalf("warning should mention the fix (import):\n%s", out)
	}

	// The warning must disappear once the file is imported.
	testutil.WriteSpecFile(t, dir, "main.drift.xml",
		`<main><spec id="a">Main spec.</spec><import path="./requirements.drift.xml" /></main>`)
	out, _ = runPlain(dir, "todo")
	if strings.Contains(out, "requirements.drift.xml is not imported") {
		t.Fatalf("warning still present after import:\n%s", out)
	}
	if !strings.Contains(out, "requirements.readme") {
		t.Fatalf("imported spec should be scanned:\n%s", out)
	}
}

func TestListWarnsOnUnimportedSpecFile(t *testing.T) {
	dir := t.TempDir()
	testutil.WriteSpecFile(t, dir, "main.drift.xml",
		`<main><spec id="a">Main spec.</spec></main>`)
	testutil.WriteSpecFile(t, dir, "requirements.drift.xml",
		`<module name="requirements"><spec id="readme">The README must exist.</spec></module>`)

	if _, code := runPlain(dir, "init"); code != 0 {
		t.Fatalf("init failed")
	}
	out, code := runPlain(dir, "list")
	if code != 0 {
		t.Fatalf("list: unexpected code %d\n%s", code, out)
	}
	if !strings.Contains(out, "requirements.drift.xml is not imported") {
		t.Fatalf("list output missing unimported-file warning:\n%s", out)
	}
}

func TestTodoJSONReportsUnimportedSpecFiles(t *testing.T) {
	dir := t.TempDir()
	testutil.WriteSpecFile(t, dir, "main.drift.xml",
		`<main><spec id="a">Main spec.</spec></main>`)
	testutil.WriteSpecFile(t, dir, "requirements.drift.xml",
		`<module name="requirements"><spec id="readme">The README must exist.</spec></module>`)

	if _, code := runPlain(dir, "init"); code != 0 {
		t.Fatalf("init failed")
	}
	out, _ := cli.RunWithRender([]string{"todo", "--json"}, dir, output.JSONPresenter{})
	var parsed struct {
		UnimportedSpecFiles []string `json:"unimported_spec_files"`
	}
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("json parse: %v\n%s", err, out)
	}
	if len(parsed.UnimportedSpecFiles) != 1 || parsed.UnimportedSpecFiles[0] != "requirements.drift.xml" {
		t.Fatalf("unimported_spec_files = %v, want [requirements.drift.xml]\n%s", parsed.UnimportedSpecFiles, out)
	}
}

func TestInitMentionsImportMechanism(t *testing.T) {
	dir := t.TempDir()
	out, code := runPlain(dir, "init")
	if code != 0 {
		t.Fatalf("init: unexpected code %d\n%s", code, out)
	}
	if !strings.Contains(out, "<import") {
		t.Fatalf("init output should mention <import> for module files:\n%s", out)
	}
}

func TestBrokenEdgeSuggestsQualifiedSpecID(t *testing.T) {
	dir := t.TempDir()
	testutil.WriteSpecFile(t, dir, "main.drift.xml",
		`<main><spec id="intent">The intent.</spec><spec id="impl">Implements <ref spec="intent">the intent</ref>.</spec></main>`)
	if _, code := runPlain(dir, "init"); code != 0 {
		t.Fatalf("init failed")
	}
	// Edit the ref to the unqualified form — a common agent mistake.
	testutil.WriteSpecFile(t, dir, "main.drift.xml",
		`<main><spec id="intent">The intent.</spec><spec id="impl">Implements <ref spec="intent2">the intent</ref>.</spec></main>`)
	out, _ := runPlain(dir, "todo")
	if !strings.Contains(out, "BROKEN-EDGE") {
		t.Fatalf("expected a BROKEN-EDGE event:\n%s", out)
	}
}

func TestBrokenEdgeDidYouMean(t *testing.T) {
	dir := t.TempDir()
	testutil.WriteSpecFile(t, dir, "main.drift.xml",
		`<main><spec id="intent">The intent.</spec><spec id="impl">Implements <ref spec="main.intent">the intent</ref>.</spec></main>`)
	if _, code := runPlain(dir, "init"); code != 0 {
		t.Fatalf("init failed")
	}
	// The ref target "intent" exists only as local id of main.intent.
	testutil.WriteSpecFile(t, dir, "main.drift.xml",
		`<main><spec id="intent">The intent.</spec><spec id="impl">Implements <ref spec="intent">the intent</ref>.</spec></main>`)
	out, _ := runPlain(dir, "todo")
	if !strings.Contains(out, `did you mean "main.intent"`) {
		t.Fatalf("broken-edge output should suggest the qualified id:\n%s", out)
	}
}
