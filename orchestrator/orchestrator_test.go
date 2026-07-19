package orchestrator_test

import (
	"errors"
	"testing"

	"drift/core"
	"drift/internal/testutil"
	"drift/orchestrator"
	"drift/scanner"
	"drift/statestore"
)

type fakeStateStore struct {
	state          statestore.State
	loadErr        error
	saveErr        error
	saved          []statestore.State
	initialized    bool
	initializedErr error
}

func (f *fakeStateStore) Load() (statestore.State, error) {
	if f.loadErr != nil {
		return statestore.State{}, f.loadErr
	}
	return f.state, nil
}

func (f *fakeStateStore) Save(state statestore.State) error {
	if f.saveErr != nil {
		return f.saveErr
	}
	f.state = state
	f.saved = append(f.saved, state)
	return nil
}

func (f *fakeStateStore) Initialized() (bool, error) {
	if f.initializedErr != nil {
		return false, f.initializedErr
	}
	return f.initialized, nil
}

func (f *fakeStateStore) Lock() (func(), error) {
	return func() {}, nil
}

type fakeScanner struct {
	result scanner.ScanResult
	err    error
}

func (f *fakeScanner) Scan() (scanner.ScanResult, error) {
	if f.err != nil {
		return scanner.ScanResult{}, f.err
	}
	return f.result, nil
}

func (f *fakeScanner) Dir() string {
	return ""
}

// TestOrchestrator_Todo_NoDrift: scan matches baseline → no closures.
func TestOrchestrator_Todo_NoDrift(t *testing.T) {
	spec := testutil.NewSpec("m.a", "hash-a")
	marker := testutil.NewMarker("cval", "hash-m")
	link := testutil.NewLink("m.a", "cval")
	store := &fakeStateStore{state: statestore.State{
		Specs:   []core.Spec{spec},
		Markers: []core.Marker{marker},
		Edges:   []core.Edge{link},
	}}
	sc := &fakeScanner{result: scanner.ScanResult{
		Specs:   []core.Spec{spec},
		Markers: []core.Marker{marker},
	}}
	orch := orchestrator.NewOrchestrator(store, sc, nil)
	state, err := orch.Todo()
	testutil.AssertNoError(t, err)
	testutil.AssertClosureCount(t, state, 0)
}

// TestOrchestrator_Todo_SpecDrift: spec hash differs → closure includes spec + marker.
func TestOrchestrator_Todo_SpecDrift(t *testing.T) {
	baselineSpec := testutil.NewSpec("m.a", "old")
	scanSpec := testutil.NewSpec("m.a", "new")
	marker := testutil.NewMarker("cval", "hash-m")
	link := testutil.NewLink("m.a", "cval")
	store := &fakeStateStore{state: statestore.State{
		Specs:   []core.Spec{baselineSpec},
		Markers: []core.Marker{marker},
		Edges:   []core.Edge{link},
	}}
	sc := &fakeScanner{result: scanner.ScanResult{
		Specs:   []core.Spec{scanSpec},
		Markers: []core.Marker{marker},
	}}
	orch := orchestrator.NewOrchestrator(store, sc, nil)
	state, err := orch.Todo()
	testutil.AssertNoError(t, err)
	testutil.AssertClosureCount(t, state, 1)
	testutil.AssertNodeInClosure(t, state.Closures[0], "m.a")
	testutil.AssertNodeInClosure(t, state.Closures[0], "cval")
}

// TestOrchestrator_ResetClosure: spec drift → reset → baseline synced.
func TestOrchestrator_ResetClosure(t *testing.T) {
	baselineSpec := testutil.NewSpec("m.a", "old")
	scanSpec := testutil.NewSpec("m.a", "new")
	store := &fakeStateStore{state: statestore.State{
		Specs: []core.Spec{baselineSpec},
	}}
	sc := &fakeScanner{result: scanner.ScanResult{
		Specs: []core.Spec{scanSpec},
	}}
	orch := orchestrator.NewOrchestrator(store, sc, nil)
	state, err := orch.Todo()
	testutil.AssertNoError(t, err)
	testutil.AssertClosureCount(t, state, 1)
	hash := state.Closures[0].Hash

	_, err = orch.ResetClosure(hash)
	testutil.AssertNoError(t, err)

	if len(store.saved) != 1 {
		t.Fatalf("expected 1 save, got %d", len(store.saved))
	}
	saved := store.saved[0]
	if len(saved.Specs) != 1 || saved.Specs[0].Hash != "new" {
		t.Fatalf("baseline not synced: %+v", saved.Specs)
	}
}

// TestOrchestrator_ResetClosure_NotFound: bad hash → ErrResetClosureNotFound.
func TestOrchestrator_ResetClosure_NotFound(t *testing.T) {
	spec := testutil.NewSpec("m.a", "x")
	store := &fakeStateStore{state: statestore.State{Specs: []core.Spec{spec}}}
	sc := &fakeScanner{result: scanner.ScanResult{Specs: []core.Spec{spec}}}
	orch := orchestrator.NewOrchestrator(store, sc, nil)
	_, err := orch.ResetClosure("deadbeef")
	if !errors.Is(err, orchestrator.ErrResetClosureNotFound) {
		t.Fatalf("expected ErrResetClosureNotFound, got %v", err)
	}
}

// TestOrchestrator_Link: link creates an edge.
func TestOrchestrator_Link(t *testing.T) {
	spec := testutil.NewSpec("m.a", "hash-a")
	marker := testutil.NewMarker("cval", "hash-m")
	store := &fakeStateStore{state: statestore.State{
		Specs:   []core.Spec{spec},
		Markers: []core.Marker{marker},
	}}
	sc := &fakeScanner{result: scanner.ScanResult{
		Specs:   []core.Spec{spec},
		Markers: []core.Marker{marker},
	}}
	orch := orchestrator.NewOrchestrator(store, sc, nil)
	if err := orch.Link("cval", "m.a"); err != nil {
		t.Fatalf("Link: %v", err)
	}
	if len(store.state.Edges) != 1 {
		t.Fatalf("expected 1 edge, got %d", len(store.state.Edges))
	}
}

// TestOrchestrator_Unlink: unlink removes the edge.
func TestOrchestrator_Unlink(t *testing.T) {
	spec := testutil.NewSpec("m.a", "hash-a")
	marker := testutil.NewMarker("cval", "hash-m")
	link := testutil.NewLink("m.a", "cval")
	store := &fakeStateStore{state: statestore.State{
		Specs:   []core.Spec{spec},
		Markers: []core.Marker{marker},
		Edges:   []core.Edge{link},
	}}
	sc := &fakeScanner{result: scanner.ScanResult{
		Specs:   []core.Spec{spec},
		Markers: []core.Marker{marker},
	}}
	orch := orchestrator.NewOrchestrator(store, sc, nil)
	if err := orch.Unlink("cval", "m.a"); err != nil {
		t.Fatalf("Unlink: %v", err)
	}
	if len(store.state.Edges) != 0 {
		t.Fatalf("expected 0 edges, got %d", len(store.state.Edges))
	}
}
