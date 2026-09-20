# Drift — Agent Guide

## Session start (MUST)

Run `drift skill` at the start of every session before touching code. It carries the current workflow, marker rules, decision tree, spec-discipline workflow, exit codes, closure properties, and the command reference. Do not rely on memory of this file alone — the skill guide is the live contract. This file holds only what is specific to this repo (the dogfood layer); the shared engineering discipline cascades from the workspace root `AGENTS.md`.

## Development discipline (MUST follow)

The shared root `AGENTS.md` carries the standing discipline: red before green at the integration level, one-closure-per-review, drift-spec everything, coverage discipline, verify-before-commit, format-only-touched, no tracker files. This repo applies them as follows:

- **Red before green means `cli.RunWithRender` or the orchestrator.** Integration tests go through `cli.RunWithRender` (real temp project) or the orchestrator, whichever matches where the behavior lives. Unit-level tests alone (e.g. core package, in-memory fixtures) are NOT sufficient: bugs like state-file corruption only appear through the real file paths. Unit tests are added alongside, not instead.
- **The build gate enforces `drift todo` cleanliness.** `make build` runs `./drift todo` as a spec-drift gate; the build fails if any drift is detected. The daily loop: `drift todo` → `drift diff <hash>` → `drift reset <hash>` (one closure per review).
- **Coverage baseline (2026-09-19): 63.0%.** The repo-wide target is ≥80% of walked lines covered by linked markers; the gap closes through normal spec work, not bulk marking.

## Critical rules

The full mechanics — marker format, ref semantics, closure identity and properties, reset events, state-file committing and locking — live in `drift skill`. The rules below are the ones sessions get wrong in THIS repo; everything else is in the skill guide.

- **Commit `.drift/state.xml` and `.drift/baselines.bin` to git.** They are shared baselines, not local artifacts. Do NOT commit `.drift/user-settings.xml`, `.drift/state.lock`, or `.drift/friction.json` (all gitignored).
- **`GOOS=windows go build ./...` and `go vet` on ALL packages** before a release: the release builds everything cross-platform; a cli/orchestrator-only Windows break must be caught here, not in CI.
- **One external dependency**: `golang.org/x/sys` (cross-platform file locking). Do not add dependencies without strong justification.
- **The race test (`cli/race_test.go`) runs on every `go test ./...`** — a regression guard for concurrent state mutations, not optional.
- **State.xml v4 only.** Pre-v4 files are refused with a clear error directing the user to re-init.
- **`make build` backs up the prior binary** to `bak/drift-<UTC-timestamp>` (gitignored). Roll back with `cp bak/drift-<ts> drift`.

## Build / test / lint

```sh
make build                              # build + drift gate (preferred)
go build -o drift ./cmd/drift           # build only, skip gate
go test -race -count=1 ./...            # full suite with race detector
GOOS=windows go build ./...             # verify Windows compiles (ALL packages —
                                        # the release builds everything cross-platform;
                                        # a cli/orchestrator-only Windows break must be
                                        # caught here, not in CI)
GOOS=windows go vet ./...               # same scope for vet
```

- Module path is `drift`, Go 1.26.
- One external dependency: `golang.org/x/sys` (for cross-platform file locking in `internal/fileio/`). Do not add dependencies without strong justification.
- `make build` runs `./drift todo` as a spec-drift gate (see Critical rules).

## Repo layout

```
cmd/drift/       # main() entry point
cli/             # CLI dispatch, command structs, output layer (Plain/Color/JSON)
  commands/      # one struct per subcommand (init, todo, link, reset, …)
  output/        # presenters, themes, tokenizer, user settings
core/            # core algorithm (Closure, DriftEvent, DeriveClosures, EvaluateState)
scanner/         # file scanner — specs and refs from *.drift.xml, markers from code
statestore/      # FileStateStore (state.xml v4), BaselineStore (baselines.bin packfile)
orchestrator/    # wires scanner + statestore + core; mutating methods receive a fileio.Session from the caller
eval/            # eval harness (subjects an LLM to a drift fixture, judges result)
internal/        # diff, testutil
business/        # product spec hierarchy (goals → modules → intent → impl)
model.drift.xml  # CONCEPTUAL SPEC — model.provenance (above all impls)
```

## Specs in this repo

The drift codebase dogfoods drift. Specs live in `*.drift.xml` files next to the code they describe:

- `model.drift.xml` — `model.provenance`: notation, axioms, algorithm
- `cli/cli.drift.xml` — CLI command contracts
- `core/core.drift.xml` — core algorithm contracts (validate, todo_action, reset_action, scan_coverage, provenance_closure)
- `orchestrator/orchestrator.drift.xml` — orchestrator method contracts
- `cli/output/output.drift.xml` + `output_impl.drift.xml` — output layer
- `statestore/statestore.drift.xml` — state.xml v4 + baseline store
- `business/` — product-level goal hierarchy

Current state: 132 specs, 71 markers, 153 edges. `drift todo` should report clean on a resting tree.

Current state: `drift todo` should report clean on a resting tree.

The daily workflow — editing code inside markers, editing specs, adding specs, citing specs — is the standard loop from `drift skill`; this repo follows it without modification. Marker shortcodes in this repo follow the module's spec files (see the specs list above).

## Eval harness

`eval/` runs an LLM ("subject") against a drift fixture workspace, then a judge LLM scores the result. Used to validate that agents can use drift correctly and that drift itself doesn't have UX footguns.

```sh
go run ./eval --battery --repeat 10 --subject <model> --judge <model>
```

Per-prompt overrides via `<name>-subject.md` and `<name>-judge.md` files alongside `<name>.md`. The `--repeat N` flag runs the same prompt N times in parallel for a statistical baseline.

For output modes, themes, and the full command reference, see `drift skill`.

