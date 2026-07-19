# Provenance Closures — Consolidated Execution Plan

## End state (one paragraph)

Specs and markers are symmetric nodes in a directed citation graph (`A →s B` = spec A declares a ref to spec B; `M →m S` = marker M links to spec S). Both specs and markers can drift. Drift propagates **in the citer direction only** (cited → citer), transitive to fixpoint. Markers cannot be cited, so drift through a marker stops there. Each drift event seeds a **closure**: the seed node plus all transitively-reachable citers, plus edges among them. Closures are strictly disjoint across seeds (no merging); non-seed citers may appear in multiple closures. Closure identity is the first 8 hex chars of `SHA1(sorted nodes + sorted edges)`, stable across drift-state changes. `drift todo` outputs closures. `drift reset <hash>` syncs the closure's seed events from scan into baseline. State.xml v4 stores baseline only — no `EdgeResolution` table.

## Conceptual model

- **Nodes**: specs and markers, treated symmetrically. Both can drift; both propagate.
- **Edges**: `marker → spec` (link, user-declared via `drift link`) and `spec → spec` (ref, auto-parsed from `<ref>`). Both are full propagation edges.
- **The single retained asymmetry**: `<ref>` cannot target a marker. Only specs are ref-targets. (Already true; remains true.)
- **Closure**: per-seed derivation — seed + transitive citers. Identified by hash of its membership. No source attribution.
- **State.xml**: baseline only. No `EdgeResolution` table. `drift reset <hash>` syncs baseline to scan for every event the closure's seed originated.

## Algorithm specification

```
DeriveClosures(scan, baseline) []Closure:

  STEP 1 — SEEDS
    For each baseline node N with scan_hash(N) != baseline_hash(N):
      register event {NODE_CHANGED, seed: N}
    For each scan edge (X, Y) not in baseline:
      register event {EDGE_ADDED, seed: X}; register event {EDGE_ADDED, seed: Y}
    For each baseline edge (X, Y) not in scan:
      register event {EDGE_REMOVED, seed: X}; register event {EDGE_REMOVED, seed: Y}
    For each scan edge (X, Y) where Y not in scan:
      register event {EDGE_BROKEN, seed: X}
    For each scan node N not in baseline:
      register event {NODE_ADDED, seed: N}
    For each baseline node N not in scan:
      register event {NODE_REMOVED, seed: N}

  STEP 2 — CLOSURE PER SEED
    Build incoming-edge map: for each edge (F, T) in (baseline ∪ scan),
    incoming[T] += F.

    For each seed S:
      closure_nodes = {S}
      BFS from S over incoming edges:
        For each C in incoming[S']: add C; recurse from C.
      closure_edges = undirected edges among closure_nodes
      closure_events = all events registered with seed == S
      closure_hash = first 8 hex chars of SHA1(
        sort(closure_nodes) ++ sort(closure_edges)
      )

  STEP 3 — MERGE SAME-HASH CLOSURES
    Distinct seeds producing identical hashes (rare: tightly-coupled seed pair
    where each cites the other) → merge into one closure with combined events.

  STEP 4 — RETURN
    Closures sorted by hash for deterministic display.
```

### Propagation rule (directed)

When node X becomes drift-impacted, find all nodes Y such that `Y → X` is an edge (in either baseline or scan). Y becomes drift-impacted. Recurse to fixpoint.

- Drift on node X propagates to nodes that cite X.
- "Cite" includes: spec→spec refs and marker→spec links (markers cite specs).
- Specs can be cited (by specs and markers).
- Markers cannot be cited.
- Consequence: marker drift stays local (no citers); spec drift walks the citer chain.

### Closure grouping rule (strictly disjoint)

Each seed produces its own closure. Closures can share non-seed citers (overlap on intermediate nodes), but each closure's events belong only to its seed. Two seeds never produce one merged closure unless their membership is identical (same hash → merge events).

### Reset semantics

`drift reset <hash>` walks the closure's events and applies per-event sync rules:

| Event kind | Reset action |
|---|---|
| `NODE_CHANGED` | Set baseline hash = scan hash for that node |
| `NODE_ADDED` | Add node (with scan hash) to baseline |
| `NODE_REMOVED` | Remove node from baseline |
| `EDGE_ADDED` | Add edge to baseline |
| `EDGE_REMOVED` | Remove edge from baseline |
| `EDGE_BROKEN` | No-op (requires scan fix — add missing target or remove the ref) |

Reset is per-seed-events: only the closure's seed's events sync. Non-seed citers' state unchanged. Other closures' events untouched.

## Truth tables

| # | Use case | Closure shape |
|---|---|---|
| 1 | Spec S edited, isolated (no refs in or out, no markers) | `{S}`, 1 node, 0 edges |
| 2 | Spec S edited, S cites S' (S →s S') | `{S}` — S' is cited BY S; not the citer direction. S' not included. |
| 3 | Spec S edited, marker M links to S (M →m S) | `{S, M}` — M cites S, so M is in closure |
| 4 | Spec S edited, S' cites S (S' →s S) | `{S, S'}` plus transitive citers of S' |
| 5 | Spec S edited, complex citation graph | `{S}` ∪ {transitive citers of S} ∪ {markers linked to those specs transitively reached via citer chain} |
| 6 | Marker M edited, linked to S | `{M, S}` — edge (M, S) drifted; both endpoints drift-impacted. Plus citers of S. |
| 7 | Marker M edited, multi-linked to S1 and S2 | `{M, S1, S2}` plus citers of S1, citers of S2. Both edges drifted. |
| 8 | Spec S edited AND marker M linked to S also edited independently | Two closures: `closure_S` = {S, M, citers of S}; `closure_M` = {M, S, citers of S}. Same membership if no other drift — merge to single closure with both events. |
| 9 | Specs S1 and S2 both edited, S2 cites S1 | Two closures: `closure_S1` = {S1, S2, citers of S2, …}; `closure_S2` = {S2, citers of S2, …}. Same membership → merge to one closure with both events. |
| 10 | Specs S1 and S2 both edited, no citation relationship | Two closures, disjoint: `closure_S1` and `closure_S2`. |
| 11 | Specs S1 and S2 both edited, both cited by S3 | Two closures: `closure_S1` = {S1, S3, citers of S3, …}; `closure_S2` = {S2, S3, citers of S3, …}. S3 in both. Strict disjoint. |
| 12 | New ref `(A →s B)` declared in scan | Closure seeded by A: {A, citers of A}. Closure seeded by B: {B, A's transitive citers, citers of B}. May merge if same membership. |
| 13 | Ref `(A →s B)` removed from scan | Closure seeded by A: {A, citers of A}. Closure seeded by B: {B, citers of B}. |
| 14 | Broken ref `(A →s B)` where B doesn't exist in scan | Closure seeded by A: {A, citers of A}. B not a node. Reset no-ops on broken event. |
| 15 | Spec S added in scan, no refs/links | `{S}`, 1 node, 0 edges. Event NODE_ADDED. |
| 16 | Spec S deleted from scan, no refs/links | `{S}`, 1 node, 0 edges. Event NODE_REMOVED. |
| 17 | Marker M linked to S1 and S2; S1 drifts | `closure_S1` = {S1, M (because M cites S1)} + citers of S1 + citers of M (none). S2 NOT in closure. |
| 18 | Marker M linked to S1 and S2; M drifts | `closure_M` = {M, S1, S2} + citers of S1 + citers of S2. Both edges drifted (M is endpoint). |

Closure identity is membership-based. Adding drift inside an existing closure doesn't change its hash. Only membership changes (added/removed nodes or edges) change the hash.

## Type definitions

```go
// core/core.go — new types

type Closure struct {
    Hash   string       // 8 hex chars
    Nodes  []NodeRef    // sorted by ID
    Edges  []Edge       // sorted by undirected key (min(from,to) + \x00 + max(from,to))
    Events []DriftEvent // all events with seed in this closure
}

type DriftEvent struct {
    Kind    EventKind
    NodeID  string  // for node events
    Edge    *Edge   // for edge events
    OldHash string  // for NODE_CHANGED
    NewHash string  // for NODE_CHANGED
    Seed    string  // ID of originating seed node
}

type EventKind int
const (
    EventNodeChanged EventKind = iota
    EventNodeAdded
    EventNodeRemoved
    EventEdgeAdded
    EventEdgeRemoved
    EventEdgeBroken
)
```

Retired types/fields: `Todo`, `TodoKind`, `SourceSpecID`, `EdgeResolution`, all rhizomatic-closure language.

## State.xml v4 schema

```xml
<driftState version="4">
  <specs>
    <spec id="..." hash="..." filepath="..." />
  </specs>
  <markers>
    <marker id="..." hash="..." filepath="..." line="..." endLine="..." />
  </markers>
  <edges>
    <edge from="..." to="..." />
  </edges>
</driftState>
```

Changes from v3: dropped `<edgeResolutions>` entirely. No migration path (clean break). State version constant bumps to 4. Refuse to load v3 files with a clear error message: *"state.xml v3 is unsupported; delete .drift/ and run drift init"*.

## CLI surface change

| Old | New |
|---|---|
| `drift todo` (flat todos) | `drift todo` (closures list) |
| `drift diff <marker>` | `drift diff <hash>` |
| `drift diff <marker> <spec>` | Removed |
| `drift diff <spec>` | Removed |
| `drift diff --all` | `drift diff --all` (iterates closures) |
| `drift reset <marker> <spec>` | Removed |
| `drift reset <spec> <spec>` | Removed |
| `drift reset <id>` (orphan) | Removed — folded into `reset <hash>` |
| (new) | `drift reset <hash>` |
| `drift link`, `unlink`, `show`, `list`, `config theme` | Unchanged |

Dots-discrimination dispatch table deleted entirely.

## Presenter shape (Plain / Color)

```
N closures with drift.

Closure a3f7b2c1  (5 nodes, 4 edges)
  Events:
    [NODE_CHANGED] spec "core.validate" (core.drift.xml:12)
      baseline: abc12345 → scan: def67890
    [EDGE_ADDED]   spec "core.validate" → spec "utils.hash"
  Members:
    specs:   core.validate, utils.hash, main.boot
    markers: cval, m2
  Inspect: drift diff a3f7b2c1
  Resolve: drift reset a3f7b2c1

Closure e1d4f8a9  (2 nodes, 1 edge)
  Events:
    [NODE_CHANGED] marker "cval" (core.go:42)
      ...

Closure 7c2b9e5f  (1 node, 0 edges)  [orphan]
  Events:
    [NODE_ADDED] spec "main.newspec" (main.drift.xml:8)
```

JSON shape: `{ "closures": [{ "hash", "nodes", "edges", "events" }] }`. Clean break — no `todos` array.

## File-by-file change list

### Core algorithm (`core/`)
- **`core.go`**: define `Closure`, `DriftEvent`, `EventKind`. Implement `DeriveClosures`. Delete `computeEdgeTodos`, `computeRhizomaticClosureTodos`, `Todo`, `TodoKind`, `SourceSpecID`.
- **`core_test.go`**: replace existing todo tests with truth-table-driven closure tests (one test per row in truth tables above).
- **`core.drift.xml`**: rewrite specs — `todo_action`, `reset_action`, `edge_todo_algorithm`, `rhizomatic_closure` → `provenance_closure`. Remove source-of-truth asymmetry language.

### Scanner (`scanner/`)
- **`scanner.go`**: no structural change (already produces what we need). Optional: pre-compute incoming-edge map for closure algorithm.
- **`scanner.drift.xml`**: minor wording updates only.

### State store (`statestore/`)
- **`pin_file.go`**: state.xml v4 schema. Drop `<edgeResolutions>` parsing/serialization. Drop `EdgeResolution` type, `RecordEdgeResolution`, `LookupEdgeResolution`. Add `SyncClosure(Closure, scan)` that applies the per-event sync rules.
- **`pin_file_test.go`**: update for v4. Drop resolution-related tests.
- **State version constant**: bump to 4. Refuse to load v3 files with clear error.

### Orchestrator (`orchestrator/`)
- **`orchestrator.go`**: replace `Reset(from, to string)` with `ResetClosure(hash string)`. Look up closure by hash from current scan+baseline, walk its events, call `statestore.SyncClosure`. Lock semantics unchanged.
- **`orchestrator.drift.xml`**: rewrite method specs.

### Model (`model.drift.xml`)
- Rewrite `model.rhizomatic` as **`model.provenance`**. New axioms:
  - Citation graph is directed (citer → cited).
  - Drift propagates in citer direction (cited → citer), transitive to fixpoint.
  - Markers cannot be cited; drift through markers stops.
  - No directed cycles among spec-spec edges (cycle detection unchanged).
  - State is baseline-only; reset = sync to scan.

### CLI commands (`cli/commands/`)
- **`todo.go`**: call `DeriveClosures`, pass to presenter.
- **`diff.go`**: accept single hash arg or `--all`. Drop `<marker>`, `<spec>`, `<marker> <spec>` forms.
- **`reset.go`**: accept single hash arg. Drop dots-discrimination dispatch entirely.
- **`link.go`, `unlink.go`**: unchanged.
- **`show.go`**: unchanged.
- **`list.go`**: unchanged (optional: closure grouping when drift active — defer if nontrivial).
- **`cli.drift.xml`**: rewrite `todo_command`, `diff_command`, `reset_command` specs.

### Output layer (`cli/output/`)
- **`presenters_plain.go`**: closure-grouped output.
- **`presenters_color.go`**: closure-grouped with theme colors.
- **`presenters_json.go`**: `{ closures: [...] }` schema.
- **`themes.go`**: add closure-level elements (closure header, event-kind colors).
- **`tokenizer.go`**: minor vocabulary updates.
- **`output.drift.xml`, `output_impl.drift.xml`**: rewrite presenter specs.

### Eval (`eval/`)
- Fixture updates: every fixture that asserts todo shape needs updating to closure shape. May require prompt rewrites too. Defer to end of implementation.

### Docs
- **`AGENTS.md`**: rewrite workflow section around closures. Update reset dispatch table (deleted), replace with closure-reset description. Update counts (specs/markers/closures).
- **`DOCUMENTATION.md`**: rewrite state.xml section (v4), drift-kinds section (events), reset semantics, propagation algorithm description. Drop "rhizomatic" everywhere; replace with "provenance."
- **`README.md`**: update anatomy section. Drop "links/refs" framing in favor of "edges in the citation graph."
- **`cli/skill.md`**: rewrite LLM workflow around closures.
- **`cli/help.txt`**: update command reference (diff/reset take hashes, no dots dispatch).

## Re-baselining sequence (drift is self-hosting)

Because drift tracks its own specs, the refactor triggers drift on drift. Sequence:

1. Make all code changes (don't touch specs yet). Build may fail at the `make build` gate.
2. Update all `*.drift.xml` specs to reflect new behavior.
3. Run `drift todo` — should show closures seeded by every spec we edited.
4. Walk each closure via `drift diff <hash>`. Verify spec text matches new code behavior.
5. `drift reset <hash>` per closure.
6. Run `make build` — should pass clean.
7. Run `go test -race -count=1 ./...`.
8. Update docs (AGENTS.md, etc.).
9. Run `drift todo` — doc-affecting markers may drift. Review and reset.
10. Final test pass + Windows build check.
11. Update eval fixtures. Run eval battery last.

## Test plan (truth-table driven)

Tests live in `core/core_test.go`. One test function per row, all using a small fixture workspace:

| # | Test name | Setup | Assertion |
|---|---|---|---|
| 1 | `TestClosure_SingletonSpec` | Baseline: spec S, no edges. Edit S. | Closure = {S}, 1 node, 0 edges, 1 event NODE_CHANGED. |
| 2 | `TestClosure_CiterDirection` | Baseline: S, S' with S →s S'. Edit S. | Closure = {S}; S' NOT in closure. |
| 3 | `TestClosure_MarkerAsCiter` | Baseline: S, M with M →m S. Edit S. | Closure = {S, M}. |
| 4 | `TestClosure_MultiLinkMarkerDrift` | Baseline: M →m S1, M →m S2. Edit M. | Closure = {M, S1, S2, citers of S1, citers of S2}. |
| 5 | `TestClosure_MultiLinkSpecDrift` | Baseline: M →m S1, M →m S2. Edit S1. | Closure = {S1, M, citers of S1}. S2 NOT in closure. |
| 6 | `TestClosure_StrictDisjoint` | Baseline: S1, S2 both cited by S3. Edit S1 and S2 independently. | TWO closures. S3 in both. |
| 7 | `TestClosure_NewEdgeMerges` | Baseline: S1 alone. Scan: S1 + new edge S1 →s S2 (S2 exists). | Closure includes S1 and S2; events EDGE_ADDED on both seeds. |
| 8 | `TestClosure_RemovedEdge` | Baseline: S1 →s S2. Scan: edge gone. | Closure includes both endpoints; events EDGE_REMOVED. |
| 9 | `TestClosure_BrokenEdge` | Baseline: S1 alone. Scan: S1 + broken ref to nonexistent S2. | Closure = {S1}, events EDGE_BROKEN. Reset no-ops on broken event. |
| 10 | `TestClosure_OrphanAdded` | Baseline: empty. Scan: S (no edges). | Closure = {S}, events NODE_ADDED. |
| 11 | `TestClosure_OrphanRemoved` | Baseline: S (no edges). Scan: empty. | Closure = {S}, events NODE_REMOVED. |
| 12 | `TestClosure_HashStability` | Same setup as #3; reset; edit S again. | Closure hash is the same as first time. |
| 13 | `TestClosure_HashChangesOnMembership` | Add a citer to S between runs. | Hash differs. |
| 14 | `TestClosure_PerSeedReset` | Setup as #6. Reset closure_S1 only. | closure_S2 still present in next todo. |
| 15 | `TestClosure_BrokenEdgePersists` | Closure has NODE_CHANGED + EDGE_BROKEN. Reset. | NODE_CHANGED syncs; EDGE_BROKEN remains as event in next todo. |
| 16 | `TestClosure_CycleStillRejected` | Baseline: S1 →s S2, S2 →s S1. | Scanner/validation rejects. |

Plus:
- `cli/race_test.go`: update for closure API. Confirm concurrent `DeriveClosures` (read-only) and `ResetClosure` (write under lock) are race-free.
- `statestore/pin_file_test.go`: v4 schema round-trip; refuse v3 files.
- Determinism: same scan+baseline → same closures, same order.

## Implementation sequence

Order matters for keeping the build working at each step where possible.

1. **State.xml v4 schema** — `statestore/pin_file.go`, drop `<edgeResolutions>`. Bump version. Refuse v3. Drift will fail to load its own state — note this and proceed.
2. **Core types** — define `Closure`, `DriftEvent`, `EventKind` in `core/core.go`. Don't delete old types yet.
3. **Implement `DeriveClosures`** alongside old functions. Add truth-table tests. Get all passing.
4. **Delete old algorithm** — remove `computeEdgeTodos`, `computeRhizomaticClosureTodos`, `Todo`, `TodoKind`, `SourceSpecID`. Update everything that referenced them.
5. **Orchestrator** — `ResetClosure(hash)` replaces `Reset(from, to)`.
6. **Presenters** — rewrite Plain/Color/JSON for closures.
7. **CLI commands** — update todo/diff/reset to new signatures.
8. **First compile-and-run** — expect compile errors throughout `cli/commands/`, `cli/output/`. Fix iteratively.
9. **Spec rewrites** — model.drift.xml, core.drift.xml, scanner.drift.xml, orchestrator.drift.xml, cli.drift.xml, output.drift.xml, output_impl.drift.xml. Business/ if needed.
10. **First drift run** — `drift todo` will show many closures from the spec edits.
11. **Re-baseline** — review and reset each closure.
12. **Test pass** — `go test -race -count=1 ./...` clean. `make build` clean.
13. **Doc rewrites** — AGENTS.md, DOCUMENTATION.md, README.md, skill.md, help.txt.
14. **Second re-baseline** — doc-marker drift.
15. **Windows build check** — `GOOS=windows go build -o /dev/null ./statestore/`.
16. **Eval fixture updates** — eval/*.md and eval helpers.
17. **Eval run** — `go run ./eval --battery ...` last.
18. **Commit and push** — likely split across 2-3 commits (algorithm/state, specs+rebaseline, docs+rebaseline, eval).

## Checkpoints (pause for review)

- After step 4 (old algorithm deleted, new path compiles, truth-table tests pass).
- After step 11 (specs rewritten, drift re-baselined, `make build` clean).
- After step 14 (docs rewritten, second re-baseline clean).

Each checkpoint = a commit (or commit group) so we can roll back if anything goes sideways. Final commit only after eval passes.

## Open risks / things to watch during execution

1. **State.xml migration**: clean break means existing `.drift/state.xml` files become unloadable. Anyone (including the drift repo itself) must `rm -rf .drift && drift init` once. This is acceptable per design call but worth a note in the commit message.

2. **`DeriveClosures` performance**: for highly-connected spec graphs, the closure-per-seed walk is O(seeds × graph size). Should be fine for typical projects; flag if profiling shows issues.

3. **Closure hash display in TTY**: 8 hex chars in a colored presenter — make sure themes render it distinctly (e.g., bold cyan, like git short SHAs). Theme update needed.

4. **The `business/` spec hierarchy** uses spec-spec refs heavily. Under directed propagation + strict disjoint, changes to a high-level goal spec may produce many closures (one per downstream path). Worth confirming this matches expectation when we re-baseline.

5. **Eval harness shape change**: eval fixtures describe expected agent behavior. The closure-based output is substantially different from the old todo list. The eval prompts themselves may need rewriting (not just fixture updates). Budget extra time here.

6. **`drift list` and `drift show`**: under strict disjoint, `drift show <id>` could optionally list closures containing that node. Useful for navigation but adds presenter work. Defer if scope balloons.

## Terminology lock

- **`rhizomatic` → `provenance`** everywhere.
- `computeRhizomaticClosureTodos` → `DeriveClosures` (or `ComputeProvenanceClosures`).
- `model.rhizomatic` spec → `model.provenance` (rewritten).
- All doc references to "rhizomatic" → "provenance".
- The conceptual axiom becomes: **"Drift propagation follows the citer direction. A node's drift flags every node that transitively cites it. Markers cannot be cited, so drift through a marker stops there."**
- `SourceSpecID` field → deleted.
- `Todo` / `TodoKind` → replaced by `Closure` + `DriftEvent` + `EventKind`.
- `EdgeResolution` → deleted.
