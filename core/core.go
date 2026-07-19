package core

import (
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"
)

type CoreAlgorithm struct{}

func NewCoreAlgorithm() *CoreAlgorithm {
	return &CoreAlgorithm{}
}

type Spec struct {
	Filepath   string
	LineNumber int
	ID         string
	Module     string
	Hash       string
	Deleted    bool
}

type Marker struct {
	Filepath      string
	LineNumber    int
	EndLineNumber int
	ID            string
	Hash          string
	Deleted       bool
}

// Edge is the unified directed relationship between two nodes (specs and/or
// markers). From declares/stores the edge (citer); To is the target (cited).
// Two establishment kinds encoded by endpoint types: marker → spec (drift link)
// and spec → spec (auto-parsed <ref>). Drift propagation follows the citer
// direction (cited → citer), transitive to fixpoint. Markers cannot be cited,
// so drift through a marker stops there.
type Edge struct {
	From string
	To   string
}

type Action interface {
	isAction()
}

type Scan struct {
	SpecHashes   map[string]string
	MarkerHashes map[string]string
	Edges        []Edge
}

type TodoAction struct {
	Scan Scan
}

func (TodoAction) isAction() {}

// ResetClosureAction targets a closure by its 8-character hash. The closure
// is re-derived from scan+baseline, then its seed events are synced into
// baseline (baseline hash → scan hash, edge added/removed, etc.). Broken-edge
// events are no-ops on reset and persist until the user fixes the scan.
type ResetClosureAction struct {
	Hash string
	Scan Scan
}

func (ResetClosureAction) isAction() {}

type CoreAlgorithmContext struct {
	Specs   []Spec
	Markers []Marker
	Edges   []Edge
	Action  Action
}

// EventKind enumerates drift event categories.
type EventKind int

const (
	EventNodeChanged EventKind = iota
	EventNodeAdded
	EventNodeRemoved
	EventEdgeAdded
	EventEdgeRemoved
	EventEdgeBroken
)

// DriftEvent is one drift event associated with a seed node. Edge events
// carry the edge; node events carry the node ID and (for NODE_CHANGED) the
// old/new hashes.
type DriftEvent struct {
	Kind    EventKind
	NodeID  string // for node events
	Edge    *Edge  // for edge events
	OldHash string // for NODE_CHANGED
	NewHash string // for NODE_CHANGED
	Seed    string // ID of the seed node that originated this event
}

// NodeRef is a lightweight reference to a node with location info, for
// closure display. IsSpec is true for specs, false for markers.
type NodeRef struct {
	ID         string
	IsSpec     bool
	Filepath   string
	LineNumber int
}

// Closure is one drift-impacted connected subgraph derived per-seed. Hash
// is the first 8 hex chars of SHA1(sorted node IDs + sorted undirected edge
// keys). Closures with identical hashes are merged with combined events.
type Closure struct {
	Hash   string
	Nodes  []NodeRef    // sorted by ID
	Edges  []Edge       // sorted by canonicalized undirected key
	Events []DriftEvent // events with seed in this closure
	Seeds  []string     // distinct seed node IDs (derived from Events[].Seed)
}

type EvaluatedState struct {
	Specs    []Spec
	Markers  []Marker
	Edges    []Edge
	Closures []Closure
}

var (
	ErrDuplicateSpecID        = errors.New("duplicate spec id")
	ErrDuplicateMarkerID      = errors.New("duplicate marker id")
	ErrEdgeUnknownFrom        = errors.New("edge references unknown from-node")
	ErrEdgeUnknownTo          = errors.New("edge references unknown to-node")
	ErrDuplicateEdge          = errors.New("duplicate edge")
	ErrEdgeSelfReference      = errors.New("edge references its own source")
	ErrEdgeCycle              = errors.New("edge graph contains a directed cycle")
	ErrScanMissingSpecHash    = errors.New("scan missing spec hash")
	ErrScanMissingMarkerHash  = errors.New("scan missing marker hash")
	ErrScanUnknownSpecHash    = errors.New("scan contains unknown spec hash")
	ErrScanUnknownMarkerHash  = errors.New("scan contains unknown marker hash")
	ErrUnknownAction          = errors.New("unknown action")
	ErrClosureNotFound        = errors.New("closure hash not found in derived closures")
	ErrBrokenEdgeNotResettable = errors.New("closure contains only broken-edge events, which require scan fix")
)

// D! id=cval range-start
func (ctx CoreAlgorithmContext) Validate() error {
	seenSpecIDs := make(map[string]bool, len(ctx.Specs))
	for _, spec := range ctx.Specs {
		if seenSpecIDs[spec.ID] {
			return fmt.Errorf("%w: %q", ErrDuplicateSpecID, spec.ID)
		}
		seenSpecIDs[spec.ID] = true
	}
	seenMarkerIDs := make(map[string]bool, len(ctx.Markers))
	for _, marker := range ctx.Markers {
		if seenMarkerIDs[marker.ID] {
			return fmt.Errorf("%w: %q", ErrDuplicateMarkerID, marker.ID)
		}
		seenMarkerIDs[marker.ID] = true
	}
	knownNode := func(id string) bool {
		return seenSpecIDs[id] || seenMarkerIDs[id]
	}
	seenEdgeKeys := make(map[string]bool, len(ctx.Edges))
	for _, edge := range ctx.Edges {
		if !knownNode(edge.From) {
			return fmt.Errorf("%w: %q", ErrEdgeUnknownFrom, edge.From)
		}
		if !knownNode(edge.To) {
			return fmt.Errorf("%w: %q", ErrEdgeUnknownTo, edge.To)
		}
		if edge.From == edge.To {
			return fmt.Errorf("%w: %q", ErrEdgeSelfReference, edge.From)
		}
		key := edge.From + "\x00" + edge.To
		if seenEdgeKeys[key] {
			return fmt.Errorf("%w: from=%q to=%q", ErrDuplicateEdge, edge.From, edge.To)
		}
		seenEdgeKeys[key] = true
	}
	if err := detectEdgeCycle(ctx.Edges); err != nil {
		return err
	}
	return nil
}

// D! id=cval range-end

// D! id=crfv range-start
func detectEdgeCycle(edges []Edge) error {
	cycles := findAllEdgeCycles(edges)
	if len(cycles) == 0 {
		return nil
	}
	var parts []string
	for i, c := range cycles {
		if i > 0 {
			parts = append(parts, "; ")
		}
		parts = append(parts, joinPath(c))
	}
	return fmt.Errorf("%w: %s", ErrEdgeCycle, strings.Join(parts, ""))
}

func findAllEdgeCycles(edges []Edge) [][]string {
	adj := make(map[string][]string)
	for _, e := range edges {
		if !isSpecID(e.From) || !isSpecID(e.To) {
			continue
		}
		adj[e.From] = append(adj[e.From], e.To)
	}
	var cycles [][]string
	const (
		white = 0
		gray  = 1
		black = 2
	)
	color := make(map[string]int)
	var stack []string
	var visit func(node string)
	visit = func(node string) {
		color[node] = gray
		stack = append(stack, node)
		nexts := append([]string(nil), adj[node]...)
		sort.Strings(nexts)
		for _, next := range nexts {
			switch color[next] {
			case gray:
				start := 0
				for i, n := range stack {
					if n == next {
						start = i
						break
					}
				}
				path := append(append([]string{}, stack[start:]...), next)
				cycles = append(cycles, path)
			case white:
				visit(next)
			}
		}
		stack = stack[:len(stack)-1]
		color[node] = black
	}
	var nodes []string
	for n := range adj {
		nodes = append(nodes, n)
	}
	sort.Strings(nodes)
	for _, n := range nodes {
		if color[n] == white {
			visit(n)
		}
	}
	return cycles
}

func joinPath(nodes []string) string {
	out := ""
	for i, n := range nodes {
		if i > 0 {
			out += " → "
		}
		out += n
	}
	return out
}

func isSpecID(id string) bool {
	first := strings.Index(id, ".")
	if first < 0 {
		return false
	}
	return strings.Index(id[first+1:], ".") < 0
}

// D! id=crfv range-end

// D! id=cscn range-start
func validateScanCoversAllNodes(scan Scan, specs []Spec, markers []Marker) error {
	for _, spec := range specs {
		if _, ok := scan.SpecHashes[spec.ID]; !ok {
			return fmt.Errorf("%w: %q", ErrScanMissingSpecHash, spec.ID)
		}
	}
	for _, marker := range markers {
		if _, ok := scan.MarkerHashes[marker.ID]; !ok {
			return fmt.Errorf("%w: %q", ErrScanMissingMarkerHash, marker.ID)
		}
	}
	for specID := range scan.SpecHashes {
		found := false
		for _, spec := range specs {
			if spec.ID == specID {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("%w: %q", ErrScanUnknownSpecHash, specID)
		}
	}
	for markerID := range scan.MarkerHashes {
		found := false
		for _, marker := range markers {
			if marker.ID == markerID {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("%w: %q", ErrScanUnknownMarkerHash, markerID)
		}
	}
	return nil
}

// D! id=cscn range-end

func (algorithm *CoreAlgorithm) EvaluateState(ctx CoreAlgorithmContext) (EvaluatedState, error) {
	if err := ctx.Validate(); err != nil {
		return EvaluatedState{}, err
	}
	switch action := ctx.Action.(type) {
	case TodoAction:
		return algorithm.evaluateTodoAction(ctx, action)
	case ResetClosureAction:
		return algorithm.evaluateResetClosureAction(ctx, action)
	default:
		return EvaluatedState{}, fmt.Errorf("%w: %T", ErrUnknownAction, ctx.Action)
	}
}

// D! id=ctodo range-start
func (algorithm *CoreAlgorithm) evaluateTodoAction(ctx CoreAlgorithmContext, action TodoAction) (EvaluatedState, error) {
	if err := validateScanCoversAllNodes(action.Scan, ctx.Specs, ctx.Markers); err != nil {
		return EvaluatedState{}, err
	}
	closures := DeriveClosures(ctx.Specs, ctx.Markers, ctx.Edges, action.Scan)
	return EvaluatedState{
		Specs:    ctx.Specs,
		Markers:  ctx.Markers,
		Edges:    ctx.Edges,
		Closures: closures,
	}, nil
}

// D! id=ctodo range-end

// D! id=crst range-start
// evaluateResetClosureAction locates the closure by hash in the
// freshly-derived set, then applies each of its events to baseline.
// Broken-edge-only closures are refused (require scan fix).
func (algorithm *CoreAlgorithm) evaluateResetClosureAction(ctx CoreAlgorithmContext, action ResetClosureAction) (EvaluatedState, error) {
	if err := validateScanCoversAllNodes(action.Scan, ctx.Specs, ctx.Markers); err != nil {
		return EvaluatedState{}, err
	}
	closures := DeriveClosures(ctx.Specs, ctx.Markers, ctx.Edges, action.Scan)
	var target *Closure
	for i := range closures {
		if closures[i].Hash == action.Hash {
			target = &closures[i]
			break
		}
	}
	if target == nil {
		return EvaluatedState{}, fmt.Errorf("%w: %q", ErrClosureNotFound, action.Hash)
	}
	// Refuse if closure has ONLY broken-edge events.
	allBroken := true
	for _, ev := range target.Events {
		if ev.Kind != EventEdgeBroken {
			allBroken = false
			break
		}
	}
	if allBroken && len(target.Events) > 0 {
		return EvaluatedState{}, fmt.Errorf("%w: closure %q", ErrBrokenEdgeNotResettable, action.Hash)
	}

	specsByID := copySpecsToMutableMap(ctx.Specs)
	markersByID := copyMarkersToMutableMap(ctx.Markers)
	edges := append([]Edge(nil), ctx.Edges...)

	for _, ev := range target.Events {
		switch ev.Kind {
		case EventNodeChanged:
			if s, ok := specsByID[ev.NodeID]; ok {
				s.Hash = ev.NewHash
			} else if m, ok := markersByID[ev.NodeID]; ok {
				m.Hash = ev.NewHash
			}
		case EventNodeAdded:
			// New node — establish baseline hash from scan. The reconciler
			// put the node in the baseline list with Hash=""; reset persists
			// the scan hash so the next todo is clean.
			if s, ok := specsByID[ev.NodeID]; ok {
				s.Hash = ev.NewHash
			} else if m, ok := markersByID[ev.NodeID]; ok {
				m.Hash = ev.NewHash
			}
		case EventNodeRemoved:
			delete(specsByID, ev.NodeID)
			delete(markersByID, ev.NodeID)
			// Remove edges touching the removed node.
			filtered := make([]Edge, 0, len(edges))
			for _, e := range edges {
				if e.From == ev.NodeID || e.To == ev.NodeID {
					continue
				}
				filtered = append(filtered, e)
			}
			edges = filtered
		case EventEdgeAdded:
			if ev.Edge != nil {
				edges = addEdgeIfMissing(edges, *ev.Edge)
			}
		case EventEdgeRemoved:
			if ev.Edge != nil {
				edges = removeEdge(edges, *ev.Edge)
			}
		case EventEdgeBroken:
			// No-op: requires scan fix.
		}
	}

	// For NODE_ADDED events, ensure the node is in the output (it already is,
	// via reconciliation). For NODE_CHANGED, the hash has been updated above.
	// For NODE_REMOVED, the node has been deleted from the maps above.

	return EvaluatedState{
		Specs:    specsFromMutableMap(specsByID),
		Markers:  markersFromMutableMap(markersByID),
		Edges:    edges,
		Closures: closures,
	}, nil
}

// D! id=crst range-end

// D! id=cdrv range-start
// DeriveClosures implements the provenance-closure algorithm:
//
//  1. SEEDS — collect drift events. Each event has a seed node (the citer
//     side of the change — the node where the change is "from").
//  2. CLOSURE PER SEED — for each seed, BFS up the citer chain (walk
//     incoming edges). Markers cannot be cited, so propagation stops there.
//  3. MERGE same-hash closures (rare: tightly-coupled seed pairs).
//  4. SORT by hash for deterministic display.
//
// Closure membership is per-seed. Two seeds with overlapping citer chains
// produce two closures that share non-seed citers — that's intentional and
// the closures remain independently resettable.
func DeriveClosures(specs []Spec, markers []Marker, baselineEdges []Edge, scan Scan) []Closure {
	specsByID := copySpecsToMutableMap(specs)
	markersByID := copyMarkersToMutableMap(markers)

	baselineEdgeSet := make(map[string]bool, len(baselineEdges))
	for _, e := range baselineEdges {
		baselineEdgeSet[e.From+"\x00"+e.To] = true
	}
	scanEdgeSet := make(map[string]bool, len(scan.Edges))
	for _, e := range scan.Edges {
		scanEdgeSet[e.From+"\x00"+e.To] = true
	}

	eventsBySeed := map[string][]DriftEvent{}

	addEvent := func(seed string, ev DriftEvent) {
		ev.Seed = seed
		eventsBySeed[seed] = append(eventsBySeed[seed], ev)
	}

	// Node drift events. Branch on the reconciler's sentinel state so
	// reviewers see the right event kind:
	//   - Deleted==true    → NODE_REMOVED (baseline has it, scan doesn't)
	//   - Hash=="" && current != "" → NODE_ADDED (scan has it, baseline doesn't)
	//   - Hash != current  → NODE_CHANGED (both sides have it, content differs)
	for _, s := range specs {
		current := scan.SpecHashes[s.ID]
		switch {
		case s.Deleted:
			addEvent(s.ID, DriftEvent{
				Kind:   EventNodeRemoved,
				NodeID: s.ID,
			})
		case s.Hash == "" && current != "":
			addEvent(s.ID, DriftEvent{
				Kind:    EventNodeAdded,
				NodeID:  s.ID,
				NewHash: current,
			})
		case s.Hash != current:
			addEvent(s.ID, DriftEvent{
				Kind:    EventNodeChanged,
				NodeID:  s.ID,
				OldHash: s.Hash,
				NewHash: current,
			})
		}
	}
	for _, m := range markers {
		current := scan.MarkerHashes[m.ID]
		switch {
		case m.Deleted:
			addEvent(m.ID, DriftEvent{
				Kind:   EventNodeRemoved,
				NodeID: m.ID,
			})
		case m.Hash == "" && current != "":
			addEvent(m.ID, DriftEvent{
				Kind:    EventNodeAdded,
				NodeID:  m.ID,
				NewHash: current,
			})
		case m.Hash != current:
			addEvent(m.ID, DriftEvent{
				Kind:    EventNodeChanged,
				NodeID:  m.ID,
				OldHash: m.Hash,
				NewHash: current,
			})
		}
	}

	// New edges (in scan, not baseline). Skip edges whose To doesn't exist
	// in scan — those are EDGE_BROKEN events, not EDGE_ADDED.
	for _, e := range scan.Edges {
		if baselineEdgeSet[e.From+"\x00"+e.To] {
			continue
		}
		toExists := false
		if h, ok := scan.SpecHashes[e.To]; ok && h != "" {
			toExists = true
		}
		if !toExists {
			if h, ok := scan.MarkerHashes[e.To]; ok && h != "" {
				toExists = true
			}
		}
		if !toExists {
			continue
		}
		edgeCopy := e
		addEvent(e.From, DriftEvent{Kind: EventEdgeAdded, Edge: &edgeCopy})
	}
	// Removed edges (in baseline, not scan). Only applies to spec-spec
	// ref edges — link edges (marker-spec) are user-curated and never
	// appear in scan, so they're never "removed" by scan-side state.
	for _, e := range baselineEdges {
		if scanEdgeSet[e.From+"\x00"+e.To] {
			continue
		}
		if !isSpecID(e.From) || !isSpecID(e.To) {
			continue
		}
		edgeCopy := e
		addEvent(e.From, DriftEvent{Kind: EventEdgeRemoved, Edge: &edgeCopy})
	}
	// Broken edges (scan edge whose To endpoint doesn't exist in scan).
	for _, e := range scan.Edges {
		toExists := false
		if _, ok := scan.SpecHashes[e.To]; ok && scan.SpecHashes[e.To] != "" {
			toExists = true
		}
		if !toExists {
			if _, ok := scan.MarkerHashes[e.To]; ok && scan.MarkerHashes[e.To] != "" {
				toExists = true
			}
		}
		if !toExists {
			edgeCopy := e
			addEvent(e.From, DriftEvent{Kind: EventEdgeBroken, Edge: &edgeCopy})
		}
	}

	// Build incoming-edge map: incoming[T] = set of nodes that have an edge T-ward.
	// Specifically, for edge (F, T), F is a citer of T. incoming[T] += F.
	// We union baseline and scan edges.
	allEdgesSet := make(map[string]Edge)
	for _, e := range baselineEdges {
		allEdgesSet[e.From+"\x00"+e.To] = e
	}
	for _, e := range scan.Edges {
		allEdgesSet[e.From+"\x00"+e.To] = e
	}
	incoming := map[string]map[string]bool{}
	for _, e := range allEdgesSet {
		if incoming[e.To] == nil {
			incoming[e.To] = map[string]bool{}
		}
		incoming[e.To][e.From] = true
	}

	// Derive closure per seed.
	closuresByHash := map[string]*Closure{}
	for seed, events := range eventsBySeed {
		nodes := map[string]bool{seed: true}
		// Marker seeds include their linked specs (outgoing edge targets)
		// so the reviewer can see the specs the marker implements. Spec
		// seeds do NOT include their cited specs (those specs' text is
		// independent of the seed's drift).
		if !isSpecID(seed) {
			for _, e := range allEdgesSet {
				if e.From == seed {
					nodes[e.To] = true
				}
			}
		}
		queue := []string{}
		for n := range nodes {
			queue = append(queue, n)
		}
		for len(queue) > 0 {
			curr := queue[0]
			queue = queue[1:]
			for citer := range incoming[curr] {
				if nodes[citer] {
					continue
				}
				nodes[citer] = true
				queue = append(queue, citer)
			}
		}

		// Collect edges among closure nodes (undirected canonicalization).
		edgeKeySet := map[string]Edge{}
		for _, e := range allEdgesSet {
			if !nodes[e.From] || !nodes[e.To] {
				continue
			}
			key := canonicalEdgeKey(e.From, e.To)
			edgeKeySet[key] = e
		}

		nodeIDs := make([]string, 0, len(nodes))
		for id := range nodes {
			nodeIDs = append(nodeIDs, id)
		}
		sort.Strings(nodeIDs)

		edgeKeys := make([]string, 0, len(edgeKeySet))
		for k := range edgeKeySet {
			edgeKeys = append(edgeKeys, k)
		}
		sort.Strings(edgeKeys)

		hash := closureHash(nodeIDs, edgeKeys)

		// Build NodeRefs.
		nodeRefs := make([]NodeRef, 0, len(nodeIDs))
		for _, id := range nodeIDs {
			nodeRefs = append(nodeRefs, makeNodeRef(id, specsByID, markersByID))
		}

		// Build edge list (canonicalized).
		closureEdges := make([]Edge, 0, len(edgeKeys))
		for _, k := range edgeKeys {
			closureEdges = append(closureEdges, edgeKeySet[k])
		}

		if existing, ok := closuresByHash[hash]; ok {
			existing.Events = append(existing.Events, events...)
			if !containsString(existing.Seeds, seed) {
				existing.Seeds = append(existing.Seeds, seed)
			}
		} else {
			closuresByHash[hash] = &Closure{
				Hash:   hash,
				Nodes:  nodeRefs,
				Edges:  closureEdges,
				Events: append([]DriftEvent(nil), events...),
				Seeds:  []string{seed},
			}
		}
	}

	out := make([]Closure, 0, len(closuresByHash))
	for _, c := range closuresByHash {
		out = append(out, *c)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Hash < out[j].Hash
	})
	return out
}

// D! id=cdrv range-end

// canonicalEdgeKey returns an undirected canonicalization of an edge's
// endpoints: min(from,to) + separator + max(from,to). Two edges that
// differ only in direction produce the same key.
func canonicalEdgeKey(a, b string) string {
	if a < b {
		return a + "\x00" + b
	}
	return b + "\x00" + a
}

func closureHash(sortedNodeIDs, sortedEdgeKeys []string) string {
	h := sha1.New()
	for _, id := range sortedNodeIDs {
		h.Write([]byte(id))
		h.Write([]byte{0})
	}
	for _, k := range sortedEdgeKeys {
		h.Write([]byte(k))
		h.Write([]byte{0})
	}
	full := hex.EncodeToString(h.Sum(nil))
	if len(full) >= 8 {
		return full[:8]
	}
	return full
}

func makeNodeRef(id string, specsByID map[string]*Spec, markersByID map[string]*Marker) NodeRef {
	if s, ok := specsByID[id]; ok {
		return NodeRef{ID: id, IsSpec: true, Filepath: s.Filepath, LineNumber: s.LineNumber}
	}
	if m, ok := markersByID[id]; ok {
		return NodeRef{ID: id, IsSpec: false, Filepath: m.Filepath, LineNumber: m.LineNumber}
	}
	return NodeRef{ID: id}
}

// --- helpers ---

func copySpecsToMutableMap(specs []Spec) map[string]*Spec {
	specsByID := make(map[string]*Spec, len(specs))
	for i := range specs {
		spec := specs[i]
		specsByID[spec.ID] = &spec
	}
	return specsByID
}

func copyMarkersToMutableMap(markers []Marker) map[string]*Marker {
	markersByID := make(map[string]*Marker, len(markers))
	for i := range markers {
		marker := markers[i]
		markersByID[marker.ID] = &marker
	}
	return markersByID
}

func specsFromMutableMap(specsByID map[string]*Spec) []Spec {
	out := make([]Spec, 0, len(specsByID))
	for _, spec := range specsByID {
		out = append(out, *spec)
	}
	return out
}

func markersFromMutableMap(markersByID map[string]*Marker) []Marker {
	out := make([]Marker, 0, len(markersByID))
	for _, marker := range markersByID {
		out = append(out, *marker)
	}
	return out
}

func addEdgeIfMissing(edges []Edge, e Edge) []Edge {
	for _, existing := range edges {
		if existing == e {
			return edges
		}
	}
	return append(edges, e)
}

func containsString(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}

// removeEdge drops any edge matching target in either direction. The
// undirected match is safe because Validate rejects directed cycles among
// spec-spec edges, so a baseline edge and its reverse can never coexist.
// If the cycle-rejection invariant is ever relaxed, this function must
// restrict itself to forward-only matching.
func removeEdge(edges []Edge, target Edge) []Edge {
	out := make([]Edge, 0, len(edges))
	for _, e := range edges {
		if (e.From == target.From && e.To == target.To) ||
			(e.From == target.To && e.To == target.From) {
			continue
		}
		out = append(out, e)
	}
	return out
}
