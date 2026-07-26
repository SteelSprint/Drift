# Documentation Plan — Pedagogically-Structured, Drift-Tracked Docs

This plan delivers a complete documentation system for the drift project. The
docs are organized as a progressive book (Rust Book style) in man-page voice,
drift-tracked via pedagogically-structured specs, and synced with the embedded
agent guide via specs-as-contract.

A fresh contributor (human or LLM agent) should be able to read this plan and
execute it end-to-end without referring to any conversation or external
context beyond the drift codebase itself.

---

## 1. Project context

**Drift** is a spec-drift detection tool for LLM coding agents. Specs describe
behavior in `*.drift.xml` files. Markers (`// D! id=<shortcode> range-start`
/ `range-end`) wrap the implementing code. When any side changes, drift derives
**closures** (per-seed drift sets) so the reviewer can verify alignment before
re-baselining. The project dogfoods itself: drift tracks its own specs (157
specs, 96 markers, 184 edges as of this writing).

**Key recent feature**: `cli.reset_friction_block` — a runtime rate-limit that
blocks the 4th `drift reset` within any 30-second window. Bypass via
`--dangerously-override-friction` flag (emits stderr squawk + JSON `warning`
field). This flag is documented but NOT advertised in error messages.

**Current documentation state**:
- `README.md` — GitHub landing page
- `DOCUMENTATION.md` — comprehensive but flat reference
- `AGENTS.md` — agent-facing contributor guide
- `SPECIFICATIONS.md` — spec authoring constitution
- `PLAN.md` — historical post-eval plan (being replaced by this file)
- `cli/skill.md` — embedded agent guide, printed by `drift skill`
- `cli/help.txt` — embedded quick reference, printed by `drift help`

These overlap heavily (closures, events, marker placement, output modes each
explained 3-4 times across files). This plan consolidates and deduplicates.

---

## 1a. How drift works (mechanical reference)

A fresh contributor needs the following to execute this plan. All of this is
discoverable from the codebase, but stating it here saves time.

### The drift binary

Built via `make build` or `go build -o drift ./cmd/drift`. The binary lands
at `./drift` in the repo root. All commands in this plan assume the binary
is at `./drift` and the working directory is the repo root.

### Spec module structure

Specs live in `*.drift.xml` files. Each file has a `<module name="X">` root
(or `<main>` for the root module). Spec IDs are `module.localid` — exactly
one dot. The local ID (after the dot) MUST NOT contain a dot.

Example `docs.drift.xml`:
```xml
<module name="docs">
<spec id="pedagogy">...</spec>
<spec id="concept_closure">...</spec>
</module>
```
Spec IDs become `docs.pedagogy`, `docs.concept_closure`, etc.

### Ref syntax

Inside a spec, cite another spec:
```xml
<ref spec="glossary.closure">closure concept</ref>
```
Self-closing form: `<ref spec="glossary.closure" />`. The label text is
preserved in the canonical hash; the `<ref>` tag itself is stripped before
hashing. Renaming a referenced spec ID does NOT invalidate the referrer's
hash.

### Marker syntax

Markers wrap code regions. In Go:
<!-- D! instruction=ignore-span-start -->
```go
// D! id=cval range-start
func Validate(...) { ... }
// D! id=cval range-end
```

In markdown (HTML comments):
```markdown
<!-- D! id=docwhy range-start -->
... chapter content ...
<!-- D! id=docwhy range-end -->
```
<!-- D! instruction=ignore-span-end -->

Marker shortcodes (IDs) contain NO dot. Spec IDs contain exactly one dot.
The scanner is language-agnostic — any text file is a valid marker host.

### Ignore-span directives

When documentation about drift includes example marker syntax, the scanner
would pick up the examples as real markers. Ignore-span directives suppress
marker detection for a contiguous region. The directive uses the prefix
`D!` followed by `instruction=ignore-span-start` (open) or
`instruction=ignore-span-end` (close). In markdown:

```
[comment]: D instruction=ignore-span-start

... content with example D! id=... range-start lines ...

[comment]: D instruction=ignore-span-end
```

(Actual syntax uses `D!` not `D`; this example is shown with `D` to avoid
the scanner treating it as a real directive within this plan file.)

Lines between the directives are not scanned for markers. The directives
must be properly paired; nesting is not allowed. See scanner.ignore_span.

### Key commands

| Command | Purpose |
|---|---|
| `drift todo` | Derive closures; report drift. Exit 0 clean, 1 drift, 2 error. |
| `drift list [--verbose]` | List specs, markers, edges, sync state. |
| `drift show <marker\|spec>` | Show full citation closure of an entity. |
| `drift diff <hash>` | Show unified diffs for every node in a closure. |
| `drift diff --all` | Show diffs for all closures. |
| `drift link <marker> <module.spec>` | Create a marker→spec edge. Marker first, spec second. |
| `drift unlink <marker> <module.spec>` | Remove a marker→spec edge. |
| `drift reset <hash>` | Resolve a closure by syncing baseline to scan. |
| `drift reset --dangerously-override-friction <hash>` | Bypass the rate-limit block. |
| `drift skill` | Print the embedded agent guide (cli/skill.md). |
| `drift help` | Print the command reference (cli/help.txt). |

### The drift gate

`make build` runs `go build` followed by `./drift todo`. If `drift todo`
reports any drift (exit 1) or any unlinked markers, the build fails. This
is the dogfooding enforcement: drift's own specs MUST be clean before
the binary builds.

**Consequence**: every spec added to `docs.drift.xml` MUST have at least
one linked marker before `make build` will pass. Unlinked specs cause
`drift todo` to exit 1 with "N unlinked markers found."

### Canonical authority modules

Concept specs in `docs.drift.xml` cite canonical definitions from:
- `glossary.drift.xml` — `glossary.spec`, `glossary.marker`,
  `glossary.closure`, `glossary.edge`, `glossary.ref`, `glossary.seed`,
  `glossary.citer`, `glossary.baseline`, `glossary.scan`,
  `glossary.session` (10 definition specs)
- `core/core.drift.xml` — `core.provenance_closure`, `core.validate`,
  `core.todo_action`, `core.reset_action`, `core.scan_coverage`,
  `core.closure_event_ordering` (6 algorithmic contracts)
- `cli/cli.drift.xml` — `cli.reset_friction_block`, `cli.no_bulk_reset`,
  `cli.reset_format`, etc. (CLI command contracts)

When a concept spec says "canonical authority: glossary.closure", it means
the formal definition lives in `glossary.closure` and the concept spec is
the pedagogical wrapper.

---

## 2. Audience and voice

### Primary audience

**Solo human developers** evaluating or adopting drift. Secondary audiences:
LLM agent users (covered by `cli/skill.md`), contributors (covered by
`docs/contributors/`), spec authors (covered by `docs/spec-format.md`).

When tradeoffs arise (depth vs brevity, formal vs conversational), solo human
developers win.

### Voice: warm tutorial

**Warm tutorial** = the Rust Book's collaborative, patient approach. The
reader and the author build something together. The tone is
conversational but precise — like pairing with an expert who explains
why things work, not just how to use them.

**Core principles:**

- Use "you" and "we/let's" freely. The reader is your pair programming
  partner. "Let's install drift." "Your agent writes specs." "We'll
  cover closures in the next chapter."
- Build a sustained project across chapters. The reader builds a real
  todo app, adding drift to it chapter by chapter.
- Show the loop: write code, drift catches something, read the diff,
  decide what to do, fix or resolve. This is the Rust Book's signature
  pattern: deliberate change, system feedback, explanation, decision.
- Treat closures as teaching moments, not just output. Explain what
  each part means and what the reader should learn.
- Use forward references. "We'll cover the citer chain in chapter 7 —
  for now, just know that changing a spec may affect others."
- Anticipate questions. "You might wonder why drift doesn't distinguish
  a refactor from a behavior change. That's intentional — here's why."
- Celebrate milestones briefly. "Drift caught it!" "Your project is
  now tracked." Not saccharine, just human.
- Be patient. Explain WHY something works, not just HOW to do it.
- Prefer prose over tables. Tables are reference material; prose is
  tutorial.
- No RFC 2119 keywords in docs prose. They are an internal spec
  convention, not something to push on users.
- No emoji.

**Examples:**

| Good (warm tutorial) | Bad (terse/impersonal) |
|---|---|
| "Let's see what happens when the agent changes something." | "The agent edits code inside an existing marker." |
| "Drift caught it! The closure shows..." | "A NODE_CHANGED event fires." |
| "You might wonder: does drift distinguish a refactor from a behavior change?" | "Refactors produce NODE_CHANGED events." |
| "Your project is now drift-tracked." | "The project is clean." |

Each chapter opens with a one-paragraph orientation: what we'll build,
what the reader will be able to do after, and what prerequisite
chapters are assumed.

### Sub-category framing

Drift creates and owns a new tool category. The docs MUST teach the problem
(spec drift in LLM-assisted codebases) before the solution. Chapter 01 leads
with the problem statement. README links to chapter 01 as the entry point.

---

## 3. Information architecture

### Final folder structure

```
README.md                              (slim landing: pitch + install + 3-line quickstart)
AGENTS.md                              (stub: agent readers pointed at cli/skill.md)
CONTRIBUTING.md                        (NEW: GitHub convention, points to docs/)

docs/
  index.md                             (TOC + reading-order guide)
  01-why-drift-exists.md               (the problem; required first read)
  02-getting-started.md                (install + first run + first spec/marker/link)
  03-specs-and-markers.md              (core concepts, worked example)
  04-closures.md                       (how drift detects changes; citation graph; events)
  05-workflow.md                       (todo → diff → reset, daily use)
  06-friction.md                       (one-closure-at-a-time; rate limit; override flag)
  07-output-modes.md                   (plain/color/json; themes; JSON contract)
  08-spec-audit.md                     (periodic semantic audit, human-voiced)
  09-internals.md                      (state.xml; .drift/ layout; dogfooding)
  10-contributing.md                   (repo layout; build/test; editing tracked code)
  spec-format.md                       (was SPECIFICATIONS.md — RFC 2119 spec authoring guide)
  contributors/
    agent-guide.md                     (was AGENTS.md — full content, drift-contributor focus)

docs.drift.xml                         (NEW: specs-as-contract for the docs themselves)

cli/skill.md                           (UNCHANGED in content: agent-canonical, terse imperative)
cli/help.txt                           (UNCHANGED)

DELETE: DOCUMENTATION.md               (redundant with new chapters)
DELETE: PLAN.md                        (replaced by this file — wait, this IS this file)
```

Note: PLAN.md is this file. After execution, it remains as the documentation
plan reference. DOCUMENTATION.md is deleted (content migrated into docs/
chapters).

### Canonical rule

- **`docs/`** is the human-canonical surface. Written in terse-progressive
  voice for humans reading top-to-bottom.
- **`cli/skill.md`** is the agent-canonical surface. Written in terse
  imperative voice for LLM agents running `drift skill` in any project. It
  stays self-contained (embedded in the binary via `go:embed`).
- Topics covered in BOTH are governed by **sync specs** in `docs.drift.xml`
  (see §5). Both docs link to the same sync spec; changing the spec drifts
  both markers.

---

## 4. Pedagogical framework

### Outcome-based education (OBE)

Every chapter spec follows OBE principles:

1. **Learning objectives** — 3-7 measurable abilities the reader gains.
2. **Content requirements** — what the chapter MUST cover to satisfy
   objectives.
3. **Prerequisites** — cited via `<ref>` to concept specs and earlier chapter
   specs.
4. **Formative assessment** — at least one "Try it yourself" exercise with a
   verifiable outcome.

### Bloom's taxonomy verbs

Learning objectives MUST use measurable Bloom verbs. Forbidden verbs are
unmeasurable.

| Allowed (measurable) | Forbidden (unmeasurable) |
|---|---|
| identify, list, name | know |
| describe, explain, summarize | understand |
| apply, demonstrate, implement | learn |
| analyze, compare, contrast | see |
| evaluate, justify, critique | grasp |
| design, construct, build | appreciate |

### Concept-level prerequisite graph

The prereq graph operates at **concept granularity**, not just chapter
granularity. Atomic concept specs (e.g. `concept_closure`) are cited by
multiple chapter specs (spiral curriculum — a concept is introduced briefly
in an early chapter, defined formally in a middle chapter, applied in a
later chapter).

Concepts form a DAG. Drift's existing cycle-rejection (`core.validate`)
enforces acyclicity. Changing a foundational concept propagates drift to
every chapter that cites it.

### Exercise style: goal + verification

Each chapter MUST include at least one "Try it yourself" exercise. Exercises
state a goal and a verification step — not a step-by-step recipe. The reader
figures out the steps; the exercise tells them how to check success.

**Example exercise**:
> **Try it yourself.** Create a second spec in `main.drift.xml`, place a
> marker in your code wrapping the implementing function, and run
> `drift link <marker> <module.spec>`. Verify: `drift todo` reports
> "No changes detected" with the spec and marker counts incremented.

---

## 5. Drift-tracking mechanism — `docs.drift.xml`

A new spec module at the repo root. Contains three tiers of specs:

### Tier 1 — Meta spec (1 spec)

`docs.pedagogy` — establishes the rules every other docs spec follows. R1-R7
cover: learning objectives must use Bloom verbs; prereqs via `<ref>`; content
requirements section; formative assessment requirement; concept specs may be
cited by multiple chapters; sync specs govern dual-source topics; concept
prereqs cited before chapter prereqs.

### Tier 2 — Concept specs (10 specs, atomic and foundational)

| Spec ID | Concept | Taught at depth in |
|---|---|---|
| `docs.concept_spec` | What a spec is | ch.1 (intro), ch.3 (define) |
| `docs.concept_marker` | What a marker is | ch.1 (intro), ch.3 (define) |
| `docs.concept_edge` | The unified edge (marker→spec and spec→spec) | ch.3 (intro), ch.4 (define) |
| `docs.concept_baseline` | What baselines are (state.xml + baselines.bin) | ch.3 (intro), ch.9 (deep) |
| `docs.concept_scan` | What scans are (filesystem walk producing current hashes) | ch.3 (intro), ch.9 (deep) |
| `docs.concept_closure` | What closures are (per-seed drift sets) | ch.1 (brief), ch.4 (define), ch.8 (apply) |
| `docs.concept_drift_event` | The 6 event types | ch.4 (define), ch.5 (apply) |
| `docs.concept_citer_chain` | Cited→citer propagation | ch.4 (define) |
| `docs.concept_friction` | Per-closure review pacing (rate limit) | ch.6 (define) |
| `docs.concept_output_mode` | Plain/Color/JSON modes | ch.7 (define) |

Each concept spec is short (~10-15 lines): definition + "Taught at depth in"
list + canonical authority ref (cites `glossary.*` or `core.*` for the
formal contract).

**Concept spec template**:

```xml
<spec id="concept_closure">
Overview: Atomic concept: a per-seed drift set computed by
core.DeriveClosures. Closure membership = seed + transitive citers
(plus, for marker seeds, the linked specs). Ephemeral (not stored in
state.xml). Identity is the first 8 hex chars of SHA1(sorted node IDs
+ sorted undirected edge keys).

Taught at depth in:
  - docs.chapter_01_why_drift_exists (introduced briefly, by analogy)
  - docs.chapter_04_closures (defined formally)
  - docs.chapter_08_spec_audit (applied as audit unit)

Definition authority: <ref spec="glossary.closure">glossary.closure</ref>
(canonical definition),
<ref spec="core.provenance_closure">core.provenance_closure</ref>
(algorithmic contract).
</spec>
```

### Tier 3 — Chapter specs (10 specs, one per chapter)

Each chapter spec follows this template:

```xml
<spec id="chapter_04_closures">
Overview: The reader learns how drift derives closures, the six event
types, and citer-chain propagation.

Learning objectives: after reading docs/04-closures.md, the reader can:
  - define what a closure is
  - describe each of the six drift event types and when each fires
  - explain the per-seed derivation model (seed + transitive citers)
  - read a closure hash and explain what it encodes and omits
  - trace drift propagation along the citer chain for a worked example
  - identify when a marker cannot be cited and explain why

Prerequisites:
  - <ref spec="docs.concept_edge">edge concept</ref>
  - <ref spec="docs.concept_closure">closure concept</ref>
  - <ref spec="docs.concept_drift_event">drift event concept</ref>
  - <ref spec="docs.chapter_03_specs_and_markers">chapter 3</ref>

Content requirements:
  - MUST walk through a worked example: 2 specs, 1 marker, 1 ref,
    showing how NODE_CHANGED on the marker produces a closure containing
    the linked spec and the citing spec
  - MUST enumerate all six event types with trigger conditions
  - MUST explain closure identity (SHA1 of sorted node IDs + sorted
    undirected edge keys, first 8 hex chars)
  - MUST explain strict-disjoint across seeds
  - MUST explain the marker asymmetry (markers can cite but cannot be cited)
  - MUST include ≥1 "Try it yourself" exercise (marker: docclos_ex)
    where the reader creates a closure by drifting a marker, runs
    `drift todo`, and identifies the closure hash

See also: docs.sync_closures (cross-source sync with skill.md),
docs.chapter_05_workflow (next chapter).
</spec>
```

### Tier 4 — Sync specs (8 specs, sibling-source alignment)

Sync specs govern topics covered in BOTH `docs/` and `cli/skill.md`. They
enumerate required coverage; markers in both files link to the same sync spec.
Changing the enumeration drifts both.

**What sync specs catch**: coverage divergence. If a new closure property is
added to the sync spec's enumeration, both the docs/ marker and the skill.md
marker drift (because the spec they're linked to changed). The reviewer must
update both.

**What sync specs do NOT catch**: wording divergence. If someone edits
skill.md's closure section without editing docs/04-closures.md, only
skill.md's marker drifts. The sync spec is unchanged. To catch wording
divergence, run the periodic spec audit (see docs/08-spec-audit.md).

**Sync spec template**:

```xml
<spec id="sync_closures">
Overview: The closure concept is covered in both docs/04-closures.md
(human-canonical, progressive) and cli/skill.md's "Closure derivation"
and "Closure properties" sections (agent-canonical, terse). Both MUST
agree on the algorithmic contract; voice differs.

Requirements:
  R1. docs.chapter_04_closures (marker: docclos in docs/04-closures.md)
      and skill.md's closure sections (marker: skillclos) MUST agree on:
      closure identity formula, per-seed derivation, strict-disjoint
      property, marker asymmetry, ephemeral nature.
  R2. Adding a new closure property to either doc requires updating the
      other; the sync spec drifts otherwise.
  R3. The closure enumeration itself lives in
      <ref spec="core.provenance_closure">core.provenance_closure</ref>;
      both docs link to it as canonical algorithmic source.

Markers governed: docclos, skillclos.
</spec>
```

| Sync spec ID | docs/ marker | skill.md marker |
|---|---|---|
| `docs.sync_cli_reference` | `docworkflow` in 05-workflow.md | `skillcli` (NEW in skill.md) |
| `docs.sync_closures` | `docclos` in 04-closures.md | `skillclos` (NEW) |
| `docs.sync_events` | `docevt` in 04-closures.md | `skillevt` (NEW) |
| `docs.sync_marker_placement` | `docmp` in 03-specs-and-markers.md | `skillmp` (EXISTS) |
| `docs.sync_drift_directory` | `docdir` in 09-internals.md | `skilldir` (NEW) |
| `docs.sync_output_modes` | `docout` in 07-output-modes.md | `skillout` (NEW) |
| `docs.sync_friction` | `docfric` in 06-friction.md | `skillfric` (NEW) |
| `docs.sync_spec_audit` | `docaudit` in 08-spec-audit.md | `skillaudit` (EXISTS) |

**Total specs in docs.drift.xml: 1 + 10 + 10 + 8 = 29.**

Plus 2 migration specs added in Phase 6: `docs.contributors_agent_guide`
(governs the migrated AGENTS.md content) and `docs.spec_format` (governs the
migrated SPECIFICATIONS.md content). Total: **31 specs**.

### Exercise marker linkage

Each chapter has TWO markers: one wrapping the main content, one wrapping
the "Try it yourself" exercise. BOTH link to the SAME chapter spec. The
chapter spec's content requirements include the exercise (R4 in
`docs.pedagogy`: "MUST include ≥1 formative assessment... tracked via a
dedicated marker"). If the exercise is removed, the chapter spec still
requires it — drift surfaces on the spec, prompting the reviewer to either
restore the exercise or update the spec.

Example for chapter 04:
- Marker `docclos` wraps the main chapter content → linked to
  `docs.chapter_04_closures`
- Marker `docclos_ex` wraps the "Try it yourself" exercise → also linked to
  `docs.chapter_04_closures`

---

## 6. Chapter content plan

Each chapter is a single `.md` file in `docs/`. Length target: 80-120 lines
of prose. Voice: terse progressive. Each ends with at least one "Try it
yourself" exercise.

### docs/01-why-drift-exists.md (~80 lines)
**Problem statement.** The reader learns what spec drift is and why
LLM-assisted codebases make it acute. No commands. No install. Pure
motivation.

Learning objectives: the reader can describe the problem of spec drift;
explain why human review fails at agent code volume; recognize the category
drift occupies (spec-code sync).

Content: concrete scenario (agent ships code violating a project rule);
why traditional review practices fail; the drift mechanism at high level
(specs as XML, markers wrap code, closures for review); forward link to
ch.02.

Exercise: none (this is a motivation chapter — no commands to try). The
meta-spec's exercise requirement is satisfied by a "thought exercise":
"identify a recent PR in your project where an agent violated an implicit
rule. What spec would have caught it?"

### docs/02-getting-started.md (~120 lines)
**Install + first run.** Ends with a green `drift todo`.

Learning objectives: the reader can install drift on macOS/Linux/Windows;
initialize a project; write a first spec; place a first marker; link them;
run `drift todo` and interpret a clean result.

Content: install (curl|bash on Unix, irm|iex on Windows — copy from
README); `drift init`; writing `main.drift.xml`; placing a `// D! id=...
range-start` / `range-end` marker; `drift link <marker> <module.spec>`;
`drift todo` clean output.

Exercise: end-to-end — install, init, write spec, place marker, link, todo
green.

### docs/03-specs-and-markers.md (~100 lines)
**Core concepts, worked example.** Deep dive on spec format and marker
syntax.

Learning objectives: the reader can write a spec with RFC 2119 keywords;
place a marker with correct syntax; explain the marker blanking rule for
nested ranges; link a marker to a spec; explain what hashing means for
specs (refs stripped) vs markers (lines between range-start/end).

Content: spec XML format (`<spec id="localid">` under `<module name="...">`);
RFC 2119 keywords (MUST/SHOULD/MAY); marker syntax; nested ranges and
blanking; `drift link`; canonical hashing (refs stripped from specs, lines
between markers for code).

Exercise: write two specs that cite each other via `<ref>`; place markers;
link both; drift one marker; observe the closure contains both.

### docs/04-closures.md (~120 lines)
**How drift detects changes.** The conceptual heart.

Learning objectives: the reader can define a closure; describe the six
event types; explain per-seed derivation; read a closure hash; trace citer
chain propagation; identify the marker asymmetry.

Content: closure = seed + transitive citers; six events
(NODE_CHANGED/ADDED/REMOVED, EDGE_ADDED/REMOVED/BROKEN) with triggers;
closure identity (SHA1 of sorted nodes + sorted undirected edges, first 8
hex); strict-disjoint across seeds; marker asymmetry (markers can cite,
cannot be cited); worked example with 2 specs, 1 marker, 1 ref.

Exercise: create a closure by drifting a marker; run `drift todo`; identify
the closure hash; run `drift diff <hash>`; verify the diff shows the
changed marker.

### docs/05-workflow.md (~100 lines)
**Daily workflow.** `todo → diff → reset`.

Learning objectives: the reader can run the daily drift workflow; interpret
`drift todo` output; review a closure via `drift diff`; resolve a closure
via `drift reset`; explain exit codes.

Content: the three-command workflow; reading `drift todo` output (closures,
unlinked markers, exit codes 0/1/2); `drift diff <hash>` for one closure;
`drift diff --all` for all closures; `drift reset <hash>` to resolve;
exit-code table; `--dry-run` for reset/link/unlink (exit 3).

Exercise: drift a spec; run todo; diff the closure; reset it; verify todo
is clean.

### docs/06-friction.md (~80 lines)
**Why one-at-a-time.** The friction principle and the rate limit.

Learning objectives: the reader can explain why drift has no bulk reset;
describe the rate-limit block (3 in 30s); identify when
`--dangerously-override-friction` is legitimate; interpret the JSON
`warning` field.

Content: the friction principle (per-closure review is the point); no
`--all` flag; rate-limit block (exit 2 on 4th reset in 30s); override flag
(legitimate for tests/CI; not advertised in error messages); JSON
`warning` field on override; `--dangerously-override-friction` is
documented in skill.md, help.txt, and command Usage but NOT suggested in
runtime error output.

Exercise: trigger the rate limit by rapidly resetting 4 closures in a test
workspace; observe the block; verify the message does NOT mention the
override flag.

### docs/07-output-modes.md (~80 lines)
**Plain/Color/JSON.**

Learning objectives: the reader can select an output mode via flags/env;
explain TTY detection; explain NO_COLOR; describe the JSON contract for
programmatic consumption; configure a theme.

Content: three modes (Plain default when piped, Color default in TTY, JSON
via `--json`); global flags (`--json`, `--no-color`, `--color=auto|always|never`);
NO_COLOR env var; TTY detection (stdlib-only, no `golang.org/x/term`);
JSON shapes per command (deterministic struct-defined field order);
themes (12 built-in, `drift config theme`, `.drift/theme.xml` custom,
`.drift/user-settings.xml` per-user).

Exercise: run `drift todo` with `--json` and parse the output with `jq`;
verify the `closures` array structure.

### docs/08-spec-audit.md (~80 lines)
**Periodic semantic audit.** Catches drift that hash-based detection
cannot.

Learning objectives: the reader can explain why `drift todo` is necessary
but not sufficient; run a spec audit (serial or parallel); triage findings
by fix direction.

Content: hash drift vs semantic drift; the three-phase audit pattern
(build index → audit → synthesize); Phase 0 commands (`drift list --json`,
`drift show --no-content --json`); the rubric (RFC 2119 keywords,
file:line evidence, diagnose only); expected yield (~80% aligned, ~12%
misaligned, ~4% violated, ~3% unclear); fix-direction triage (code vs spec
vs marker placement).

Exercise: run a 5-spec audit slice on your own project; report findings in
the structured JSON format.

### docs/09-internals.md (~100 lines)
**Under the hood.** state.xml, baselines.bin, .drift/ layout, dogfooding.

Learning objectives: the reader can describe the `.drift/` directory
layout; explain state.xml v4; explain baselines.bin; explain the
fileio.Session lock; describe the dogfooding principle.

Content: `.drift/` layout (state.xml committed, baselines.bin committed,
theme.xml committed, user-settings.xml gitignored, state.lock gitignored,
friction.json gitignored); state.xml v4 (baseline only, no resolutions
table); baselines.bin (gob-encoded packfile); `fileio.Begin` lock
(flock/LockFileEx for the entire CLI invocation); dogfooding
(`make build` runs `drift todo` as a gate).

Exercise: inspect `.drift/state.xml` after a `drift link`; identify the
new edge entry.

### docs/10-contributing.md (~80 lines)
**Contributing to drift.** Repo layout, build/test, editing tracked code.

Learning objectives: the reader can navigate the repo layout; run `make
build` and interpret the drift gate; edit drift-tracked code without
breaking the gate; submit a PR.

Content: repo layout (cmd/drift, cli/, core/, scanner/, statestore/,
orchestrator/, internal/, business/); `make build` (runs `drift todo`
gate); `go test -race -count=1 ./...`; editing tracked code (run todo →
diff → decide code/spec/citation → reset); PR checklist.

Exercise: clone the repo; run `make build`; verify the gate is green.

### docs/index.md (~30 lines)
TOC + reading-order guidance. "First time? Start with 01." Lists all 10
chapters with one-line descriptions. Links to `docs/spec-format.md` and
`docs/contributors/agent-guide.md`.

### docs/spec-format.md (~140 lines)
Migrated from `SPECIFICATIONS.md`. RFC 2119 spec authoring guide. Covers:
format, capitalized keywords, requirement numbering, defined terms, marker
placement, intent vs implementation specs, conceptual specs, spec ID
conventions, when to add a new spec, reviewing drift.

### docs/contributors/agent-guide.md (~160 lines)
Migrated from `AGENTS.md`. Full agent-contributor guide. Covers: spec
discipline workflow, critical rules, build/test/lint, repo layout, specs in
this repo, editing tracked code, adding new specs, citing other specs,
closure properties, eval harness, output modes, themes, quick reference.

### CONTRIBUTING.md (~30 lines)
GitHub convention. Fork → branch → `make build` → PR checklist → spec
discipline reminder → pointer to `docs/10-contributing.md`.

---

## 7. Migration plan

### Files to DELETE
- `DOCUMENTATION.md` — content migrated into docs/ chapters.
- `PLAN.md` — this file replaces it. (Wait — this IS PLAN.md. After
  execution completes, PLAN.md remains as the documentation plan
  reference. Do not delete it.)

### Files to MOVE
- `AGENTS.md` content → `docs/contributors/agent-guide.md` (full content).
- `SPECIFICATIONS.md` content → `docs/spec-format.md` (lightly refreshed).

### Files to SLIM
- `README.md` — keep pitch + install + 3-line quickstart. Replace
  "Development principles" and "Anatomy" sections with links to `docs/`.
  Add at top: "Full docs: `docs/index.md` · In-CLI guide: `drift skill`".
- `AGENTS.md` — reduce to ~30 lines: title + "Spec discipline workflow
  (MUST follow)" summary (steps 1-4 + "NEVER batch-reset") + pointer to
  `docs/contributors/agent-guide.md` and `cli/skill.md`.

### Files UNCHANGED
- `cli/skill.md` — content unchanged. Six new sync markers added (see §5
  sync table) but no prose changes.
- `cli/help.txt` — unchanged.

### Files NEW
- `docs.drift.xml` — new spec module (29 specs).
- `docs/index.md` — TOC.
- `docs/01-why-drift-exists.md` through `docs/10-contributing.md` — 10
  chapters.
- `docs/contributors/agent-guide.md` — migrated AGENTS.md.
- `docs/spec-format.md` — migrated SPECIFICATIONS.md.
- `CONTRIBUTING.md` — GitHub convention.
- ~30 new markers across docs/ files and skill.md.

---

## 8. Execution phases

Execute in order. Each phase ends with `drift todo` clean (all new markers
linked, all closures resolved). Use `--dangerously-override-friction` when
resolving more than 3 closures in 30 seconds (legitimate — the author is
reviewing each closure as it is created).

### Phase 1 — docs.drift.xml skeleton + meta-spec (~1 hour)
1. Create `docs.drift.xml` with `<module name="docs">`.
2. Write `docs.pedagogy` meta-spec in full (R1-R7 from §5).
3. Stub all 29 specs (spec id + Overview + TODO body).
4. `drift todo` — reports 29 unlinked specs (no markers yet). Expected.

### Phase 2 — Foundational chapters 01-03 + concept specs (~2 hours)
1. Author concept specs: `concept_spec`, `concept_marker`, `concept_edge`,
   `concept_baseline`, `concept_scan` (full body, ~10-15 lines each).
2. Author chapter specs: `chapter_01_why_drift_exists`,
   `chapter_02_getting_started`, `chapter_03_specs_and_markers` (full body
   per template in §5).
3. Author the three `.md` chapter files (terse-progressive voice,
   exercises included, ~80-120 lines each).
4. Place markers in each .md file: `docwhy` + `docwhy_ex`, `docstart` +
   `docstart_ex`, `docsm` + `docsm_ex`. Marker syntax in markdown:
   `<!-- D! id=<id> range-start -->` / `<!-- D! id=<id> range-end -->`.
5. `drift link` each marker to its spec.
6. `drift todo` — resolve closures (`drift reset` or
   `drift reset --dangerously-override-friction` if >3).

### Phase 3 — Core chapters 04-06 + remaining core concepts (~2 hours)
1. Author concept specs: `concept_closure`, `concept_drift_event`,
   `concept_citer_chain`, `concept_friction`.
2. Author chapter specs: `chapter_04_closures`, `chapter_05_workflow`,
   `chapter_06_friction`.
3. Author the three .md files with exercises.
4. Place markers: `docclos` + `docclos_ex`, `docflow` + `docflow_ex`,
   `docfric` + `docfric_ex`.
5. Link, resolve closures.

### Phase 4 — Application chapters 07-10 + final concept (~2 hours)
1. Author concept spec: `concept_output_mode`.
2. Author chapter specs: `chapter_07_output_modes`,
   `chapter_08_spec_audit`, `chapter_09_internals`,
   `chapter_10_contributing`.
3. Author the four .md files with exercises.
4. Place markers: `docout` + `docout_ex`, `docaudit` + `docaudit_ex`,
   `docint` + `docint_ex`, `doccont` + `doccont_ex`.
5. Link, resolve closures.

### Phase 5 — Sync specs + skill.md sync markers (~1 hour)
1. Author 8 sync specs in `docs.drift.xml` (per §5 sync table).
2. Add 6 new sync markers to `cli/skill.md` wrapping the corresponding
   sections: `skillcli`, `skillclos`, `skillevt`, `skilldir`, `skillout`,
   `skillfric`. Use HTML comment marker syntax.
3. Link sync markers to sync specs. Note: each sync spec is linked from
   BOTH the docs/ marker and the skill.md marker.
4. Resolve closures.

### Phase 6 — Migrations + README/AGENTS slim + CONTRIBUTING (~1 hour)
1. Copy `AGENTS.md` → `docs/contributors/agent-guide.md` (full content).
   Place marker `docagent` wrapping the content. Link to a new
   `docs.contributors_agent_guide` spec (add to docs.drift.xml).
2. Move `SPECIFICATIONS.md` → `docs/spec-format.md`. Place marker
   `docspec`. Link to a new `docs.spec_format` spec.
3. Slim root `AGENTS.md` to ~30 lines: title + MUST-FOLLOW workflow
   summary + pointer to `docs/contributors/agent-guide.md`.
4. Slim `README.md`: keep pitch + install + 3-line quickstart. Replace
   detailed sections with links to `docs/`.
5. Write `CONTRIBUTING.md` (~30 lines).
6. Write `docs/index.md` (TOC).
7. Delete `DOCUMENTATION.md`.
8. Resolve closures.

### Phase 7 — Verify + close drift closures (~30 min)
1. `make build` — drift gate must pass.
2. `go test -race -count=1 ./...` — all tests pass.
3. `drift todo` — clean (exit 0).
4. Manual link check: open `docs/index.md`, follow links to every chapter.
5. Verify no broken markdown links (`grep -r '\[.*\](\.' docs/ README.md
   AGENTS.md CONTRIBUTING.md`).

---

## 9. Drift workflow conventions

### Marker syntax in markdown files

Markdown files use HTML comment markers:

<!-- D! instruction=ignore-span-start -->
```markdown
<!-- D! id=docwhy range-start -->

# Why drift exists

... chapter content ...

<!-- D! id=docwhy range-end -->
```
<!-- D! instruction=ignore-span-end -->

The scanner blank inner-marker declarations before hashing, so nested
markers (e.g. an exercise marker inside a chapter marker) work correctly.

### Using the friction override

When executing this plan, you will create many closures (each new spec +
each new marker + each migration). Resolving them one-at-a-time at 30s
intervals is impractical. Use `--dangerously-override-friction`:

```sh
drift reset --dangerously-override-friction <hash>
```

This is the legitimate use case for the override: the author is reviewing
each closure as it is created. The override emits a stderr squawk and
populates the JSON `warning` field; the reset proceeds normally and
records the timestamp.

### Drift discipline during execution

Follow the spec discipline workflow at all times:
1. `drift todo` — see what drifted.
2. `drift diff <hash>` — review the closure.
3. Decide: code wrong / spec wrong / citation wrong.
4. `drift reset <hash>` — resolve ONE closure at a time.

For batches of self-authored closures, the override is acceptable. For
closures discovered in pre-existing code, review each carefully before
resetting.

---

## 10. Style and conventions

### RFC 2119 keywords

In specs: MUST, MUST NOT, SHOULD, SHOULD NOT, MAY — capitalized. Reserve
for actual normative requirements.

In docs prose: use sparingly. "The scanner MUST reject duplicate spec IDs"
is normative. "You should run drift todo daily" is not — prefer "Run
`drift todo` to check for drift."

### No emoji

No emoji in docs, specs, or code comments. Exceptions: none.

### No advertising the override flag

The `--dangerously-override-friction` flag is documented in `skill.md`,
`help.txt`, and the reset command's Usage text. It MUST NOT be suggested
in error messages or runtime warnings. Advertising increases the likelihood
of LLM agents adopting bypass behavior.

The block message when the rate limit fires:
> "rate limit: 3 closures resolved in the last 30s. Drift's friction
> principle expects per-closure review — the intended workflow is `drift
> todo` → `drift diff --all` → `drift reset <hash>` one closure at a time,
> with each closure reviewed before it is reset."

Note: the override flag is NOT named in this message.

### No "understand" in learning objectives

Learning objectives use measurable Bloom verbs (identify, describe, explain,
apply, analyze, troubleshoot, design). "Understand," "know," and "learn"
are forbidden — they are unmeasurable.

### Section structure for chapter .md files

```markdown
# <Chapter title>

<One-paragraph orientation: what this chapter covers, what the reader
will be able to do after, what prerequisites are assumed.>

## <Section>

<Content>

## Try it yourself

<Exercise with goal + verification>
```

---

## 11. Verification — what done looks like

- `docs/` folder exists with 10 numbered chapters + index.md + spec-format.md
  + contributors/agent-guide.md.
- `docs.drift.xml` exists with 31 specs (1 meta + 10 concept + 10 chapter
  + 8 sync + 2 migration).
- ~30 new markers placed across docs/ files and skill.md.
- `cli/skill.md` has 6 new sync markers (content otherwise unchanged; the
  file already has 4 existing markers: `skillmp`, `skilldt`, `skillaudit`,
  `auth_login` — all linked to existing `cli.*` specs).
- `README.md` slimmed (pitch + install + 3-line quickstart + links to docs/).
- `AGENTS.md` slimmed to ~30-line stub.
- `CONTRIBUTING.md` exists at root.
- `DOCUMENTATION.md` deleted.
- `make build` passes (drift gate green).
- `go test -race -count=1 ./...` passes.
- `drift todo` reports clean (exit 0).
- All markdown links resolve.

Final counts (approximate): 188 specs (157 + 31), 126 markers (96 + 30),
194 edges (184 + ~10 sync + migration edges).

---

## 11a. Writing process — how to author a chapter

For each chapter, write in this order:

1. **Spec first.** Author the chapter spec in `docs.drift.xml` with learning
   objectives, prerequisites (via `<ref>`), and content requirements. This is
   the contract the chapter must satisfy.
2. **Verify the prereq graph.** Run `drift show docs.chapter_04_closures`
   after linking to confirm the closure includes the expected concept specs
   and earlier chapters. If the closure is empty or wrong, the `<ref>` tags
   are misconfigured.
3. **Write the prose.** Author the `.md` file in terse-progressive voice.
   Start with the orientation paragraph. Then sections. Then the exercise.
4. **Place markers.** Wrap the main content with the chapter marker
   (`docclos`) and the exercise with the exercise marker (`docclos_ex`).
   Link both to the chapter spec via `drift link`.
5. **Verify the exercise.** Mentally (or actually) run through the exercise.
   Confirm the verification step produces the stated outcome. If the
   exercise references `drift todo` exit codes or output formats, confirm
   they match current behavior.
6. **Resolve drift.** Run `drift todo`. The new spec + markers produce
   closures (NODE_ADDED, EDGE_ADDED). Review via `drift diff <hash>`. Reset.

### Cycle warning

The pedagogical prereq graph MUST be acyclic. Drift's `core.validate`
rejects directed cycles among spec-spec edges. If chapter A cites chapter B
as a prerequisite and chapter B cites chapter A, `drift todo` will report
an error and the gate will fail.

To avoid cycles: prerequisites always point backward in reading order.
Chapter N may cite chapters 1 through N-1 and any concept spec. Concept
specs may cite other concept specs or canonical authority specs
(`glossary.*`, `core.*`) but NOT chapter specs (concepts are atomic;
chapters are delivery vehicles).

### Marker naming convention

| Prefix/suffix | Meaning | Example |
|---|---|---|
| `doc*` | Marker in a `docs/*.md` file | `docwhy`, `docclos`, `docfric` |
| `doc*_ex` | Exercise marker within a docs chapter | `docclos_ex`, `docfric_ex` |
| `skill*` | Sync marker in `cli/skill.md` | `skillclos`, `skillout` |
| `docagent` | Marker in `docs/contributors/agent-guide.md` | (single) |
| `docspec` | Marker in `docs/spec-format.md` | (single) |
| `docidx` | Marker in `docs/index.md` | (single) |

### Commit strategy

One commit per phase. Commit message format:

```
Add docs phase N: <short description>

<body describing what was added/changed in this phase>
```

After each commit, `make build` must pass. If the drift gate fails, fix
the issue before committing — do not commit a broken gate.

### Chapter length overflow

If a chapter exceeds 120 lines, consider splitting it. Each chapter should
cover one cohesive concept cluster. If two distinct concepts are fighting
for space, they may warrant separate chapters (renumber subsequent
chapters). If a single concept is simply dense (e.g. closures), 150 lines
is acceptable — the 120-line target is a guideline, not a hard limit.

### Using `drift show` to verify the graph

After linking markers to specs, run:

```sh
drift show docs.chapter_04_closures --no-content
```

This shows the full citation closure: the chapter spec, its concept prereqs,
its earlier-chapter prereqs, and any sync specs that cite it. Verify the
graph matches the intended pedagogical structure. An empty or wrong closure
indicates misconfigured `<ref>` tags.

---

## 12. Risks and mitigations

| Risk | Mitigation |
|---|---|
| Voice drifts conversational mid-chapter | Re-read §2 voice examples before each chapter. Write a sample paragraph, check against the rules, then continue. |
| Learning objectives use "understand" | Run `grep -i 'understand\|know\|learn' docs/*.md` after each phase. Replace with measurable Bloom verbs. |
| Sync markers in skill.md break its embedding | skill.md is embedded via `go:embed` in `cli/cli.go`. HTML comment markers are valid markdown and won't affect rendering. Verify `drift skill` output still reads cleanly after adding markers. |
| Forgetting to use `--dangerously-override-friction` | When `drift reset` returns exit 2 with the friction message, switch to the override flag. The message intentionally does NOT remind you. |
| Breaking the drift gate mid-execution | Each phase ends with `drift todo` clean. Do not proceed to the next phase with unresolved closures. |
| Duplicate content between docs/ and skill.md | Sync specs (§5 Tier 4) govern shared topics. If a topic is in both, it MUST have a sync spec. If a topic is only in docs/, no sync spec needed. |
| Markdown link rot | Phase 7 includes a link check. Run `grep -r '\[.*\](\.' docs/ README.md AGENTS.md CONTRIBUTING.md` and verify each link resolves. |
| Exercise outcomes drift as drift evolves | Exercises should reference stable behaviors (drift todo exit codes, marker syntax) not implementation details. Avoid line-number references in exercises. |

---

## 13. Open notes

- The `docs.pedagogy` meta-spec is the authority for how to write chapter
  specs. When in doubt, re-read it.
- Concept specs cite canonical authority (e.g. `concept_closure` cites
  `glossary.closure` and `core.provenance_closure`). This grounds the
  pedagogical concept in the formal contract.
- Sync specs are the mechanism that prevents skill.md and docs/ from
  drifting apart semantically. They don't catch wording divergence (that
  requires periodic audit — see docs/08-spec-audit.md), but they catch
  coverage divergence (adding/removing topics).
- The "terse progressive" voice is a synthesis of Rust Book (progressive
  structure) and man pages (terse, declarative, third-person). When in
  doubt, write less. Each sentence should convey one fact.
- This plan is itself drift-tracked: after execution, `PLAN.md` remains in
  the repo as the documentation plan reference. Future doc work should
  update this file (or supersede it with a new plan).

---

## 14. Execution strategy — incremental spec creation

The original Phase 1 (§8) called for stubbing all 29 specs at once, then
fleshing them out in Phases 2-4. This creates double work: each stub gets
baselined, then re-drifts (NODE_CHANGED) when overwritten with real content,
requiring a second reset per spec.

**Revised approach**: create specs in full, incrementally, within their
natural phase. No stubs. Each phase creates its specs with final content,
places markers, links, and resolves drift — once.

### Revised phase schedule

| Phase | Specs created (in full) | Markers placed | Output |
|---|---|---|---|
| **1** | `docs.pedagogy` (meta-spec) + module skeleton + import in main.drift.xml | 0 | docs.drift.xml exists, module imported |
| **2** | 5 concept specs (spec, marker, edge, baseline, scan) + 3 chapter specs (01-03) | 6 (docwhy±_ex, docstart±_ex, docsm±_ex) | docs/01-03 + concept tier started |
| **3** | 4 concept specs (closure, drift_event, citer_chain, friction) + 3 chapter specs (04-06) | 6 (docclos±_ex, docflow±_ex, docfric±_ex) | docs/04-06 |
| **4** | 1 concept spec (output_mode) + 4 chapter specs (07-10) | 8 (docout±_ex, docaudit±_ex, docint±_ex, doccont±_ex) | docs/07-10 |
| **5** | 8 sync specs | 6 new skill.md sync markers + 2 re-links | sync tier complete |
| **6** | 2 migration specs (contributors_agent_guide, spec_format) | ~5 (docagent, docspec, docidx) | migrations + README/AGENTS slim + CONTRIBUTING |
| **7** | 0 | 0 | verify: make build, go test, drift todo clean |

**Totals**: 1 + 8 + 7 + 5 + 8 + 2 = 31 specs; ~31 markers.

### Phase 1 status (complete)

- [x] `<import path="./docs.drift.xml" />` added to `main.drift.xml`.
- [x] `docs.drift.xml` created with `<module name="docs">`.
- [x] `docs.pedagogy` meta-spec authored (R1-R7).
- [x] Closure `e1fdd412` reviewed and reset (NODE_ADDED + 2 EDGE_ADDED for glossary.spec, glossary.ref refs).
- [x] `drift todo` clean: 159 specs, 98 markers, 189 edges.

### What each subsequent phase does

Each phase follows the same loop:
1. Author specs in `docs.drift.xml` (full content, not stubs).
2. Author the `.md` chapter files in terse-progressive voice with exercises.
3. Place markers in the `.md` files (HTML comment syntax).
4. `drift link` each marker to its spec.
5. `drift todo` → review closures → `drift reset` (use `--dangerously-override-friction` for batches >3).
6. `make build` to verify the gate.

**Review checkpoint**: pause at end of each phase for review before proceeding.

---

## 15. Revised chapter structure (task-oriented)

The original chapter structure (§6) was concept-oriented: "Specs and
markers", "Closures", "Friction", etc. After discussion, the structure
was revised to be **task-oriented** — each chapter is a real task a
developer performs, phrased in natural language.

### Core framing

**Drift is a sync layer between specs and code. The agent writes both.
Drift keeps them aligned.** Specs are compressed intent. Code is the
expansion. The human prompts; the agent writes specs and code; drift
keeps them in sync. This makes the agent more accurate (checks its own
work against specs) and more agile (moves fast, knowing drift catches
misalignment).

### Final chapter list (15 chapters + 3 appendices)

```
01. Why drift exists
02. Getting started
03. How to add drift to a project
04. How to add a feature
05. How to fix a bug
06. How to refactor a module
07. How to change a spec
08. How to remove a feature
09. How to explore the codebase
10. How to configure drift
11. How to set up CI with drift
12. How to commit drift to git
13. How to audit your specs
14. How to troubleshoot drift
15. How to get more information about drift commands

Appendices:
A. Output modes
B. Internals
C. Spec-format reference
```

### Structural decisions

- **No RFC 2119 in docs prose.** RFC 2119 keywords are an internal
  convention for writing specs (in `*.drift.xml`), not something to push
  on users in documentation.
- **No "Contributing" chapter.** Deferred.
- **No separate Reference section.** Friction and closures fold into
  the task chapters that use them ("How to troubleshoot drift" covers
  friction; "How to fix a bug" covers closures).
- **Agent is the primary operator.** The docs frame drift as a tool for
  LLMs. The human prompts; the agent writes specs and code. The docs
  teach the human what drift is and how to steer through specs.

### Revised execution phases

| Phase | Scope | Chapters |
|---|---|---|
| **0** | Foundation: restructure meta-spec, concepts, ch.01-03 | 01-03 |
| **1** | Daily-work tasks | 04-08 |
| **2** | Operations + recovery | 09-15 |
| **3** | Appendices + migrations | A, B, C + README/AGENTS slim |
| **4** | Sync specs + skill.md markers | sync specs |
| **5** | Final verification | make build, go test, drift todo clean |

### Phase 0 status (complete)

- [x] docs.pedagogy updated (R3 dropped RFC 2119, R5 task-driven spiral,
      R6 stale ref fixed, overview sync-layer framing)
- [x] 4 new concept specs added (closure, drift_event, citer_chain,
      friction)
- [x] ch.01 rewritten (sync-layer framing)
- [x] ch.02 rewritten (install + agent handoff)
- [x] ch.03 rewritten (How to add drift to a project; replaces old
      specs-and-markers chapter)
- [x] Old markers docsm/docsm_ex unlinked and removed
- [x] New markers docaddp/docaddp_ex linked to chapter_03
- [x] All closures resolved
- [x] `drift todo` clean: 171 specs, 104 markers, 217 edges
- [x] `make build` passes
