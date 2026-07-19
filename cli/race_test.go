package cli_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"drift/cli"
	"drift/cli/output"
	"drift/internal/testutil"
)

// TestConcurrentResetClosureRace sets up a workspace with 3 distinct drift
// closures (3 drifted markers), then resets all 3 concurrently. The state
// file lock must serialize the Load→modify→Save windows; without the lock,
// concurrent resets would silently overwrite each other and lose work.
//
// Regression guard: this test FAILS if locking is removed or broken, because
// some closures would remain unresolved after the parallel burst.
func TestConcurrentResetClosureRace(t *testing.T) {
	dir := t.TempDir()
	testutil.WriteSpecFile(t, dir, "main.drift.xml",
		`<module name="m">
<spec id="a">A spec.</spec>
<spec id="b">B spec.</spec>
<spec id="c">C spec.</spec>
</module>`)
	// Three markers, each in its own code file so we can mutate independently.
	for _, id := range []string{"ca", "cb", "cc"} {
		testutil.WriteCodeFile(t, dir, id+".go",
			"// D! id="+id+" range-start\npackage main\n// D! id="+id+" range-end\n")
	}

	run := func(args ...string) (string, int) {
		return cli.RunWithRender(args, dir, output.PlainPresenter{})
	}
	runJSON := func(args ...string) (string, int) {
		return cli.RunWithRender(args, dir, output.JSONPresenter{})
	}

	if _, code := run("init"); code != 0 {
		t.Fatalf("init failed")
	}
	for _, pair := range [][2]string{{"ca", "m.a"}, {"cb", "m.b"}, {"cc", "m.c"}} {
		if out, code := run("link", pair[0], pair[1]); code != 0 {
			t.Fatalf("link %v failed: %d\n%s", pair, code, out)
		}
	}

	// Mutate all 3 markers' code so all 3 drift independently.
	for _, id := range []string{"ca", "cb", "cc"} {
		testutil.WriteCodeFile(t, dir, id+".go",
			"// D! id="+id+" range-start\npackage main\nvar _ = 1\n// D! id="+id+" range-end\n")
	}

	// Parse closure hashes from todo --json.
	out, code := runJSON("todo", "--json")
	if code != 1 {
		t.Fatalf("todo --json: expected code 1, got %d\n%s", code, out)
	}
	var parsed struct {
		Closures []struct {
			Hash string `json:"hash"`
		} `json:"closures"`
	}
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("json parse: %v\n%s", err, out)
	}
	if len(parsed.Closures) != 3 {
		t.Fatalf("expected 3 closures, got %d\n%s", len(parsed.Closures), out)
	}
	hashes := make([]string, 3)
	for i, c := range parsed.Closures {
		hashes[i] = c.Hash
	}

	// Snapshot baseline marker hashes for later comparison.
	baselineStatePath := filepath.Join(dir, ".drift", "state.xml")
	baselineSnapshot, err := os.ReadFile(baselineStatePath)
	if err != nil {
		t.Fatal(err)
	}

	// Reset all 3 closures concurrently. Without the lock, two of the three
	// Save calls would clobber each other and some resets would be lost.
	var wg sync.WaitGroup
	for _, h := range hashes {
		wg.Add(1)
		go func(hash string) {
			defer wg.Done()
			run("reset", hash)
		}(h)
	}
	wg.Wait()

	// After all resets, the state file must reflect ALL 3 baseline updates.
	// If any reset was silently overwritten, the next todo will report drift.
	out, code = run("todo")
	if code != 0 {
		postResetState, _ := os.ReadFile(baselineStatePath)
		t.Fatalf("post-reset todo not clean (code=%d). Some reset was lost to a race.\nbaseline before resets:\n%s\nstate after resets:\n%s\ntodo output:\n%s",
			code,
			string(baselineSnapshot),
			string(postResetState),
			out)
	}
}
