package statestore_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"drift/core"
	"drift/internal/fileio"
	"drift/statestore"
)

// beginTestSession creates a fresh project dir + .drift/ and returns a
// fileio.Session rooted there. The Session is closed automatically via
// t.Cleanup.
func beginTestSession(t *testing.T, dir string) *fileio.Session {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(dir, ".drift"), 0755); err != nil {
		t.Fatal(err)
	}
	sess, err := fileio.Begin(dir)
	if err != nil {
		t.Fatalf("fileio.Begin: %v", err)
	}
	t.Cleanup(func() { sess.Close() })
	return sess
}

func TestFileStateStore_SaveLoad_v4(t *testing.T) {
	dir := t.TempDir()
	sess := beginTestSession(t, dir)
	store := statestore.NewFileStateStore(dir)

	want := statestore.State{
		Specs:   []core.Spec{{ID: "m.a", Hash: "aaa", Filepath: "a.xml", LineNumber: 1}},
		Markers: []core.Marker{{ID: "cval", Hash: "mmm", Filepath: "a.go", LineNumber: 10, EndLineNumber: 20}},
		Edges:   []core.Edge{{From: "cval", To: "m.a"}},
	}
	if err := store.Save(sess, want); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := store.Load(sess)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(got.Specs) != 1 || got.Specs[0] != want.Specs[0] {
		t.Fatalf("specs round-trip failed: got %+v want %+v", got.Specs, want.Specs)
	}
	if len(got.Markers) != 1 || got.Markers[0] != want.Markers[0] {
		t.Fatalf("markers round-trip failed: got %+v want %+v", got.Markers, want.Markers)
	}
	if len(got.Edges) != 1 || got.Edges[0] != want.Edges[0] {
		t.Fatalf("edges round-trip failed: got %+v want %+v", got.Edges, want.Edges)
	}
}

func TestFileStateStore_RefusesPreV4(t *testing.T) {
	dir := t.TempDir()
	sess := beginTestSession(t, dir)
	// Write a v3 file (with edgeResolutions).
	v3Content := `<?xml version="1.0" encoding="UTF-8"?>
<drift version="3">
  <specs></specs>
  <markers></markers>
  <edges></edges>
  <edgeResolutions></edgeResolutions>
</drift>
`
	if err := os.WriteFile(filepath.Join(dir, ".drift", "state.xml"), []byte(v3Content), 0644); err != nil {
		t.Fatal(err)
	}
	store := statestore.NewFileStateStore(dir)
	_, err := store.Load(sess)
	if err == nil {
		t.Fatalf("expected error for v3 file, got nil")
	}
}

func TestFileStateStore_Initialized(t *testing.T) {
	dir := t.TempDir()
	store := statestore.NewFileStateStore(dir)
	if ok, _ := store.Initialized(); ok {
		t.Fatal("expected not initialized")
	}
	sess := beginTestSession(t, dir)
	if err := store.Save(sess, statestore.State{}); err != nil {
		t.Fatal(err)
	}
	if ok, _ := store.Initialized(); !ok {
		t.Fatal("expected initialized after Save")
	}
}

// TestFileStateStore_Save_RejectsDanglingEdge is the persistence-layer
// backstop for the dangling-edge class of corruption. A per-call-site fix
// in the orchestrator already failed once; enforcing the invariant at Save
// makes the class unreachable regardless of which operation built the state.
// Both endpoints must resolve against the union of specs AND markers, since
// marker→spec edges are legitimate.
func TestFileStateStore_Save_RejectsDanglingEdge(t *testing.T) {
	dir := t.TempDir()
	sess := beginTestSession(t, dir)
	store := statestore.NewFileStateStore(dir)

	// Valid state: marker→spec and spec→spec edges both resolve.
	good := statestore.State{
		Specs:   []core.Spec{{ID: "m.a", Hash: "aaa", Filepath: "a.xml", LineNumber: 1}, {ID: "m.b", Hash: "bbb", Filepath: "b.xml", LineNumber: 1}},
		Markers: []core.Marker{{ID: "mk", Hash: "mmm", Filepath: "a.go", LineNumber: 10, EndLineNumber: 20}},
		Edges:   []core.Edge{{From: "mk", To: "m.a"}, {From: "m.a", To: "m.b"}},
	}
	if err := store.Save(sess, good); err != nil {
		t.Fatalf("expected good state to save, got: %v", err)
	}

	// Dangling To: edge targets a spec not in the state.
	danglingTo := statestore.State{
		Specs: []core.Spec{{ID: "m.a", Hash: "aaa", Filepath: "a.xml", LineNumber: 1}},
		Edges: []core.Edge{{From: "m.a", To: "m.ghost"}},
	}
	err := store.Save(sess, danglingTo)
	if !errors.Is(err, statestore.ErrDanglingEdge) {
		t.Fatalf("expected ErrDanglingEdge for unknown To, got: %v", err)
	}

	// Dangling From: edge originates from a spec not in the state.
	danglingFrom := statestore.State{
		Specs: []core.Spec{{ID: "m.a", Hash: "aaa", Filepath: "a.xml", LineNumber: 1}},
		Edges: []core.Edge{{From: "m.ghost", To: "m.a"}},
	}
	if err := store.Save(sess, danglingFrom); !errors.Is(err, statestore.ErrDanglingEdge) {
		t.Fatalf("expected ErrDanglingEdge for unknown From, got: %v", err)
	}

	// Sanity: the original good state must still round-trip (no partial write
	// left the on-disk file in a bad shape).
	got, err := store.Load(sess)
	if err != nil {
		t.Fatalf("Load after rejected saves: %v", err)
	}
	if len(got.Edges) != 2 {
		t.Fatalf("expected good state to survive rejected saves, got edges: %+v", got.Edges)
	}
}
