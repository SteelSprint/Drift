package cli_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"drift/cli"
	"drift/cli/output"
	"drift/internal/testutil"
)

// setupFrictionProject creates a project with N drifted markers so we have
// N closures to reset. Returns the dir and the list of closure hashes.
func setupFrictionProject(t *testing.T, n int) (dir string, hashes []string) {
	t.Helper()
	dir = t.TempDir()
	testutil.WriteSpecFile(t, dir, "main.drift.xml", `<module name="m">`+
		strings.Repeat("<spec id=\"s\">spec.</spec>", 0)+
		buildNSpecs(n)+
		`</module>`)
	for i := 0; i < n; i++ {
		id := markerID(i)
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
	for i := 0; i < n; i++ {
		if out, code := run("link", markerID(i), "m."+specID(i)); code != 0 {
			t.Fatalf("link %d failed: %d\n%s", i, code, out)
		}
	}
	// Mutate all markers so all drift.
	for i := 0; i < n; i++ {
		id := markerID(i)
		testutil.WriteCodeFile(t, dir, id+".go",
			"// D! id="+id+" range-start\npackage main\nvar _ = "+itoa(i)+"\n// D! id="+id+" range-end\n")
	}
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
	if len(parsed.Closures) != n {
		t.Fatalf("expected %d closures, got %d\n%s", n, len(parsed.Closures), out)
	}
	for _, c := range parsed.Closures {
		hashes = append(hashes, c.Hash)
	}
	return dir, hashes
}

func markerID(i int) string { return "m" + itoa(i) }
func specID(i int) string   { return "s" + itoa(i) }

// itoa is a tiny strconv.Itoa without the import (keeps test deps clean).
func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	neg := false
	if i < 0 {
		neg = true
		i = -i
	}
	var buf [20]byte
	pos := len(buf)
	for i > 0 {
		pos--
		buf[pos] = byte('0' + i%10)
		i /= 10
	}
	if neg {
		pos--
		buf[pos] = '-'
	}
	return string(buf[pos:])
}

func buildNSpecs(n int) string {
	var sb strings.Builder
	for i := 0; i < n; i++ {
		sb.WriteString("<spec id=\"" + specID(i) + "\">spec " + itoa(i) + ".</spec>")
	}
	return sb.String()
}

// TestResetFrictionBlock_BlocksFourthReset verifies that the 4th rapid reset
// is blocked with exit 2 and the message names the friction principle but
// does NOT advertise the override flag.
func TestResetFrictionBlock_BlocksFourthReset(t *testing.T) {
	dir, hashes := setupFrictionProject(t, 4)
	run := func(args ...string) (string, int) {
		return cli.RunWithRender(args, dir, output.PlainPresenter{})
	}
	// 3 rapid resets: all succeed.
	for i := 0; i < 3; i++ {
		if out, code := run("reset", hashes[i]); code != 0 {
			t.Fatalf("reset %d (hash=%s) expected code 0, got %d\n%s", i, hashes[i], code, out)
		}
	}
	// 4th rapid reset: blocked with exit 2.
	out, code := run("reset", hashes[3])
	if code != 2 {
		t.Fatalf("expected 4th reset to block with exit 2, got %d\n%s", code, out)
	}
	if !strings.Contains(out, "friction") {
		t.Errorf("expected block message to mention friction: %s", out)
	}
	if strings.Contains(out, "--dangerously-override-friction") {
		t.Errorf("block message MUST NOT advertise the override flag (R1): %s", out)
	}
	if strings.Contains(out, "bypass") {
		t.Errorf("block message MUST NOT mention bypass: %s", out)
	}
}

// TestResetFrictionBlock_OverrideBypassesAndSquawks verifies that
// --dangerously-override-friction bypasses the block, emits a stderr
// squawk, and populates the Warning field in JSON output.
func TestResetFrictionBlock_OverrideBypassesAndSquawks(t *testing.T) {
	dir, hashes := setupFrictionProject(t, 4)
	run := func(args ...string) (string, int) {
		return cli.RunWithRender(args, dir, output.PlainPresenter{})
	}
	// Trip the rate limit with 3 resets.
	for i := 0; i < 3; i++ {
		run("reset", hashes[i])
	}
	// Capture stderr — RunWithRender routes stderr from the override path
	// through the harness; we read it from the rendered output's Warning
	// field via JSON mode instead.
	out, code := cli.RunWithRender(
		[]string{"reset", "--dangerously-override-friction", hashes[3]},
		dir, output.JSONPresenter{})
	if code != 0 {
		t.Fatalf("override reset expected exit 0, got %d\n%s", code, out)
	}
	var parsed struct {
		Warning string `json:"warning"`
	}
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("json parse: %v\n%s", err, out)
	}
	if parsed.Warning == "" {
		t.Errorf("expected warning field populated on override, got: %s", out)
	}
}

// TestResetFrictionBlock_DryRunDoesNotCount verifies that --dry-run resets
// neither trip the rate limit nor record a timestamp.
func TestResetFrictionBlock_DryRunDoesNotCount(t *testing.T) {
	dir, hashes := setupFrictionProject(t, 4)
	run := func(args ...string) (string, int) {
		return cli.RunWithRender(args, dir, output.PlainPresenter{})
	}
	// 3 dry-run resets: should not count.
	for i := 0; i < 3; i++ {
		if out, code := run("reset", "--dry-run", hashes[i]); code != 3 {
			t.Fatalf("dry-run reset %d expected exit 3, got %d\n%s", i, code, out)
		}
	}
	// No friction.json should exist (dry-run doesn't record).
	frictionPath := filepath.Join(dir, ".drift", "friction.json")
	if _, err := os.Stat(frictionPath); err == nil {
		t.Errorf("dry-run resets must not write friction.json")
	}
	// First real reset (4th invocation but 0 recorded) should succeed.
	if out, code := run("reset", hashes[0]); code != 0 {
		t.Fatalf("first real reset expected exit 0, got %d\n%s", code, out)
	}
}

// TestResetFrictionBlock_AfterWindowExpires confirms the limit resets
// after the window passes. Uses a mutated friction.json with old timestamps.
func TestResetFrictionBlock_AfterWindowExpires(t *testing.T) {
	dir, hashes := setupFrictionProject(t, 4)
	run := func(args ...string) (string, int) {
		return cli.RunWithRender(args, dir, output.PlainPresenter{})
	}
	// 3 rapid resets to trip the limit.
	for i := 0; i < 3; i++ {
		run("reset", hashes[i])
	}
	// 4th blocked.
	if _, code := run("reset", hashes[3]); code != 2 {
		t.Fatalf("expected block on 4th, got %d", code)
	}
	// Age out: rewrite friction.json with old timestamps.
	old := time.Now().Unix() - 60
	oldData := []byte(`{"resets":[` + itoa64(old-1) + `,` + itoa64(old-2) + `,` + itoa64(old-3) + `]}`)
	if err := os.WriteFile(filepath.Join(dir, ".drift", "friction.json"), oldData, 0644); err != nil {
		t.Fatal(err)
	}
	// Now should succeed.
	if out, code := run("reset", hashes[3]); code != 0 {
		t.Fatalf("expected exit 0 after aging out, got %d\n%s", code, out)
	}
}

// TestResetFrictionBlock_RapidResetsUnderLockConcurrent confirms that
// concurrent resets don't corrupt friction.json — the Session lock
// serializes the read-modify-write window.
func TestResetFrictionBlock_RapidResetsUnderLockConcurrent(t *testing.T) {
	dir, hashes := setupFrictionProject(t, 8)
	var wg sync.WaitGroup
	results := make([]int, len(hashes))
	for i, h := range hashes {
		wg.Add(1)
		go func(idx int, hash string) {
			defer wg.Done()
			_, code := cli.RunWithRender([]string{"reset", "--dangerously-override-friction", hash}, dir, output.PlainPresenter{})
			results[idx] = code
		}(i, h)
	}
	wg.Wait()
	succeed := 0
	for _, c := range results {
		if c == 0 {
			succeed++
		}
	}
	if succeed != len(hashes) {
		t.Errorf("expected all %d override resets to succeed, got %d", len(hashes), succeed)
	}
	// friction.json should be valid JSON after concurrent writes.
	data, err := os.ReadFile(filepath.Join(dir, ".drift", "friction.json"))
	if err != nil {
		t.Fatal(err)
	}
	var ff struct {
		Resets []int64 `json:"resets"`
	}
	if err := json.Unmarshal(data, &ff); err != nil {
		t.Fatalf("friction.json corrupted after concurrent writes: %v\n%s", err, string(data))
	}
	if len(ff.Resets) != len(hashes) {
		t.Errorf("expected %d recorded timestamps, got %d", len(hashes), len(ff.Resets))
	}
}

func itoa64(n int64) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	pos := len(buf)
	for n > 0 {
		pos--
		buf[pos] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[pos:])
}
