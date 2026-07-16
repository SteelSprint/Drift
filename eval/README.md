# Driftpin Eval Pipeline

LLM-as-judge evaluation of driftpin's cold-start UX. A subject LLM is handed the tool (binary + docs only) and asked to build a project using driftpin end-to-end. A judge LLM evaluates the result and produces tool-improvement recommendations.

## Usage

### Single prompt

```sh
make eval PROMPT="create a working CLI version of poker"
```

### Full test battery

```sh
go run ./eval --battery
```

### Override models

```sh
go run ./eval --subject openrouter/anthropic/claude-haiku-4.5 --judge openrouter/anthropic/claude-opus-4.8 "build a TODO app"
```

### Dry run (stage only, skip LLM calls)

```sh
go run ./eval --dry-run "build a TODO app"
```

## How it works

1. **Stage** — Builds the `drift` binary, copies it + `README.md` + `DOCUMENTATION.md` into a fresh `eval/runs/<timestamp>/tool/` directory. Creates an empty `workspace/` dir. Stages custom agent definitions for the subject and judge.

2. **Subject run** — An LLM (default: MiMo v2.5 Pro) is launched in the empty workspace via `opencode run --agent eval-subject --auto`. It reads the docs from `../tool/`, completes the task, uses driftpin end-to-end (init, specs, markers, links, `drift todo`), and writes a `self-debrief.md` with structured feedback.

3. **Judge run** — A smarter LLM (default: GLM-5.2) is launched in the run directory via `opencode run --agent eval-judge --auto`. It inspects the subject's workspace (runs `drift todo`, reads spec files, checks markers/links), reads the subject's `self-debrief.md`, samples the transcript, and writes a `report.md`.

4. **Surface** — The `report.md` is printed to stdout. A row is appended to `eval/runs/log.csv`.

## The feedback loop

- **Subject → Judge**: `self-debrief.md` — the end-user LLM's direct feedback (what worked, what confused them, errors, missing docs, suggestions).
- **Judge → Tool authors**: `report.md` section 3 — prioritized tool-improvement recommendations. These get triaged into `PLAN.md`.

## Run artifacts

```
eval/runs/<timestamp>/
  tool/              # binary + docs (the "tool" handed to the subject)
  workspace/         # subject's completed project (ground truth)
    self-debrief.md  # subject's feedback
  .opencode/agents/  # judge agent definition
  subject.jsonl      # full subject transcript
  judge.jsonl        # full judge transcript
  report.md          # the evaluation (scorecard + qualitative + recommendations)
```

## Agent configuration

- **`eval-subject`** (`eval/agents/eval-subject.md`): primary agent, 40 steps, full tool access + external directory access. Cold-start system prompt for run-to-run consistency.
- **`eval-judge`** (`eval/agents/eval-judge.md`): primary agent, read + bash access, edit scoped to `report.md` only. Cannot corrupt the workspace.
