package statestore_test

import (
	"os"
	"path/filepath"
	"testing"

	"drift/core"
	"drift/statestore"
)

func TestFileStateStore_SaveLoad_v4(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".drift"), 0755); err != nil {
		t.Fatal(err)
	}
	store := statestore.NewFileStateStore(dir)

	want := statestore.State{
		Specs:   []core.Spec{{ID: "m.a", Hash: "aaa", Filepath: "a.xml", LineNumber: 1}},
		Markers: []core.Marker{{ID: "cval", Hash: "mmm", Filepath: "a.go", LineNumber: 10, EndLineNumber: 20}},
		Edges:   []core.Edge{{From: "cval", To: "m.a"}},
	}
	if err := store.Save(want); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := store.Load()
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
	if err := os.MkdirAll(filepath.Join(dir, ".drift"), 0755); err != nil {
		t.Fatal(err)
	}
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
	_, err := store.Load()
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
	if err := os.MkdirAll(filepath.Join(dir, ".drift"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := store.Save(statestore.State{}); err != nil {
		t.Fatal(err)
	}
	if ok, _ := store.Initialized(); !ok {
		t.Fatal("expected initialized after Save")
	}
}
