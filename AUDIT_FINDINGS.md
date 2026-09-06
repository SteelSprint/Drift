# Spec Audit Findings — drift repository

- Date: 2026-09-05
- Tree state: `main` at b69989a (v1.3.5 + skill design-patterns commit), `drift todo` clean, 218 specs / 163 markers / 328 edges.
- Method: parallelized spec audit (see `drift skill` → Design patterns). Closure inventory built from `drift list --json` (97 citation components). 10 read-only walkers graded all 218 specs against their marker regions (or, for prose specs, internal consistency + citers). Diagnose only — nothing was fixed.
- Two findings were pre-graded before the audit (orch.error_sentinels, orch.reconcile_specs/markers) and independently re-confirmed by walker W7.

## Verdict tally

| Verdict | Count | Meaning |
|---|---|---|
| aligned / holds | 168 | spec matches code (or prose internally consistent) |
| misaligned | 29 | spec text contradicts code in at least one requirement |
| violated | 12 | a MUST-level requirement is not met |
| unclear | 9 | spec ambiguous or claims that cannot be verified as stated |

---

# A. Contract violations (code is wrong per its own specs — fix the code)

## A1. The guardrail property is violated, and the tests that should catch it do not exist
- `output.guardrail_property` — VIOLATED. R1 (MUST): `stripANSI(ColorPresenter.X(r))` must equal `PlainPresenter.X(r)` byte-for-byte. Fails for `ChangeSummary`: Plain pads node-change kinds to width 8 (`plain.go:668` `%-8s`), Color does not (`color.go:554`) — stripped outputs differ by spaces. Same class at `plain.go:676` vs `color.go:565`.
- `output_impl.guardrail_test` — VIOLATED. The claimed spot-check tests do not exist: no `stripANSI` helper and no Color-vs-Plain comparison anywhere in the test tree (`theme_test.go:8-91` covers only `Style.Apply`).
- `output.theme_system` — R4 asserts the guardrail "holds for every theme"; false as stated because of A1.
- Fix direction: fix the padding in one presenter, add the promised strip-equality spot-check tests (they are the enforcement mechanism the spec names), then reset.

## A2. `fileio.platform_syscalls` R6: fd leak on unlock failure
- R6: "release before close; if release fails the fd MUST still be closed." Code returns early on Flock/LockFileEx error and skips `Close()` on BOTH platforms (`fileio_unix.go:22-24`, `fileio_windows.go:40-42`).
- Fix direction: two-line fix per platform (defer-close), then reset.

## A3. `.gitignore` bootstrap paths do not converge — `friction.json` can be committed
- `cli.gitignore_bootstrap` — MISALIGNED (R2, R6). The init path writes `.drift/.gitignore` with all three entries (`helpers.go:26`); the config-theme path writes only `user-settings.xml` + `state.lock`, omitting `friction.json` (`config.go:99`). A project themed before init's gitignore exists would commit `friction.json`. The spec's convergence claim (R6) and `cli.init_template` R3 both require all three.
- Fix direction: add `friction.json` to the config-path content; one line.

## A4. `pinstore.load` accepts unversioned pre-v4 files silently
- `pinstore.load` / `pinstore.xml_types` — UNCLEAR leaning misaligned. The version guard is `Version > 0 && Version < 4` (`pin_file.go:119`), so explicit v1–v3 are refused but version-0 (unversioned) pre-v4 files load. Both specs state pre-v4 files are refused.
- Fix direction: decide whether unversioned files must be refused (spec letter) or accepted (tolerant behavior), then align the smaller side.

## A5. `output.json_schema` — `drift todo --json` always emits an undocumented `orphans` field
- VIOLATED (R1). The todo shape in the spec omits `orphans`; the code always emits it (`json.go:37,93-96,104`). All other 10 command shapes match.
- Fix direction: document the field in the spec (additive, cheapest) or add `omitempty` + drop always-emit (behavior change for consumers).

---

# B. Spec-text drift (spec is wrong; code is right — fix the spec text)

## B1. NODE_CHANGED vs NODE_ADDED cluster (4 spec texts + 2 code comments)
- `orch.reconcile_specs`, `orch.reconcile_markers` — MISALIGNED. Both say empty-baseline-hash new nodes are flagged "as NODE_CHANGED on the next todo". The core derives **NODE_ADDED** for `Hash=="" && current != ""` (`core/core.go:572-577,595-600`) — which is also what `core.todo_action` correctly documents.
- The same wrong kind appears in the reconciler's inline code comments (`orchestrator.go:610-612` and the markers-region counterpart).
- Related gap: `core.reset_action` and `orch.reset_closure` enumerate the reset event set but OMIT NODE_ADDED even though the code applies it (`core/core.go:478-486`, baseline refresh at `orchestrator.go:269`).
- Fix direction: four small text corrections (2 specs, 2 comments) + add NODE_ADDED to the two reset event lists. This is exactly the stale-but-stable class the audit exists for.

## B2. Sentinel errors declared but never returned
- `orch.error_sentinels` — MISALIGNED. Claims every sentinel "maps a user-facing intent ... so the CLI can pattern-match on errors.Is". Three of nine are declared-only: `ErrLinkMarkerNotFound`, `ErrLinkSpecNotFound` (link returns ad-hoc `fmt.Errorf` with the same prose at `orchestrator.go:366,381`), and `ErrDiffNodeNotFound` (zero return sites repo-wide).
- Fix direction: either wrap the two link sentinels at their return sites (preserves the prescriptive messages, restores the errors.Is contract) or amend the spec; `ErrDiffNodeNotFound` is dead and should be deleted.

## B3. Exit-0 contract omits the orphan gate (3 specs + help text)
- `cli.format_todo`, `cli.unlinked_warning`, `cli.todo_alignment_vs_correctness` — MISALIGNED / VIOLATED. All three state exit 0 requires "(a) all markers linked and (b) no closures". The code additionally requires zero orphans (`cli/commands/todo.go:21`), and `core.reachability` documents that orphans flip exit to 1. `cli/help.txt:37` repeats the incomplete contract.
- Fix direction: add the orphan condition to the three specs and the help line.

## B4. Flag enumeration is stale
- `cli.unknown_flag_rejection` — MISALIGNED (R3). "Any other `--foo` is rejected" contradicts accepted flags `--no-content` (show) and `--dangerously-override-friction` (reset), both named in sibling specs (`cli.show_command`, `cli.reset_friction_block` R8).
- Fix direction: update the recognized-flag enumeration.

## B5. Stale migration/history claims in output-layer specs
- `output_impl.plain_presenter` — MISALIGNED: "byte-identical continuation ... format* functions migrated verbatim" — the closure pivot rewrote `Todo/List/Show/Diff`; only `unlinkedMarkerWarning`/`formatDiffSide` survive by name.
- `output.plain_mode` — VIOLATED (R2): same naming claim; three of five named functions no longer exist.
- `output_impl.presenter_interface` — MISALIGNED: enumerates 9 interface methods; the interface has 10 (`ChangeSummary`, `presenter.go:15`) — contradicts `output_impl.result_types`, which is correct.
- `output_impl.command_interface_impl` — MISALIGNED (R3): Context field list omits `Sess *fileio.Session`.
- `cli.help` — UNCLEAR (R1): "via the help generator" — no generator exists; content is embedded `help.txt` (the cited spec acknowledges embedded is current).
- `output.result_types` — R3 misattributes the `(string,int)` return to the wrong layer; contradicts `output.dispatch_pipeline` step 8.
- `cli.legacy_run_entries` — MISALIGNED: `Run`/`RunAuto` have zero callers (dead code); spec claims main.go calls RunAuto.
- Fix direction: rewrite the history sentences to state current structure; decide whether Run/RunAuto stay (then spec says "reserved for tests") or are deleted.

## B6. Guide-content specs
- `cli.skill` — MISALIGNED (R4, R10): "module/import system" — no import concept exists in the tool (module-qualification is what's covered); the `.drift/` layout list omits `.gitignore` (which init creates).
- `cli.skill` R16 + `output_intent.metadata_single_source` — UNCLEAR: intent-level claims that only partially hold (main help still embedded; eval-harness claim lives outside the marker).
- Fix direction: small text amendments.

## B7. Glossary and contract prose contradicts its own cited authority
- `glossary.closure` — VIOLATED: "exactly one seed" vs same-hash merge carrying multiple seeds (`core/core.go:747-751`); also omits marker-seed linked-spec expansion.
- `docs.concept_closure` — VIOLATED: same "one seed node" defect.
- `glossary.baseline` — VIOLATED: "the set ... at the time of the most recent drift reset" — reset is per-seed, and link/unlink also write baseline; the per-node sentence in the same entry is correct.
- `glossary.spec` — VIOLATED: depicts `<spec id="module.localid">`; the id attribute is local-only (`scanner.spec_id_format`).
- `docs.concept_friction` — UNCLEAR: "cannot be bypassed" vs the documented `--dangerously-override-friction` bypass.
- Fix direction: amend the four glossary/concept texts; they contradict `core.provenance_closure`, which they cite.

## B8. User-facing docs describe the old link/closure behavior
- `docs.chapter_04_add_feature` — MISALIGNED: promises "No changes detected" immediately after link — contradicts the new scoped-link semantics (new spec stays NODE_ADDED until reviewed; the correct flow is in chapter 03).
- `docs.chapter_03_add_drift_to_project` — MISALIGNED: mixed old/new — correctly shows spec NODE_ADDED, but also claims a marker NODE_ADDED after link (markers are registered by link now); example output text diverges from actual.
- `docs.chapter_08_remove_feature` — MISALIGNED: describes spec+marker deletion as two closures (same membership → same hash → merged into one) and instructs an unnecessary unlink ("edge persists unless unlinked" — NODE_REMOVED reset filters touching edges per `core/core.go:487-498`).
- `docs.chapter_02_getting_started` — MISALIGNED (R2: promised inline spec/marker examples deferred to ch03; R4 verb list).
- `docs.pedagogy` — VIOLATED (R1: chapters use verbs outside the fixed list; R5: `docs.concept_friction` cites a cli spec, authority rule violated; R6: promised sync specs for topics overlapping skill.md don't exist).
- Fix direction: docs pass after the B1/B3 spec fixes; chapter 03/04/08 need the new link semantics; pedagogy either enforce or relax the spec.

## B9. Remaining misalignments (small text fixes)
- `cli.config_command` / `output.config_theme` — R3: `drift config theme` (no arg) prints the stored preference, not the effective theme (ignores `.drift/theme.xml` precedence). Either implement effective-theme display or amend R3.
- `output.custom_theme` — R2: the invalid-custom-theme rejection error is constructed but swallowed by the sole caller (`tty.go:126`) — silent fallback. Surface it or amend.
- `output.built_in_themes` — R2: protanopia SpecID is magenta (95), spec says "blue/yellow/cyan only".
- `output.code_colorization` / `output_impl.tokenizer` — keyword count 101 vs "~80" (hedge, but 26% low).
- `cli.help_flag` — R3: 4 of 9 enumerated subcommands lack an Example line in usage text.
- `cli.no_bulk_reset` — R2 naming stale (`ResetClosureWithSummary`, hash is first non-flag positional); extra positionals silently dropped.
- `cli.spec_id_predicate_commands` — rationale-only: "to avoid an import cycle" is false (core's predicate is unexported); behavior claims hold.
- `scanner.spec_id_format` — R3: one-dot invariant holds only if module names are dot-free; nothing enforces that.
- `scanner.ignore_file` — "gitignore-style" overstates `filepath.Match` (no `**`, no negation).
- `scanner.scanner_interface` — R5 "orchestrator never constructs specs/markers directly" is literally false (reconcile/link build values); scope the claim to scan provenance.
- `cli.skill` R4 / guide: see B6.
- `output_impl.edge_sort_helper` / `output_impl.list_sort_helpers` — stale counts in rationale prose ("~290 edges", "~180 specs"); current 153/132.
- `open_source_tools_drift.purpose` — stale "to be expanded when output-layer work lands" clause; the referenced specs exist.
- `output_impl.show_rendering` — nit: seed "kind" token never printed in show output.
- `docs.page_navigation` — index.md next-label doesn't match the target chapter's title.

---

# C. Cross-closure contradictions (synthesis pass)

1. **Exit-0 triangle**: `cli.format_todo` + `cli.unlinked_warning` + `cli.todo_alignment_vs_correctness` + `help.txt:37` vs `core.reachability` + `todo.go:21`. Three consumer-facing specs share one wrong sentence. (B3)
2. **Guardrail triangle**: `output.guardrail_property` (property) + `output_impl.guardrail_test` (enforcement) + `output.theme_system` R4 (assertion) vs the actual ChangeSummary padding difference. No walker could see all three; none found the tests. (A1)
3. **Event-kind triangle**: `orch.reconcile_specs`/`orch.reconcile_markers` + two code comments say NODE_CHANGED; `core.todo_action` says NODE_ADDED; two reset specs omit NODE_ADDED entirely. Four texts to fix. (B1)
4. **Flag-enforcement split**: `cli.unknown_flag_rejection` ("reject everything else") vs `cli.show_command`/`cli.reset_friction_block` (require extra flags). (B4)
5. **`.gitignore` convergence**: `cli.gitignore_bootstrap` R6 vs `cli.init_template` R3 vs actual config-path content. (A3)
6. **No-marker docs chapter 08 vs merge rule**: chapter text, `glossary.closure`, and `docs.concept_closure` all predate the same-hash merge rule; `core.provenance_closure` is the only current authority. (B7, B8)

# D. Unclear specs needing a human decision

| Spec | Question |
|---|---|
| `pinstore.load` / `pinstore.xml_types` | Must unversioned (v0) files be refused, or is tolerance intended? (A4) |
| `cli.help` / `output_impl.help_generator` | Is help generation still a planned landing, or is embedded help the permanent state? |
| `output_intent.metadata_single_source` | Does "single source" cover the main help text, or only per-command metadata? |
| `core.edge_helpers` | Is reset's inline NODE_REMOVED edge filter acceptable as a third mutation primitive, or should it route through the named helpers? |
| `scanner.scanner_interface` | Scope of "sole producer of ScanResult" — scan provenance only, or all construction? |
| `docs.concept_friction` | "Cannot be bypassed" — tighten to "cannot be bypassed without an explicit, squawked flag"? |
| `scanner.ignore_file` | Document `filepath.Match` limits honestly, or implement real gitignore semantics? |
| `output.color_mode` / `output.color_palette` | "Status: deleted from disk" renders with StatusWarn in Show but StatusError in DiffClosure — which is intended? |
| `output.code_colorization` | Accept 101 keywords and update the spec, or trim the map? |

# E. Non-issues checked and cleared

- `orch.link`, `orch.reset_closure` body semantics vs code: aligned (this week's rewrite is correctly specified). The only reset finding is the omitted NODE_ADDED event name (B1).
- `model.provenance` (the contract): holds — RESOLUTION section matches current reset exactly; link consistency verified.
- `principles.friction`, `cli.reset_friction_block`, `cli.no_bulk_reset` friction mechanics: aligned end to end.
- `pinstore.dangling_edge_invariant`, `pinstore.save`, `core.validate`, `diff.unified_format`, `fileio.session`, `scanner.ref_parsing`, `output.change_summary_format`, all `testutil.*`: aligned.
- `eval.entry_point`: aligned (README omits newer flags; spec targets main.go, which has them).

# Recommended fix order

1. **A1** (guardrail: fix padding + write the promised tests) — a MUST violation with no enforcement today.
2. **A3** (friction.json gitignore) — one line, prevents a real leak.
3. **A2** (fd leak) — two two-line fixes.
4. **B1 + B3** (event-kind texts + exit-0 triangle) — four text edits, kills two contradiction clusters.
5. **A5, A4, B2** — small, each needs one decision.
6. **B8** (docs chapters 03/04/08) — user-facing correctness.
7. **B5–B7, B9** — batch of spec-text amendments.
8. **D** — human decisions, then fold into the batches above.

Every accepted fix is a normal drift event afterwards: `drift todo` → `drift diff <hash>` → `drift reset <hash>`.
