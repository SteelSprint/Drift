# Plan — Phase 6 (Updated)

> All decisions in this document are LOCKED. Implementation proceeds against this plan.
> Source: Observation 0006 — Phase 5 full battery (6 runs).

## Architecture (unchanged)

```
┌──────────────────────────────────────────────────────────────┐
│  CLI (cli.go)                                                 │
│  drift init                                                   │
│  drift todo                                                    │
│  drift list [--verbose]                                       │
│  drift link <marker> <module.spec>                            │
│  drift unlink <marker> <module.spec>                          │
│  drift reset <marker> <module.spec>                           │
│  drift reset <id>           (orphan cleanup)                  │
│  drift show <marker|spec>    (NEW)                             │
│  drift help / drift skill                                     │
├──────────────────────────────────────────────────────────────┤
│  Orchestrator                                                 │
│  load pin → scan → reconcile → build ctx → core              │
│  → save (reset/link/unlink only)                              │
├─────────────────────────┬────────────────────────────────────┤
│  PinStore               │  Scanner                           │
│  read/write drift.pin   │  follow main.pin.xml imports →     │
│  (XML codec)            │  discover specs (module-qualified)  │
│                         │  validate spec/marker ID format     │
│                         │  walk dir tree → discover markers   │
│                         │  range-start/range-end pairing      │
│                         │  hash content → produce ScanResult  │
├─────────────────────────┴────────────────────────────────────┤
│  Core (core.go)                                               │
│  pure, stateless                                            │
│  EvaluateState(ctx) → EvaluatedState                        │
│  - drift detection (including deletion = drift)              │
│  - collapse (prune deleted nodes after resolution)            │
└──────────────────────────────────────────────────────────────┘
```

## D1: Marker hash model rework ✅ LOCKED

### Syntax

Every marker is an explicit range. No other marker types.

```
// D! id=foo range-start    (marks the start of the tracked region)
// D! id=foo range-end      (marks the end of the tracked region)
```

### Rules

- Every `range-start` with ID `X` must have a matching `range-end` with ID `X` **in the same file**, appearing **after** the start.
- Every `range-end` with ID `X` must have a matching `range-start` with ID `X` **in the same file**, appearing **before** the end.
- Unpaired start or end → **error**. Scanner reports **all** unpaired markers at once (not fail-on-first), with clear messages:
  ```
  scanner error: main.go:15: marker "foo" has range-start but no matching range-end in the same file
  scanner error: main.go:42: marker "bar" has range-end but no matching range-start in the same file
  ```
- Old-style `// D! id=foo` (without `range-start` or `range-end` suffix) → **error**. Forces switchover.
- Marker shortcodes still must not contain dots (existing rule).
- Duplicate marker IDs still error (existing rule).
- Nested ranges: **allowed**.
- Overlapping ranges (non-nested): **allowed**.

### Hashing

- For marker `foo` with `range-start` at line S and `range-end` at line E: hash lines S+1 through E-1 (exclusive of both marker lines).
- Before hashing, **blank every line within the window that matches the marker pattern**: strip from `D!` to end of line, leaving the comment prefix (`// `, `# `, etc.). This makes markers invisible to each other's hashes. Nested and overlapping ranges work naturally.
- Hash function: SHA1 hex-encoded (unchanged).

### drift.pin storage

Store both `line` (start line) and `endline` (end line) per marker:

```xml
<marker id="cval" hash="7dc34f7516f4..." filepath="core.go" line="108" endline="124"/>
```

### Scanner implementation

Two-pass per file:
1. **Pass 1:** Find all marker declarations. Capture ID + suffix (range-start/range-end) + line number. Validate pairs (all-at-once). Validate ID format (no dots). Validate no old-style markers.
2. **Pass 2:** For each marker, compute its hash window (start+1 to end-1), blank other marker declaration lines within the window, hash the result.

### Migration

All ~42 existing `// D! id=foo` markers in the codebase must be converted to `range-start`/`range-end` pairs. This is the first implementation step — every marker wraps the function/block it was tracking.

---

## D2: Folds into D6

`drift diff` is replaced by `drift show` (see D6). No separate diff command.

---

## D3: Spec text in `drift list` ✅ LOCKED

- Default `drift list` output stays compact (current format, updated to show `main.go:9-24` for ranges).
- `drift list --verbose` adds spec text (truncated to ~80 chars) and first line of marker range content.
- **Note:** This assumes SOTA-model capability to handle verbosity. This assumption should be tested in future evals with smaller models. Decision deferred to that point.

---

## D4: Fix `line="0"` for specs ✅ LOCKED

- Drop the `:line` suffix for specs in `drift list`. Show `core.pin.xml` instead of `core.pin.xml:0`.
- For markers, show `main.go:9-24` (start-end range).
- Specs don't have meaningful line numbers — their content is what matters.

---

## D5: No linting ✅ LOCKED (rejected)

- No `drift lint` command.
- Empty ranges (start and end adjacent) are allowed (technically an error condition but not blocked).
- No language-specific keyword checks (impossible to support all languages).
- No arbitrary size limits (dangerous, would need per-language tuning).

---

## D6: `drift show` command ✅ LOCKED

One command, replacing both `drift diff` and `drift inspect`/`drift show` proposals.

### `drift show <marker|spec>`

Shows current state of a marker or spec, regardless of whether drift exists.

**Output format (spec shown first, then content):**

```
Spec: main.validate
File: core.pin.xml
Hash: afd4321ea69c...

<spec text here>

---

Marker: cval
File: core.go
Lines: 108-124
Hash: 7dc34f7516f4...

<range content here>
```

### Key properties

- Shows **current** state. No historical comparison (driftpin is not git-aware).
- Provides filepath and line ranges so the agent can run `git diff` or `git log` on those exact lines immediately after.
- Works for both `<marker>` and `<spec>` arguments.
- If the argument is a marker ID (no dot), shows the marker's range content + any linked spec text.
- If the argument is a spec ID (has dot), shows the spec text + any linked markers' range content.

---

## D7: No `--json` output ✅ LOCKED (rejected)

No `--json` flag. Not needed.

---

## D8: `drift todo` exit codes ✅ LOCKED

- `drift todo` exits **0** when clean, **1** when drift exists, **2** on error.
- No separate `drift check` or `drift verify` command.
- CI/agents use: `drift todo && echo "clean"` or check `$?`.

---

## D9: No `drift reset --all` ✅ LOCKED (rejected)

- No bulk reset. Ever.
- Driftpin is a fine-grained checking tool. Bulk actions invite abuse and error.
- LLMs are less likely to make errors when they must resolve edges one at a time.
- Every reset is a deliberate, conscious action.

---

## D10: `drift reset` confirmation ✅ LOCKED

On successful `drift reset <marker> <module.spec>`, print:

```
Resolved: <marker> → <spec>. Baseline updated.
```

(No ASCII checkmark.)

Also: update `drift skill` to clarify what "collapses baselines" means: "reset rewrites the baseline hash to the current hash and clears the resolution entry."

---

## D11: Document edge cases in `drift skill` ✅ LOCKED

Update `drift skill` with an "Edge cases" section reflecting the new directions:

- **Marker syntax:** Only `range-start`/`range-end` pairs. No old-style bare markers.
- **Unpaired markers:** Scanner errors with all offenders listed at once.
- **Nested ranges:** Allowed. Inner marker declarations are blanked from outer range hashes.
- **Overlapping ranges:** Allowed.
- **Empty ranges:** Allowed (start and end adjacent). Hashes empty string. Not blocked.
- **Deleted markers:** Treated as drift (sentinel hash `""`). Shows in `drift todo`. Resolved via `drift reset <marker> <spec>`. Pruned after resolution.
- **Deleted specs:** Same deletion-as-drift model.
- **Orphaned entries (deleted, no links):** Shown with `[deleted]` tag in `drift list`. Cleaned via `drift reset <id>`.
- **`drift reset` semantics:** Rewrites baseline hash to current hash, clears resolution entry. Prints confirmation.
- **`drift todo` exit codes:** 0 = clean, 1 = drift, 2 = error.
- **`drift show`:** Shows current spec text + marker range content + filepath + line ranges. Not git-aware.

---

## Low-priority items (deferred)

These remain in the backlog but are not part of Phase 6:

| # | Item | Disposition |
|---|---|---|
| 12 | Spec-ID vs marker-ID qualification docs | Defer |
| 13 | `--dry-run` on `drift reset` | Defer |
| 14 | Per-subcommand `--help` | Defer |
| 15 | Normalize help-flag handling | Defer |
| 16 | Detect duplicate `drift link` | Defer |
| 17 | `drift init` placement hint | Defer |
| 18 | `drift demo` / `drift init --demo` | Defer |
| 19 | `drift --version` | Defer |
| 20 | Clarify `drift.ignore` examples | Defer |
| 21 | `drift auto-link` | Defer |
| 22 | Example workflows in `drift skill` | Defer |
| 23 | `drift validate` semantic pass | Defer |
| 24 | Relativize `drift.pin` paths | Defer |

---

## Implementation order

1. **D1: Marker hash model rework** — scanner, pinstore, migration of all existing markers
2. **D6: `drift show` command** — new CLI command
3. **D3: `drift list --verbose`** — add verbose flag
4. **D4: Fix `line="0"` for specs** — drop `:line` suffix, update `drift list` format
5. **D8: `drift todo` exit codes** — non-zero on drift
6. **D10: `drift reset` confirmation** — print message
7. **D11: Document edge cases in `drift skill`** — after all above, captures new behavior
8. **Rebuild drift.pin** — after all code changes
9. **Tests + vet + gofmt** — verify
10. **Re-run evals** — file observation 0007

## Future steel cables

### Steel cable 8: Ref-based drift

Parse `<ref>` elements in spec content. Implement dual-hash model:
- Self hash: hash of spec content excluding resolved refs
- Composite hash: hash including resolved ref content
- Markers link to composite hash
- Drift output distinguishes "you changed this" vs "a dependency changed"

### Steel cable 9: AST

Replace flat prose specs with structured AST nodes. Each node hashable independently. Markers link to specific AST nodes, not whole specs.
