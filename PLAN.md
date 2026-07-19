# Post-Eval Improvements Plan — Closure UX Polish

This plan addresses the convergent feedback from the atomic-cohort eval run
(`atomic-closures-1`, 6 subjects). All 6 subjects completed their tasks
successfully; the closure-driven model lands well. The work below polishes
the UX gaps the subjects surfaced.

## Confirmed decisions

- **Hash format in change summary**: truncated to 8 chars (matches closure-hash vocabulary; full 40-char hashes remain in state.xml and `drift list`).
- **Dry-run exit code**: special code `3` (not 0) to alert LLM consumers that no changes were written. Exit table becomes:
  - `0` clean success
  - `1` drift exists / action pending
  - `2` error (bad args, corrupt state)
  - `3` dry-run preview (no changes written)
- **JSON output**: dry-run and post-apply both include a structured `ChangeSummary` field.

## Specs to add or modify (commit 1)

`cli/cli.drift.xml` — **new specs:**
- `cli.reset_dry_run` — `drift reset --dry-run <hash>` previews the per-event change summary without writing state. Exits 3.
- `cli.reset_summary` — `drift reset <hash>` prints a per-event change summary after applying (same format as the dry-run preview). Exits 0.
- `cli.link_dry_run` — `drift link --dry-run <marker> <spec>` previews. Exits 3.
- `cli.unlink_dry_run` — `drift unlink --dry-run <marker> <spec>` previews. Exits 3.

`cli/cli.drift.xml` — **modify existing specs:**
- `cli.list_format` — edges sorted by `(From, To)` lexicographically.
- `cli.diff_command` — each node annotated `[SEED]` or `[citer]`.
- `cli.skill` — must include the decision-tree cheat sheet and marker-placement guidance as required content.
- `cli.dispatch`, `cli.reset_format`, `cli.link_format`, `cli.unlink_format` — add `--dry-run` to recognized flags; document exit code 3.

`cli/output/output.drift.xml` — **new specs:**
- `output.change_summary_format` — shared structure returned by reset/link/unlink (both preview and post-apply).
- `output.diff_seed_label` — `[SEED]` / `[citer]` annotation rule in diff output.

`cli/output/output_impl.drift.xml` — **modify:**
- `output_impl.result_types` — add `ChangeSummaryResult` to the sealed Result interface.

## Walking skeleton order

Full red/green/refactor cycle per item, easiest first. Each skeleton ends with a clean drift gate (re-baseline if specs changed).

### Skeleton 1 — `#8` list edge sort (trivial, validates the pattern)

**Spec mod**: `cli.list_format` (edges sorted by From,To).

**Red**: `cli/cli_test.go::TestCLI_ListEdgesSorted` — fixture with edges in non-alphabetical order; assert `drift list` output has them sorted.

**Green**: sort `state.Edges` by `(From, To)` in `plain.go`, `color.go`, `json.go` List methods. Pull into helper `sortEdgesByFromTo` in build.go.

### Skeleton 2 — `#4` `[SEED]` / `[citer]` labels in diff

**Spec mod**: `cli.diff_command` + new `output.diff_seed_label`.

**Red**: `cli/cli_test.go::TestCLI_DiffSeedLabel` — drift a spec; `drift diff <hash>`; assert output contains `[SEED]` next to seed and `[citer]` next to non-seed.

**Green**:
1. `core.Closure` — add `Seeds []string` (collected from `event.Seed`).
2. `orchestrator.DiffResult` — add `IsSeed bool`.
3. `DiffClosure` / `DiffAll` — set `IsSeed` based on whether node ID is in the closure's seed set.
4. Presenters — append `[SEED]` or `[citer]` after the ID.

### Skeleton 3 — `#1 + #2` reset/link/unlink dry-run + summary

Walking skeleton: **reset only first**, then propagate to link/unlink.

**Red (reset)**: `TestCLI_ResetDryRun` and `TestCLI_ResetSummary` in `cli/cli_test.go`.
- Dry-run: assert output contains change-summary lines, state.xml unchanged, exit code 3.
- Summary: assert output contains change-summary lines after "Closure resolved", exit code 0.

**Green (reset)**:
1. Define types in `orchestrator/orchestrator.go`:
   ```go
   type ChangeSummary struct {
       Operation   string
       NodeChanges []NodeChange
       EdgeChanges []EdgeChange
   }
   type NodeChange struct {
       ID              string
       Kind            string  // "changed" / "added" / "removed"
       OldHash, NewHash string  // truncated to 8 chars at render time
   }
   type EdgeChange struct {
       From, To string
       Kind     string  // "added" / "removed"
   }
   ```
2. Add `ResetClosureWithSummary(hash) (ChangeSummary, EvaluatedState, error)` — does the work, returns summary, saves.
3. Add `PreviewResetClosure(hash) (ChangeSummary, error)` — same logic, skips Save.
4. Add `--dry-run` to `ResetCommand.Meta().Flags`.
5. Update `ResetCommand.Run`: branch on `--dry-run`; render via new `output.ChangeSummaryResult`.
6. Non-dry-run path renders the summary after "Closure HASH resolved."

**Red (link/unlink)**: parallel tests.

**Green (link/unlink)**: `LinkWithSummary` / `PreviewLink`; `UnlinkWithSummary` / `PreviewUnlink`. Add `--dry-run` flags. Summaries are simpler — just `EdgeChanges`.

**Refactor**: pull change-summary computation (diff two `statestore.State` values) into a helper.

### Skeleton 4 — `#5` decision tree + `#6` marker placement (doc-only)

**Spec mod**: `cli.skill` must include both sections.

**Red**: `cli/cli_test.go::TestSkill_ContainsDecisionTree` and `TestSkill_ContainsMarkerPlacementGuide` — load `skill.md` via go:embed, assert headings present.

**Green**: add two sections to `cli/skill.md` mirroring the drafts in the eval-review thread. Mirror briefly in `DOCUMENTATION.md`.

## Documentation placement

- `cli/skill.md` — primary home for #5 (decision tree) and #6 (marker placement). Both rendered by `drift skill`.
- `DOCUMENTATION.md` — short mirror versions for non-agent readers.
- `cli/help.txt` — update exit-code table to include code 3.

## Verification per skeleton

1. Red test fails for the right reason.
2. Implement to green.
3. `make build` — gate runs `drift todo`; if specs changed, re-baseline closures.
4. `go test -race -count=1 ./...` clean.
5. `GOOS=windows go build ./statestore/` clean.
6. Commit. Push.

## Deferred items

- `#3 drift why <hash>` — wait to see if #4 reduces demand.
- `#7 drift check / lint` — separate planning round (heuristic design questions).
