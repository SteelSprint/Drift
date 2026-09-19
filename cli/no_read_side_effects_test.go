package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"drift/cli/commands"
	"drift/cli/output"
)

// TestNoReadSideEffects_BareInvocation: bare `drift` and `drift help` must
// not create .drift/ in an uninitialized folder (principles.no_read_side_effects).
func TestNoReadSideEffects_BareInvocation(t *testing.T) {
	dir := t.TempDir()
	for _, args := range [][]string{{}, {"help"}, {"skill"}, {"version"}} {
		if _, code := RunWithRender(args, dir, output.PlainPresenter{}); code != 0 {
			t.Fatalf("drift %v: code=%d", args, code)
		}
		if _, err := os.Stat(filepath.Join(dir, ".drift")); !os.IsNotExist(err) {
			t.Fatalf("drift %v must not create .drift/", args)
		}
	}
}

// TestNoReadSideEffects_UnknownCommand: an unknown command must not create .drift/.
func TestNoReadSideEffects_UnknownCommand(t *testing.T) {
	dir := t.TempDir()
	_, code := RunWithRender([]string{"nosuchcmd"}, dir, output.PlainPresenter{})
	if code != 1 {
		t.Fatalf("unknown command: code=%d", code)
	}
	if _, err := os.Stat(filepath.Join(dir, ".drift")); !os.IsNotExist(err) {
		t.Fatalf("unknown command must not create .drift/")
	}
}

// TestDispatchLockModes_NotInitialized: every LockRequire command fails loud
// with exit 2, a prescriptive message, and no .drift/ directory created.
func TestDispatchLockModes_NotInitialized(t *testing.T) {
	for _, cmd := range []string{"todo", "list", "show", "coverage", "diff", "link", "unlink", "reset", "config"} {
		dir := t.TempDir()
		out, code := RunWithRender([]string{cmd}, dir, output.PlainPresenter{})
		if code != 2 {
			t.Fatalf("%s (uninitialized): code=%d out=%s", cmd, code, out)
		}
		if !strings.Contains(out, "not initialized") || !strings.Contains(out, "drift init") {
			t.Fatalf("%s (uninitialized): expected prescriptive message, got:\n%s", cmd, out)
		}
		if _, err := os.Stat(filepath.Join(dir, ".drift")); !os.IsNotExist(err) {
			t.Fatalf("%s (uninitialized) must not create .drift/", cmd)
		}
	}
}

// TestDispatchLockModes_Initialized: LockRequire commands run normally on an
// initialized project (todo on a fresh init exits 0/1 — not 2).
func TestDispatchLockModes_Initialized(t *testing.T) {
	dir := t.TempDir()
	if _, code := RunWithRender([]string{"init"}, dir, output.PlainPresenter{}); code != 0 {
		t.Fatalf("init: code=%d", code)
	}
	_, code := RunWithRender([]string{"todo"}, dir, output.PlainPresenter{})
	if code == 2 {
		t.Fatalf("todo on initialized project must not exit 2")
	}
}

// TestLockMode_MetaCoverage: every registered command declares an explicit
// Lock mode; only init is LockInit; only help/skill/version are LockNone.
func TestLockMode_MetaCoverage(t *testing.T) {
	for name, cmd := range Registry {
		switch cmd.Meta().Lock {
		case commands.LockInit:
			if name != "init" {
				t.Fatalf("%s: LockInit reserved for init", name)
			}
		case commands.LockNone:
			if name != "help" && name != "skill" && name != "version" {
				t.Fatalf("%s: LockNone must be a deliberate state-free declaration", name)
			}
		}
	}
}
