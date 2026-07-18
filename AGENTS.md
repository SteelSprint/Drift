# Drift — Agent Guide

Drift is a spec-drift detection tool for LLM coding agents. Specs describe behavior; markers wrap the code that implements each spec. When either side changes, `drift todo` surfaces the drift so the agent can verify alignment before resolving.

## Spec discipline workflow (MUST follow)

1. **`drift todo`** — check what drifted (specs, markers, or unlinked)
2. **`drift diff --all`** — review every broken edge's changes in one pass
3. **For each edge:** decide whether the *code* is wrong (fix the code) or the *spec* is wrong (update the spec)
4. **`drift reset <marker> <module.spec>`** — resolve ONE edge at a time, only after reviewing it

**NEVER batch-reset.** There is no `drift reset --all`. This friction is the point — blind reset defeats the tool.

**`drift todo` exit 1 means unfinished work.** Exit 0 requires both (a) all markers linked and (b) all edges in sync. Unlinked markers are actionable drift.

## Critical rules

- **Specs are the source of truth.** When a spec and code disagree, first decide which is correct. If the code is right, update the spec *then* reset. If the spec is right, fix the code *then* reset.
- **Spec IDs have exactly one dot** (module separator): `main.bootstrap`, `orch.link`. Marker shortcodes have no dot. Never put a dot in a `<spec id="...">` local ID.
- **Markers wrap the implementation region** with `// D! id=<shortcode> range-start` and `// D! id=<shortcode> range-end`. The scanner hashes the lines between the markers.
- **Commit `.drift/state.xml` and `.drift/baselines/` to git.** They are shared baselines, not local artifacts. Do NOT commit `.drift/user-settings.xml` or `.drift/state.lock` (both gitignored).
- **State file locking is built in.** Concurrent `drift link`/`unlink`/`reset` calls are safe — flock (Unix) or LockFileEx (Windows) serializes Load→Save. Safe to batch these in parallel tool calls.

## Build / test / lint

```sh
go build -o drift ./cmd/drift          # build the binary
go test -race -count=1 ./...           # full suite with race detector
GOOS=windows go build -o /dev/null ./statestore/   # verify Windows compiles
```

- Module path is `drift`, Go 1.26.
- One external dependency: `golang.org/x/sys` (for cross-platform file locking in `statestore/`). Do not add dependencies without strong justification.
- The race test (`cli/race_test.go`) runs on every `go test ./...` — it is a regression guard for concurrent state mutations, not optional.

## Repo layout

```
cmd/drift/       # main() entry point
cli/             # CLI dispatch, command structs, output layer (Plain/Color/JSON)
  commands/      # one struct per subcommand (init, todo, link, reset, …)
  output/        # presenters, themes, tokenizer, user settings
core/            # core algorithm (evaluated state, scan, reconcile)
scanner/         # file scanner — specs from *.drift.xml, markers from code
statestore/      # FileStateStore (state.xml), BaselineStore, file locking
orchestrator/    # wires scanner + statestore + core; mutating methods hold lock
eval/            # eval harness (subjects an LLM to a drift fixture, judges result)
internal/        # diff, testutil
business/        # product spec hierarchy (goals → modules → intent → impl)
```

## Specs in this repo

The drift codebase is self-hosting on drift. Specs live in `*.drift.xml` files next to the code they describe:

- `cli/cli.drift.xml` — CLI command contracts
- `orchestrator/orchestrator.drift.xml` — orchestrator method contracts
- `cli/output/output.drift.xml` + `output_impl.drift.xml` — output layer (L1/L2/L3)
- `business/` — product-level goal hierarchy

Current state: 121 specs, 68 markers, 81 links. `drift todo` should report clean on a resting tree.

## Editing code that drift tracks

When you change code inside a `// D! id=… range-start … range-end` region:

1. Run `drift todo` — the edge will show as drifted
2. Run `drift diff <marker> <module.spec>` — see the code delta
3. Read the linked spec and decide: does the spec still describe the new code?
4. If yes → `drift reset <marker> <module.spec>` (baseline collapses)
5. If no → update the spec text in the `*.drift.xml` file, then reset

When you change a spec's wording in a `*.drift.xml` file:

1. Run `drift todo` — the edge will show as drifted (spec side)
2. Read the linked marker region in the code
3. Decide: does the code still implement the new spec?
4. If yes → `drift reset`
5. If no → fix the code, then reset

## Adding new specs

1. Add `<spec id="localid">description</spec>` to the relevant `*.drift.xml` module file (local ID must NOT contain a dot)
2. Wrap the implementing code region with `// D! id=<shortcode> range-start` / `range-end`
3. `drift link <shortcode> <module.localid>`
4. `drift todo` — should report clean

## Eval harness

`eval/` runs an LLM ("subject") against a drift fixture workspace, then a judge LLM scores the result. Used to validate that agents can use drift correctly and that drift itself doesn't have UX footguns.

```sh
go run ./eval --battery --repeat 10 --subject <model> --judge <model>
```

Per-prompt overrides via `<name>-subject.md` and `<name>-judge.md` files alongside `<name>.md`. The `--repeat N` flag runs the same prompt N times in parallel for a statistical baseline.

## Output modes

Every command supports three output modes:

- **Plain** (default when piped) — stable text, no ANSI
- **Color** (default in TTY) — themed ANSI + syntax highlighting
- **JSON** (`--json`) — structured output for programmatic consumption

For scripting or LLM consumption, use `--json` or `--no-color`.

## Themes

`drift config theme <name>` sets a per-user preference (stored in `.drift/user-settings.xml`, not committed). 12 built-in themes. Project-level custom theme via `.drift/theme.xml` (committed, full override of all 18 elements).

## Quick reference

| Task | Command |
|---|---|
| What drifted? | `drift todo` |
| Show the diffs | `drift diff --all` |
| Resolve one edge | `drift reset <marker> <module.spec>` |
| List everything | `drift list --verbose` |
| Show one entity | `drift show <marker\|spec>` |
| Full guide | `drift skill` |
| Command reference | `drift help` |
| Structured output | `drift todo --json` |
