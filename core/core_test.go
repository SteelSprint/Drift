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
// set baseline Hash="" so DeriveClosures emits NODE_ADDED.
func TestClosure_OrphanAdded(t *testing.T) {
	specs := []core.Spec{testutil.NewSpec("m.s", "")}
	scan := makeScan(map[string]string{"m.s": hash("new")}, nil, nil)
	closures := core.DeriveClosures(specs, nil, nil, scan)
	testutil.AssertClosureCount(t, core.EvaluatedState{Closures: closures}, 1)
	if len(closures[0].Events) != 1 || closures[0].Events[0].Kind != core.EventNodeAdded {
		t.Fatalf("want 1 NODE_ADDED event (orphan with empty baseline), got %+v", closures[0].Events)
	}
}

// TestClosure_SpecRemoved: spec deleted from scan → NODE_REMOVED event.
func TestClosure_SpecRemoved(t *testing.T) {
	specs := []core.Spec{{ID: "m.s", Hash: "old", Filepath: "m.xml", Deleted: true}}
	scan := makeScan(map[string]string{"m.s": ""}, nil, nil)
	closures := core.DeriveClosures(specs, nil, nil, scan)
	testutil.AssertClosureCount(t, core.EvaluatedState{Closures: closures}, 1)
	if len(closures[0].Events) != 1 || closures[0].Events[0].Kind != core.EventNodeRemoved {
		t.Fatalf("want 1 NODE_REMOVED event, got %+v", closures[0].Events)
	}
	if closures[0].Events[0].NodeID != "m.s" {
		t.Fatalf("event NodeID = %q, want %q", closures[0].Events[0].NodeID, "m.s")
	}
}

// TestClosure_MarkerRemoved: marker deleted from scan → NODE_REMOVED event.
func TestClosure_MarkerRemoved(t *testing.T) {
	markers := []core.Marker{{ID: "cval", Hash: "old", Filepath: "c.go", Deleted: true}}
	scan := makeScan(nil, map[string]string{"cval": ""}, nil)
	closures := core.DeriveClosures(nil, markers, nil, scan)
	testutil.AssertClosureCount(t, core.EvaluatedState{Closures: closures}, 1)
	if len(closures[0].Events) != 1 || closures[0].Events[0].Kind != core.EventNodeRemoved {
		t.Fatalf("want 1 NODE_REMOVED event, got %+v", closures[0].Events)
	}
}

// TestResetClosure_NodeAdded: reset a NODE_ADDED closure → baseline hash established.
func TestResetClosure_NodeAdded(t *testing.T) {
	specs := []core.Spec{testutil.NewSpec("m.s", "")} // empty baseline hash
	scan := makeScan(map[string]string{"m.s": hash("new")}, nil, nil)
	closures := core.DeriveClosures(specs, nil, nil, scan)
	if len(closures) != 1 || closures[0].Events[0].Kind != core.EventNodeAdded {
		t.Fatalf("expected 1 NODE_ADDED closure, got %+v", closures)
	}
	ctx := core.CoreAlgorithmContext{
		Specs:  specs,
		Action: core.ResetClosureAction{Hash: closures[0].Hash, Scan: scan},
	}
	alg := core.NewCoreAlgorithm()
	evaluated, err := alg.EvaluateState(ctx)
	if err != nil {
		t.Fatalf("reset failed: %v", err)
	}
	if len(evaluated.Specs) != 1 || evaluated.Specs[0].Hash != hash("new") {
		t.Fatalf("baseline not established: %+v", evaluated.Specs)
	}
}

// TestResetClosure_NodeRemoved: reset a NODE_REMOVED closure → node gone from baseline.
func TestResetClosure_NodeRemoved(t *testing.T) {
	specs := []core.Spec{{ID: "m.s", Hash: "old", Filepath: "m.xml", Deleted: true}}
	markers := []core.Marker{testutil.NewMarker("cval", hash("m"))}
	baselineEdges := []core.Edge{testutil.NewLink("m.s", "cval")}
	scan := makeScan(map[string]string{"m.s": ""}, map[string]string{"cval": hash("m")}, baselineEdges)
	closures := core.DeriveClosures(specs, markers, baselineEdges, scan)
	if len(closures) != 1 || closures[0].Events[0].Kind != core.EventNodeRemoved {
		t.Fatalf("expected 1 NODE_REMOVED closure, got %+v", closures)
	}
	// Edges touching the removed node should also be filtered.
	ctx := core.CoreAlgorithmContext{
		Specs:   specs,
		Markers: markers,
		Edges:   baselineEdges,
		Action:  core.ResetClosureAction{Hash: closures[0].Hash, Scan: scan},
	}
	alg := core.NewCoreAlgorithm()
	evaluated, err := alg.EvaluateState(ctx)
	if err != nil {
		t.Fatalf("reset failed: %v", err)
	}
	if len(evaluated.Specs) != 0 {
		t.Fatalf("removed spec still in baseline: %+v", evaluated.Specs)
	}
	if len(evaluated.Edges) != 0 {
		t.Fatalf("edges touching removed spec still present: %+v", evaluated.Edges)
	}
	// Invariant: after any state-producing operation, every edge endpoint
	// must resolve to a node in that state. See statestore.validateEdgeEndpoints.
	testutil.AssertEdgesResolve(t, testutil.EvaluatedToState(evaluated))
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

// Guardrail: Broken-edge event persists through reset when mixed with a
// NODE_CHANGED event. The core treats BROKEN_EDGE as a no-op. The
// orchestrator's mergeScannedEdges is responsible for NOT writing broken
// edges to baseline (tested at the orchestrator level).
func TestGuardrail_BrokenEdgePersistsThroughReset(t *testing.T) {
	specs := []core.Spec{
		testutil.NewSpec("m.a", hash("old")),
	}
	baselineEdges := []core.Edge{}
	// Scan: m.a changed AND has a broken edge (m.a → m.missing).
	scanEdges := []core.Edge{testutil.NewRef("m.a", "m.missing")}
	scan := makeScan(map[string]string{"m.a": hash("new")}, nil, scanEdges)

	closures := core.DeriveClosures(specs, nil, baselineEdges, scan)
	if len(closures) != 1 {
		t.Fatalf("expected 1 closure, got %d", len(closures))
	}
	hasNodeChanged := false
	hasBrokenEdge := false
	for _, ev := range closures[0].Events {
		if ev.Kind == core.EventNodeChanged {
			hasNodeChanged = true
		}
		if ev.Kind == core.EventEdgeBroken {
			hasBrokenEdge = true
		}
	}
	if !hasNodeChanged || !hasBrokenEdge {
		t.Fatalf("closure should have both NODE_CHANGED and EDGE_BROKEN events, got: %+v", closures[0].Events)
	}

	// Reset should succeed at the core level (mixed closure, not pure-broken).
	ctx := core.CoreAlgorithmContext{
		Specs:  specs,
		Action: core.ResetClosureAction{Hash: closures[0].Hash, Scan: scan},
	}
	alg := core.NewCoreAlgorithm()
	evaluated, err := alg.EvaluateState(ctx)
	if err != nil {
		t.Fatalf("reset of mixed closure should succeed at core level: %v", err)
	}
	// Broken edges must not appear in the evaluated state's edge list — the
	// orchestrator's mergeScannedEdges is responsible for the same at save.
	testutil.AssertEdgesResolve(t, testutil.EvaluatedToState(evaluated))
	for _, e := range evaluated.Edges {
		if e.To == "m.missing" {
			t.Fatalf("broken edge leaked into evaluated state: %+v", e)
		}
	}
}

// Guardrail: Closure identity is stable when drift-state changes but
// membership does not. Per AGENTS.md: "Identity ... stable across drift-state
// changes; changes only when nodes/edges are added or removed." The hash is
// computed from sorted node IDs + sorted undirected edge keys, NOT from event
// kinds or hashes. So the same seed producing different NewHash values should
// yield the same closure hash.
func TestGuardrail_ClosureIdentityEventAgnostic(t *testing.T) {
	specs := []core.Spec{
		testutil.NewSpec("m.s", hash("old")),
		testutil.NewSpec("m.s2", hash("s2")),
	}
	// S2 cites S. When S drifts, closure = {S, S2} (seed + citer).
	edges := []core.Edge{testutil.NewRef("m.s2", "m.s")}

	// State 1: S changed to "newA".
	scan1 := makeScan(map[string]string{"m.s": hash("newA"), "m.s2": hash("s2")}, nil, edges)
	c1 := core.DeriveClosures(specs, nil, edges, scan1)
	if len(c1) != 1 {
		t.Fatalf("expected 1 closure, got %d", len(c1))
	}

	// State 2: S changed to "newB" (different NewHash, same membership).
	scan2 := makeScan(map[string]string{"m.s": hash("newB"), "m.s2": hash("s2")}, nil, edges)
	c2 := core.DeriveClosures(specs, nil, edges, scan2)
	if len(c2) != 1 {
		t.Fatalf("expected 1 closure, got %d", len(c2))
	}

	// Same membership (m.s + m.s2 + edge) → same hash, even though NewHash differs.
	if c1[0].Hash != c2[0].Hash {
		t.Fatalf("closure hash should be event-agnostic (same membership, different NewHash): %q vs %q", c1[0].Hash, c2[0].Hash)
	}
}

// Guardrail: Closure identity changes when membership changes. Adding a new
// citer to the closure must produce a different hash.
func TestGuardrail_ClosureIdentityChangesWithMembership(t *testing.T) {
	specs := []core.Spec{
		testutil.NewSpec("m.s", hash("old")),
		testutil.NewSpec("m.s2", hash("s2")),
	}
	// Two specs, one edge (S2 cites S).
	edges := []core.Edge{testutil.NewRef("m.s2", "m.s")}
	scan := makeScan(map[string]string{"m.s": hash("new"), "m.s2": hash("s2")}, nil, edges)
	c1 := core.DeriveClosures(specs, nil, edges, scan)
	if len(c1) != 1 {
		t.Fatalf("expected 1 closure, got %d", len(c1))
	}

	// Add a third spec S3 that also cites S. Now the closure expands.
	specs3 := append(specs, testutil.NewSpec("m.s3", hash("s3")))
	edges3 := append(edges, testutil.NewRef("m.s3", "m.s"))
	scan3 := makeScan(map[string]string{"m.s": hash("new"), "m.s2": hash("s2"), "m.s3": hash("s3")}, nil, edges3)
	c2 := core.DeriveClosures(specs3, nil, edges3, scan3)
	if len(c2) != 1 {
		t.Fatalf("expected 1 closure, got %d", len(c2))
	}
	if c1[0].Hash == c2[0].Hash {
		t.Fatalf("closure hash should change when membership changes: both = %q", c1[0].Hash)
	}
}

// TestDeriveOrphans_AllReachable: marker links A, A cites B, B cites C → no orphans.
func TestDeriveOrphans_AllReachable(t *testing.T) {
	specs := []core.Spec{
		testutil.NewSpec("m.a", hash("a")),
		testutil.NewSpec("m.b", hash("b")),
		testutil.NewSpec("m.c", hash("c")),
	}
	baselineEdges := []core.Edge{testutil.NewLink("m.a", "mk")} // marker mk → m.a
	scanEdges := []core.Edge{
		testutil.NewRef("m.a", "m.b"),
		testutil.NewRef("m.b", "m.c"),
	}
	orphans := core.DeriveOrphans(specs, baselineEdges, scanEdges)
	if len(orphans) != 0 {
		t.Fatalf("expected no orphans, got %v", orphans)
	}
}

// TestDeriveOrphans_DirectAndTransitive: a marker-linked spec and its transitive
// citers are covered; an unlinked spec with no citer is an orphan.
func TestDeriveOrphans_DirectAndTransitive(t *testing.T) {
	specs := []core.Spec{
		testutil.NewSpec("m.a", hash("a")),
		testutil.NewSpec("m.b", hash("b")),
		testutil.NewSpec("m.orphan", hash("o")),
	}
	baselineEdges := []core.Edge{testutil.NewLink("m.a", "mk")}
	scanEdges := []core.Edge{testutil.NewRef("m.a", "m.b")}
	orphans := core.DeriveOrphans(specs, baselineEdges, scanEdges)
	if len(orphans) != 1 || orphans[0] != "m.orphan" {
		t.Fatalf("expected [m.orphan], got %v", orphans)
	}
}

// TestDeriveOrphans_NoMarkers: with no marker-link edges, every spec is an orphan.
// This is the empty-marker-set case (a fresh project before any drift link).
func TestDeriveOrphans_NoMarkers(t *testing.T) {
	specs := []core.Spec{
		testutil.NewSpec("m.a", hash("a")),
		testutil.NewSpec("m.b", hash("b")),
	}
	orphans := core.DeriveOrphans(specs, nil, []core.Edge{testutil.NewRef("m.a", "m.b")})
	if len(orphans) != 2 {
		t.Fatalf("expected 2 orphans when no markers exist, got %v", orphans)
	}
}

// TestDeriveOrphans_RefCycle: a cycle reachable from a marker covers all its
// members; a cycle with no marker path is fully orphan. The visited-set must
// prevent infinite recursion.
func TestDeriveOrphans_RefCycle(t *testing.T) {
	specs := []core.Spec{
		testutil.NewSpec("m.a", hash("a")),
		testutil.NewSpec("m.b", hash("b")),
		testutil.NewSpec("m.c", hash("c")),
		testutil.NewSpec("m.d", hash("d")),
	}
	baselineEdges := []core.Edge{testutil.NewLink("m.a", "mk")} // marker → a
	scanEdges := []core.Edge{
		testutil.NewRef("m.a", "m.b"),
		testutil.NewRef("m.b", "m.a"), // cycle a↔b (already reachable)
		testutil.NewRef("m.c", "m.d"), // disconnected cycle c↔d
		testutil.NewRef("m.d", "m.c"),
	}
	orphans := core.DeriveOrphans(specs, baselineEdges, scanEdges)
	if len(orphans) != 2 {
		t.Fatalf("expected 2 orphans [m.c m.d], got %v", orphans)
	}
	if orphans[0] != "m.c" || orphans[1] != "m.d" {
		t.Fatalf("expected [m.c m.d], got %v", orphans)
	}
}

// TestDeriveOrphans_SelfRef: a self-citation does not make a spec reachable.
// Only a marker path does.
func TestDeriveOrphans_SelfRef(t *testing.T) {
	specs := []core.Spec{testutil.NewSpec("m.a", hash("a"))}
	scanEdges := []core.Edge{testutil.NewRef("m.a", "m.a")}
	orphans := core.DeriveOrphans(specs, nil, scanEdges)
	if len(orphans) != 1 || orphans[0] != "m.a" {
		t.Fatalf("self-ref without marker path should be orphan, got %v", orphans)
	}
}

// TestDeriveOrphans_BrokenRefIgnored: a ref to a nonexistent spec is not a
// reachability path (and is already reported separately as EDGE_BROKEN). The
// phantom target is simply absent from the spec universe.
func TestDeriveOrphans_BrokenRefIgnored(t *testing.T) {
	specs := []core.Spec{
		testutil.NewSpec("m.a", hash("a")),
		testutil.NewSpec("m.b", hash("b")),
	}
	baselineEdges := []core.Edge{testutil.NewLink("m.a", "mk")}
	// m.a cites m.phantom (which does not exist as a spec) and m.b (which does).
	scanEdges := []core.Edge{
		testutil.NewRef("m.a", "m.phantom"),
		testutil.NewRef("m.a", "m.b"),
	}
	orphans := core.DeriveOrphans(specs, baselineEdges, scanEdges)
	if len(orphans) != 0 {
		t.Fatalf("broken ref should not affect reachability of real specs, got %v", orphans)
	}
}

// TestDeriveOrphans_BaselineRefEdgesIgnored: reachability uses the LIVE scan
// for ref edges, not stale baseline edges. If m.a→m.b is in baseline but the
// scan no longer contains it, m.b is NOT reachable through the stale edge.
func TestDeriveOrphans_BaselineRefEdgesIgnored(t *testing.T) {
	specs := []core.Spec{
		testutil.NewSpec("m.a", hash("a")),
		testutil.NewSpec("m.b", hash("b")),
	}
	baselineEdges := []core.Edge{
		testutil.NewLink("m.a", "mk"),    // marker → m.a (legitimate seed source)
		testutil.NewRef("m.a", "m.b"),    // stale baseline ref (scan no longer has it)
	}
	// Scan has no ref edges at all: m.b has no live path from the marker.
	orphans := core.DeriveOrphans(specs, baselineEdges, nil)
	if len(orphans) != 1 || orphans[0] != "m.b" {
		t.Fatalf("stale baseline ref must not satisfy reachability, got %v", orphans)
	}
}

// TestDeriveOrphans_DeletedSpecsExcluded: specs flagged Deleted (baseline-only,
// awaiting NODE_REMOVED reset) are not counted as orphans — they're on their
// way out and reporting them as orphan would be noise on top of the removal.
func TestDeriveOrphans_DeletedSpecsExcluded(t *testing.T) {
	specs := []core.Spec{
		testutil.NewSpec("m.a", hash("a")),
		{ID: "m.ghost", Hash: hash("g"), Deleted: true},
	}
	baselineEdges := []core.Edge{testutil.NewLink("m.a", "mk")}
	orphans := core.DeriveOrphans(specs, baselineEdges, nil)
	if len(orphans) != 0 {
		t.Fatalf("deleted spec should be excluded from orphan universe, got %v", orphans)
	}
}

// TestDeriveOrphans_SortedAscending: output is deterministic (sorted) so the
// CLI presenter and JSON are stable across runs.
func TestDeriveOrphans_SortedAscending(t *testing.T) {
	specs := []core.Spec{
		testutil.NewSpec("m.z", hash("z")),
		testutil.NewSpec("m.a", hash("a")),
		testutil.NewSpec("m.m", hash("m")),
	}
	orphans := core.DeriveOrphans(specs, nil, nil)
	want := []string{"m.a", "m.m", "m.z"}
	if len(orphans) != len(want) {
		t.Fatalf("expected %v, got %v", want, orphans)
	}
	for i, id := range want {
		if orphans[i] != id {
			t.Fatalf("orphans not sorted: got %v want %v", orphans, want)
		}
	}
}
