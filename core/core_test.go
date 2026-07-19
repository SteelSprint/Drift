package core_test

import (
	"errors"
	"testing"

	"drift/core"
	"drift/internal/testutil"
)

// hash returns a deterministic placeholder hash derived from a string.
func hash(s string) string { return s + "-hash"
}

// makeScan builds a core.Scan from given spec/marker hashes plus scan edges.
func makeScan(specHashes, markerHashes map[string]string, scanEdges []core.Edge) core.Scan {
	return core.Scan{SpecHashes: specHashes, MarkerHashes: markerHashes, Edges: scanEdges}
}

// TestClosure_SingletonSpec: spec S edited, isolated (no edges).
// Truth table row 1. Closure = {S}, 1 node, 0 edges, 1 NODE_CHANGED event.
func TestClosure_SingletonSpec(t *testing.T) {
	specs := []core.Spec{testutil.NewSpec("m.s", hash("old"))}
	markers := []core.Marker{}
	baselineEdges := []core.Edge{}
	scan := makeScan(map[string]string{"m.s": hash("new")}, nil, nil)
	closures := core.DeriveClosures(specs, markers, baselineEdges, scan)
	testutil.AssertClosureCount(t, core.EvaluatedState{Closures: closures}, 1)
	c := closures[0]
	testutil.AssertNodeInClosure(t, c, "m.s")
	if len(c.Events) != 1 || c.Events[0].Kind != core.EventNodeChanged {
		t.Fatalf("want 1 NODE_CHANGED event, got %+v", c.Events)
	}
	if c.Events[0].NodeID != "m.s" {
		t.Fatalf("event NodeID = %q, want %q", c.Events[0].NodeID, "m.s")
	}
}

// TestClosure_CiterDirection: spec S drifts; S has a ref to S'.
// S' should NOT be in closure (S' is cited by S, not the citer direction).
func TestClosure_CiterDirection(t *testing.T) {
	specs := []core.Spec{
		testutil.NewSpec("m.s", hash("old")),
		testutil.NewSpec("m.s2", hash("x")),
	}
	baselineEdges := []core.Edge{testutil.NewRef("m.s", "m.s2")}
	scan := makeScan(
		map[string]string{"m.s": hash("new"), "m.s2": hash("x")},
		nil,
		baselineEdges,
	)
	closures := core.DeriveClosures(specs, nil, baselineEdges, scan)
	testutil.AssertClosureCount(t, core.EvaluatedState{Closures: closures}, 1)
	testutil.AssertNodeInClosure(t, closures[0], "m.s")
	testutil.AssertNodeNotInClosure(t, closures[0], "m.s2")
}

// TestClosure_MarkerAsCiter: spec S drifts, marker M links to S.
// Closure = {S, M}. M cites S.
func TestClosure_MarkerAsCiter(t *testing.T) {
	specs := []core.Spec{testutil.NewSpec("m.s", hash("old"))}
	markers := []core.Marker{testutil.NewMarker("cval", hash("m"))}
	baselineEdges := []core.Edge{testutil.NewLink("m.s", "cval")}
	scan := makeScan(
		map[string]string{"m.s": hash("new")},
		map[string]string{"cval": hash("m")},
		baselineEdges,
	)
	closures := core.DeriveClosures(specs, markers, baselineEdges, scan)
	testutil.AssertClosureCount(t, core.EvaluatedState{Closures: closures}, 1)
	testutil.AssertNodeInClosure(t, closures[0], "m.s")
	testutil.AssertNodeInClosure(t, closures[0], "cval")
}

// TestClosure_MultiLinkMarkerDrift: marker M drifts, linked to S1 and S2.
// Closure = {M, S1, S2}. Both edges drifted (M's hash changed).
func TestClosure_MultiLinkMarkerDrift(t *testing.T) {
	specs := []core.Spec{
		testutil.NewSpec("m.s1", hash("s1")),
		testutil.NewSpec("m.s2", hash("s2")),
	}
	markers := []core.Marker{testutil.NewMarker("cval", hash("old"))}
	baselineEdges := []core.Edge{
		testutil.NewLink("m.s1", "cval"),
		testutil.NewLink("m.s2", "cval"),
	}
	scan := makeScan(
		map[string]string{"m.s1": hash("s1"), "m.s2": hash("s2")},
		map[string]string{"cval": hash("new")},
		baselineEdges,
	)
	closures := core.DeriveClosures(specs, markers, baselineEdges, scan)
	testutil.AssertClosureCount(t, core.EvaluatedState{Closures: closures}, 1)
	testutil.AssertNodeInClosure(t, closures[0], "cval")
	testutil.AssertNodeInClosure(t, closures[0], "m.s1")
	testutil.AssertNodeInClosure(t, closures[0], "m.s2")
}

// TestClosure_MultiLinkSpecDrift: marker M linked to S1 and S2; S1 drifts.
// Closure = {S1, M}. S2 NOT in closure (S2 is cited by M, not citer direction).
func TestClosure_MultiLinkSpecDrift(t *testing.T) {
	specs := []core.Spec{
		testutil.NewSpec("m.s1", hash("old")),
		testutil.NewSpec("m.s2", hash("s2")),
	}
	markers := []core.Marker{testutil.NewMarker("cval", hash("m"))}
	baselineEdges := []core.Edge{
		testutil.NewLink("m.s1", "cval"),
		testutil.NewLink("m.s2", "cval"),
	}
	scan := makeScan(
		map[string]string{"m.s1": hash("new"), "m.s2": hash("s2")},
		map[string]string{"cval": hash("m")},
		baselineEdges,
	)
	closures := core.DeriveClosures(specs, markers, baselineEdges, scan)
	testutil.AssertClosureCount(t, core.EvaluatedState{Closures: closures}, 1)
	testutil.AssertNodeInClosure(t, closures[0], "m.s1")
	testutil.AssertNodeInClosure(t, closures[0], "cval")
	testutil.AssertNodeNotInClosure(t, closures[0], "m.s2")
}

// TestClosure_StrictDisjoint: S1 and S2 both cited by S3. S1 and S2 both drift.
// Two closures: closure_S1 = {S1, S3, ...}, closure_S2 = {S2, S3, ...}. S3 in both.
func TestClosure_StrictDisjoint(t *testing.T) {
	specs := []core.Spec{
		testutil.NewSpec("m.s1", hash("old1")),
		testutil.NewSpec("m.s2", hash("old2")),
		testutil.NewSpec("m.s3", hash("s3")),
	}
	baselineEdges := []core.Edge{
		testutil.NewRef("m.s3", "m.s1"), // S3 cites S1
		testutil.NewRef("m.s3", "m.s2"), // S3 cites S2
	}
	scan := makeScan(
		map[string]string{
			"m.s1": hash("new1"),
			"m.s2": hash("new2"),
			"m.s3": hash("s3"),
		},
		nil,
		baselineEdges,
	)
	closures := core.DeriveClosures(specs, nil, baselineEdges, scan)
	testutil.AssertClosureCount(t, core.EvaluatedState{Closures: closures}, 2)
	s1Closure := testutil.FindClosureContainingNode(t, core.EvaluatedState{Closures: closures}, "m.s1")
	testutil.AssertNodeInClosure(t, s1Closure, "m.s3")
	s2Closure := testutil.FindClosureContainingNode(t, core.EvaluatedState{Closures: closures}, "m.s2")
	testutil.AssertNodeInClosure(t, s2Closure, "m.s3")
	// Different hashes for the two closures.
	if s1Closure.Hash == s2Closure.Hash {
		t.Fatalf("strict disjoint closures should have different hashes, both = %q", s1Closure.Hash)
	}
}

// TestClosure_NewEdge: scan has a new edge (A → B) not in baseline.
// Closure seeded by A (citer). Closure = {A + citers of A}.
func TestClosure_NewEdge(t *testing.T) {
	specs := []core.Spec{
		testutil.NewSpec("m.a", hash("a")),
		testutil.NewSpec("m.b", hash("b")),
	}
	baselineEdges := []core.Edge{}
	scanEdges := []core.Edge{testutil.NewRef("m.a", "m.b")}
	scan := makeScan(
		map[string]string{"m.a": hash("a"), "m.b": hash("b")},
		nil,
		scanEdges,
	)
	closures := core.DeriveClosures(specs, nil, baselineEdges, scan)
	testutil.AssertClosureCount(t, core.EvaluatedState{Closures: closures}, 1)
	testutil.AssertNodeInClosure(t, closures[0], "m.a")
	// B is in the closure (because A→B edge exists among closure nodes? No —
	// closure nodes are determined by citer walk from A. B doesn't cite A,
	// so B is not reached. But edge A→B is among closure nodes only if both
	// A and B are in closure. Since B is not in closure, edge A→B is not a
	// closure edge. The event still records A→B.)
	testutil.AssertNodeNotInClosure(t, closures[0], "m.b")
	if len(closures[0].Events) != 1 || closures[0].Events[0].Kind != core.EventEdgeAdded {
		t.Fatalf("want 1 EDGE_ADDED event, got %+v", closures[0].Events)
	}
}

// TestClosure_RemovedEdge: baseline has edge (A → B), scan doesn't.
// Closure seeded by A. Closure = {A + citers of A}.
func TestClosure_RemovedEdge(t *testing.T) {
	specs := []core.Spec{
		testutil.NewSpec("m.a", hash("a")),
		testutil.NewSpec("m.b", hash("b")),
	}
	baselineEdges := []core.Edge{testutil.NewRef("m.a", "m.b")}
	scan := makeScan(
		map[string]string{"m.a": hash("a"), "m.b": hash("b")},
		nil,
		nil,
	)
	closures := core.DeriveClosures(specs, nil, baselineEdges, scan)
	testutil.AssertClosureCount(t, core.EvaluatedState{Closures: closures}, 1)
	if len(closures[0].Events) != 1 || closures[0].Events[0].Kind != core.EventEdgeRemoved {
		t.Fatalf("want 1 EDGE_REMOVED event, got %+v", closures[0].Events)
	}
}

// TestClosure_BrokenEdge: scan has edge (A → B) where B doesn't exist.
// Closure seeded by A. Closure = {A + citers of A}. Event EDGE_BROKEN.
func TestClosure_BrokenEdge(t *testing.T) {
	specs := []core.Spec{testutil.NewSpec("m.a", hash("a"))}
	baselineEdges := []core.Edge{}
	scanEdges := []core.Edge{testutil.NewRef("m.a", "m.b")}
	scan := makeScan(
		map[string]string{"m.a": hash("a")},
		nil,
		scanEdges,
	)
	closures := core.DeriveClosures(specs, nil, baselineEdges, scan)
	testutil.AssertClosureCount(t, core.EvaluatedState{Closures: closures}, 1)
	if len(closures[0].Events) != 1 || closures[0].Events[0].Kind != core.EventEdgeBroken {
		t.Fatalf("want 1 EDGE_BROKEN event, got %+v", closures[0].Events)
	}
}

// TestClosure_OrphanAdded: new spec in scan, no edges. The reconciler would
// set baseline Hash="" so DeriveClosures sees a NODE_CHANGED event (baseline
// empty vs scan hash).
func TestClosure_OrphanAdded(t *testing.T) {
	specs := []core.Spec{testutil.NewSpec("m.s", "")}
	scan := makeScan(map[string]string{"m.s": hash("new")}, nil, nil)
	closures := core.DeriveClosures(specs, nil, nil, scan)
	testutil.AssertClosureCount(t, core.EvaluatedState{Closures: closures}, 1)
	if len(closures[0].Events) != 1 || closures[0].Events[0].Kind != core.EventNodeChanged {
		t.Fatalf("want 1 NODE_CHANGED event (orphan with empty baseline), got %+v", closures[0].Events)
	}
}

// TestClosure_NoDrift: scan matches baseline. No closures.
func TestClosure_NoDrift(t *testing.T) {
	specs := []core.Spec{testutil.NewSpec("m.s", hash("x"))}
	markers := []core.Marker{testutil.NewMarker("cval", hash("y"))}
	baselineEdges := []core.Edge{testutil.NewLink("m.s", "cval")}
	scan := makeScan(
		map[string]string{"m.s": hash("x")},
		map[string]string{"cval": hash("y")},
		baselineEdges,
	)
	closures := core.DeriveClosures(specs, markers, baselineEdges, scan)
	testutil.AssertClosureCount(t, core.EvaluatedState{Closures: closures}, 0)
}

// TestClosure_HashStability: same setup, drift, derive, derive again → same hash.
func TestClosure_HashStability(t *testing.T) {
	specs := []core.Spec{
		testutil.NewSpec("m.s", hash("old")),
		testutil.NewSpec("m.s2", hash("s2")),
	}
	baselineEdges := []core.Edge{testutil.NewRef("m.s2", "m.s")} // S2 cites S
	scan := makeScan(
		map[string]string{"m.s": hash("new"), "m.s2": hash("s2")},
		nil,
		baselineEdges,
	)
	c1 := core.DeriveClosures(specs, nil, baselineEdges, scan)
	c2 := core.DeriveClosures(specs, nil, baselineEdges, scan)
	if c1[0].Hash != c2[0].Hash {
		t.Fatalf("hash not stable across runs: %q vs %q", c1[0].Hash, c2[0].Hash)
	}
}

// TestClosure_CycleRejected: validation rejects directed spec-spec cycle.
func TestClosure_CycleRejected(t *testing.T) {
	ctx := core.CoreAlgorithmContext{
		Specs: []core.Spec{
			testutil.NewSpec("m.a", hash("a")),
			testutil.NewSpec("m.b", hash("b")),
		},
		Edges: []core.Edge{
			testutil.NewRef("m.a", "m.b"),
			testutil.NewRef("m.b", "m.a"),
		},
		Action: core.TodoAction{Scan: makeScan(map[string]string{"m.a": hash("a"), "m.b": hash("b")}, nil, nil)},
	}
	alg := core.NewCoreAlgorithm()
	if _, err := alg.EvaluateState(ctx); err == nil {
		t.Fatalf("expected cycle error, got nil")
	}
}

// TestResetClosure_NodeChanged: reset a NODE_CHANGED closure → baseline hash syncs.
func TestResetClosure_NodeChanged(t *testing.T) {
	specs := []core.Spec{testutil.NewSpec("m.s", hash("old"))}
	baselineEdges := []core.Edge{}
	scan := makeScan(map[string]string{"m.s": hash("new")}, nil, nil)
	closures := core.DeriveClosures(specs, nil, baselineEdges, scan)
	if len(closures) != 1 {
		t.Fatalf("expected 1 closure, got %d", len(closures))
	}
	ctx := core.CoreAlgorithmContext{
		Specs:   specs,
		Edges:   baselineEdges,
		Action:  core.ResetClosureAction{Hash: closures[0].Hash, Scan: scan},
	}
	alg := core.NewCoreAlgorithm()
	evaluated, err := alg.EvaluateState(ctx)
	if err != nil {
		t.Fatalf("reset failed: %v", err)
	}
	if len(evaluated.Specs) != 1 || evaluated.Specs[0].Hash != hash("new") {
		t.Fatalf("baseline not synced: %+v", evaluated.Specs)
	}
}

// TestResetClosure_NotFound: bad hash → ErrClosureNotFound.
func TestResetClosure_NotFound(t *testing.T) {
	specs := []core.Spec{testutil.NewSpec("m.s", hash("old"))}
	scan := makeScan(map[string]string{"m.s": hash("new")}, nil, nil)
	ctx := core.CoreAlgorithmContext{
		Specs:  specs,
		Action: core.ResetClosureAction{Hash: "deadbeef", Scan: scan},
	}
	alg := core.NewCoreAlgorithm()
	_, err := alg.EvaluateState(ctx)
	if !errors.Is(err, core.ErrClosureNotFound) {
		t.Fatalf("expected ErrClosureNotFound, got %v", err)
	}
}

// TestResetClosure_BrokenEdgeOnlyRefused: closure with only broken edges → error.
func TestResetClosure_BrokenEdgeOnlyRefused(t *testing.T) {
	specs := []core.Spec{testutil.NewSpec("m.a", hash("a"))}
	scanEdges := []core.Edge{testutil.NewRef("m.a", "m.b")}
	scan := makeScan(map[string]string{"m.a": hash("a")}, nil, scanEdges)
	closures := core.DeriveClosures(specs, nil, nil, scan)
	if len(closures) != 1 {
		t.Fatalf("expected 1 closure, got %d", len(closures))
	}
	ctx := core.CoreAlgorithmContext{
		Specs:  specs,
		Action: core.ResetClosureAction{Hash: closures[0].Hash, Scan: scan},
	}
	alg := core.NewCoreAlgorithm()
	_, err := alg.EvaluateState(ctx)
	if !errors.Is(err, core.ErrBrokenEdgeNotResettable) {
		t.Fatalf("expected ErrBrokenEdgeNotResettable, got %v", err)
	}
}
