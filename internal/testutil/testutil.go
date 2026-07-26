package testutil

import (
	"crypto/sha1"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"drift/core"
	"drift/statestore"
)

// D! id=tcon range-start

func NewSpec(id string, hash string) core.Spec {
	return core.Spec{ID: id, Hash: hash, Filepath: id + ".xml", LineNumber: 10}
}

func NewSpecWithLocation(id string, hash string, filepath string, lineNumber int) core.Spec {
	return core.Spec{ID: id, Hash: hash, Filepath: filepath, LineNumber: lineNumber}
}

func NewMarker(id string, hash string) core.Marker {
	return core.Marker{ID: id, Hash: hash, Filepath: id + ".go", LineNumber: 20, EndLineNumber: 30}
}

func NewMarkerWithLocation(id string, hash string, filepath string, lineNumber int) core.Marker {
	return core.Marker{ID: id, Hash: hash, Filepath: filepath, LineNumber: lineNumber, EndLineNumber: lineNumber + 10}
}

// NewLink constructs a link-style Edge: marker stores edge to spec.
// Argument order preserved from the pre-collapse API for test readability.
func NewLink(specID string, markerID string) core.Edge {
	return core.Edge{From: markerID, To: specID}
}

// NewRef constructs a spec-spec Edge: fromSpec stores edge to toSpec.
func NewRef(fromSpec, toSpec string) core.Edge {
	return core.Edge{From: fromSpec, To: toSpec}
}

// D! id=tcon range-end

// D! id=tassert range-start

func AssertNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func AssertErrorWraps(t *testing.T, err error, target error) {
	t.Helper()
	if !errors.Is(err, target) {
		t.Fatalf("error %v does not wrap %v", err, target)
	}
}

func AssertClosureCount(t *testing.T, state core.EvaluatedState, want int) {
	t.Helper()
	if len(state.Closures) != want {
		t.Fatalf("closure count = %d, want %d", len(state.Closures), want)
	}
}

func AssertBaselineHashes(t *testing.T, state core.EvaluatedState, specID string, wantSpecHash string, markerID string, wantMarkerHash string) {
	t.Helper()
	if specID != "" {
		found := false
		for _, spec := range state.Specs {
			if spec.ID == specID {
				if spec.Hash != wantSpecHash {
					t.Fatalf("spec %q baseline hash = %q, want %q", specID, spec.Hash, wantSpecHash)
				}
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("spec %q not found in evaluated state", specID)
		}
	}
	if markerID != "" {
		found := false
		for _, marker := range state.Markers {
			if marker.ID == markerID {
				if marker.Hash != wantMarkerHash {
					t.Fatalf("marker %q baseline hash = %q, want %q", markerID, marker.Hash, wantMarkerHash)
				}
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("marker %q not found in evaluated state", markerID)
		}
	}
}

func AssertStateEquals(t *testing.T, got, want statestore.State) {
	t.Helper()
	// Normalize slice order per R5: sort by ID for specs/markers, by (From,To) for edges.
	gotSpecs := make([]core.Spec, len(got.Specs))
	copy(gotSpecs, got.Specs)
	sort.Slice(gotSpecs, func(i, j int) bool { return gotSpecs[i].ID < gotSpecs[j].ID })

	wantSpecs := make([]core.Spec, len(want.Specs))
	copy(wantSpecs, want.Specs)
	sort.Slice(wantSpecs, func(i, j int) bool { return wantSpecs[i].ID < wantSpecs[j].ID })

	gotMarkers := make([]core.Marker, len(got.Markers))
	copy(gotMarkers, got.Markers)
	sort.Slice(gotMarkers, func(i, j int) bool { return gotMarkers[i].ID < gotMarkers[j].ID })

	wantMarkers := make([]core.Marker, len(want.Markers))
	copy(wantMarkers, want.Markers)
	sort.Slice(wantMarkers, func(i, j int) bool { return wantMarkers[i].ID < wantMarkers[j].ID })

	gotEdges := make([]core.Edge, len(got.Edges))
	copy(gotEdges, got.Edges)
	sort.Slice(gotEdges, func(i, j int) bool {
		if gotEdges[i].From != gotEdges[j].From {
			return gotEdges[i].From < gotEdges[j].From
		}
		return gotEdges[i].To < gotEdges[j].To
	})

	wantEdges := make([]core.Edge, len(want.Edges))
	copy(wantEdges, want.Edges)
	sort.Slice(wantEdges, func(i, j int) bool {
		if wantEdges[i].From != wantEdges[j].From {
			return wantEdges[i].From < wantEdges[j].From
		}
		return wantEdges[i].To < wantEdges[j].To
	})

	if len(gotSpecs) != len(wantSpecs) {
		t.Fatalf("specs length = %d, want %d (got=%v want=%v)", len(gotSpecs), len(wantSpecs), gotSpecs, wantSpecs)
	}
	for i := range gotSpecs {
		if gotSpecs[i] != wantSpecs[i] {
			t.Fatalf("spec[%d] = %+v, want %+v", i, gotSpecs[i], wantSpecs[i])
		}
	}
	if len(gotMarkers) != len(wantMarkers) {
		t.Fatalf("markers length = %d, want %d (got=%v want=%v)", len(gotMarkers), len(wantMarkers), gotMarkers, wantMarkers)
	}
	for i := range gotMarkers {
		if gotMarkers[i] != wantMarkers[i] {
			t.Fatalf("marker[%d] = %+v, want %+v", i, gotMarkers[i], wantMarkers[i])
		}
	}
	if len(gotEdges) != len(wantEdges) {
		t.Fatalf("edges length = %d, want %d (got=%v want=%v)", len(gotEdges), len(wantEdges), gotEdges, wantEdges)
	}
	for i := range gotEdges {
		if gotEdges[i] != wantEdges[i] {
			t.Fatalf("edge[%d] = %+v, want %+v", i, gotEdges[i], wantEdges[i])
		}
	}
}

// D! id=tassert range-end

// D! id=tsha range-start

func ExpectedSha1Hex(content string) string {
	h := sha1.Sum([]byte(content))
	return fmt.Sprintf("%x", h)
}

// D! id=tsha range-end

// D! id=tfind range-start

func FindScanResultSpec(results []core.Spec, id string) (core.Spec, bool) {
	for _, s := range results {
		if s.ID == id {
			return s, true
		}
	}
	return core.Spec{}, false
}

func FindScanResultMarker(results []core.Marker, id string) (core.Marker, bool) {
	for _, m := range results {
		if m.ID == id {
			return m, true
		}
	}
	return core.Marker{}, false
}

// D! id=tfind range-end

// D! id=tfix range-start

func WriteSpecFile(t *testing.T, dir, name, content string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write spec file %s: %v", name, err)
	}
}

func WriteCodeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write code file %s: %v", name, err)
	}
}

func WriteIgnoreFile(t *testing.T, dir, content string) {
	t.Helper()
	path := filepath.Join(dir, "drift.ignore")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write drift.ignore: %v", err)
	}
}

// D! id=tfix range-end

// D! id=tfind2 range-start

func FindSpecInEvaluatedState(t *testing.T, state core.EvaluatedState, id string) core.Spec {
	t.Helper()
	for _, s := range state.Specs {
		if s.ID == id {
			return s
		}
	}
	t.Fatalf("spec %q not found in evaluated state", id)
	return core.Spec{}
}

func FindMarkerInEvaluatedState(t *testing.T, state core.EvaluatedState, id string) core.Marker {
	t.Helper()
	for _, m := range state.Markers {
		if m.ID == id {
			return m
		}
	}
	t.Fatalf("marker %q not found in evaluated state", id)
	return core.Marker{}
}

func EvaluatedToState(state core.EvaluatedState) statestore.State {
	return statestore.State{
		Specs:   state.Specs,
		Markers: state.Markers,
		Edges:   state.Edges,
	}
}

// FindClosureByHash returns the closure with the given hash, or fails the test.
func FindClosureByHash(t *testing.T, state core.EvaluatedState, hash string) core.Closure {
	t.Helper()
	for _, c := range state.Closures {
		if c.Hash == hash {
			return c
		}
	}
	t.Fatalf("closure with hash %q not found", hash)
	return core.Closure{}
}

// FindClosureContainingNode returns the first closure containing the given
// node ID, or fails the test.
func FindClosureContainingNode(t *testing.T, state core.EvaluatedState, nodeID string) core.Closure {
	t.Helper()
	for _, c := range state.Closures {
		for _, n := range c.Nodes {
			if n.ID == nodeID {
				return c
			}
		}
	}
	t.Fatalf("closure containing node %q not found", nodeID)
	return core.Closure{}
}

// D! id=tfind2 range-end

// D! id=tassert2 range-start

// AssertNodeInClosure asserts the closure has a node with the given ID.
func AssertNodeInClosure(t *testing.T, c core.Closure, nodeID string) {
	t.Helper()
	for _, n := range c.Nodes {
		if n.ID == nodeID {
			return
		}
	}
	t.Fatalf("node %q not in closure %q (nodes: %v)", nodeID, c.Hash, c.Nodes)
}

// AssertNodeNotInClosure asserts the closure does NOT have a node with the given ID.
func AssertNodeNotInClosure(t *testing.T, c core.Closure, nodeID string) {
	t.Helper()
	for _, n := range c.Nodes {
		if n.ID == nodeID {
			t.Fatalf("node %q unexpectedly in closure %q", nodeID, c.Hash)
		}
	}
}

// D! id=tassert2 range-end
