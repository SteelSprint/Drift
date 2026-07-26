# Getting started

[&larr; Why drift exists](01-why-drift-exists.md) | [Index](index.md) | [How to add drift to a project &rarr;](03-how-to-add-drift-to-a-project.md)

Let's get drift installed and hand it off to your LLM agent. By the end
of this chapter, your agent will be writing specs, placing markers, and
checking its own work, all autonomously.

<!-- D! id=docstart range-start -->

## Install

**macOS and Linux:**

```bash
curl -fsSL https://raw.githubusercontent.com/SteelSprint/Drift/main/scripts/install.sh | bash
```

**Windows (PowerShell):**

```powershell
irm https://raw.githubusercontent.com/SteelSprint/Drift/main/scripts/install.ps1 | iex
```

The binary installs to `~/.local/bin/drift` (macOS/Linux) or
`%USERPROFILE%\.local\bin\drift.exe` (Windows). Add it to your `PATH`
if needed. To build from source instead: `make build` (requires
Go 1.26 or later).

## Give drift to your agent

Drift is a tool for LLMs. The agent is the primary operator. It
writes specs, places markers, runs the drift loop, and resolves
closures. You prompt; the agent does the rest.

After installing, tell your agent to run `drift skill`:

```
Run drift skill to learn how drift works, then initialize drift in
this project.
```

The agent reads the skill guide, runs `drift init`, and starts
surveying your codebase for rules worth specing. It writes specs,
places markers around implementing code, links them, and runs
`drift todo` to verify alignment.

You might be surprised at how little you need to do here. The agent
handles the entire bootstrap process. Your job is to review the specs
it writes. Do they capture the rules you care about?

## What the agent does

<!-- D! instruction=ignore-span-start -->
```
drift init          create .drift/ and a starter main.drift.xml
drift skill         print the agent guide (the agent reads this first)
drift link          connect a marker to a spec
drift todo          check for drift (exit 0 clean, 1 drift, 2 error)
drift diff          show what changed in a closure
drift reset         resolve a closure after reviewing it
```
<!-- D! instruction=ignore-span-end -->

The agent writes specs (compressed intent in XML), places markers
(comment lines wrapping implementing code), links markers to specs,
and runs `drift todo` to verify alignment. When the code or a spec
changes, drift derives a closure and the agent reviews it.

## Verify: drift todo

Let's check that everything is set up correctly. Have the agent run
`drift todo` and show you the output:

```bash
$ drift todo
No changes detected. 3 specs, 3 markers, 3 edges in sync.
```

Your project is now drift-tracked! The agent wrote the specs, placed
the markers, and resolved the initial closures. From here on, drift
has your back: any change to either side produces a closure the agent
will review before reporting done.

Both specs and code are always visible to you. Edit either side.
Drift keeps them in sync.

In the next chapter, we'll walk through adding drift to an existing
project from scratch, so you can see exactly what the agent does during
bootstrap.

<!-- D! id=docstart range-end -->

## Try it yourself

<!-- D! id=docstart_ex range-start -->

**Goal:** Install drift, give the binary to your agent, and tell it to
run `drift skill`. Let the agent initialize the project, write a few
specs for existing code, and run `drift todo`.

**Verify:** The agent's `drift todo` reports "No changes detected" with
the spec and marker counts incremented. If you see closures instead,
that's fine. The agent just hasn't resolved them yet. Tell it to review
and resolve each one.

<!-- D! id=docstart_ex range-end -->

---

<!-- D! id=docnav2 range-start -->
[&larr; Why drift exists](01-why-drift-exists.md) | [Index](index.md) | [How to add drift to a project &rarr;](03-how-to-add-drift-to-a-project.md)
<!-- D! id=docnav2 range-end -->
