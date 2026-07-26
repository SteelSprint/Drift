# Drift is a sync layer between your specs and your code

**Drift is a sync layer between your specs and your code. Your agent
writes both. Drift keeps them aligned.**

You prompt your agent like you already do. The agent writes specs
(capturing your intent) and code (implementing it). Drift connects
them and keeps them in sync. When future edits break alignment, the
agent catches it, and fixes it before you do.

Specs are plain-English rules in XML files. Markers are comment lines
that wrap the implementing code. Any language, any text file. Single
static binary, zero dependencies.

## Documentation

The full docs live in [`docs/`](docs/index.md). Start with
[Why drift exists](docs/01-why-drift-exists.md) for the problem and
how drift solves it, or jump straight to
[Getting started](docs/02-getting-started.md) to install and hand off
to your agent.

## Install

**macOS / Linux:**

```bash
curl -fsSL https://raw.githubusercontent.com/SteelSprint/Drift/main/scripts/install.sh | bash
```

**Windows (PowerShell):**

```powershell
irm https://raw.githubusercontent.com/SteelSprint/Drift/main/scripts/install.ps1 | iex
```

Or pin a version:

```bash
# macOS / Linux
DRIFT_VERSION=v1.0.0 curl -fsSL https://raw.githubusercontent.com/SteelSprint/Drift/main/scripts/install.sh | bash

# Windows
$env:DRIFT_VERSION='v1.0.0'; irm https://raw.githubusercontent.com/SteelSprint/Drift/main/scripts/install.ps1 | iex
```

Installs to `~/.local/bin/drift` (macOS/Linux) or
`%USERPROFILE%\.local\bin\drift.exe` (Windows); override with `DESTDIR`.
To build from source: `make build`.

## Quickstart

1. Install drift (see above).
2. Give the binary to your LLM agent and tell it to run `drift skill`.

That's it. The agent learns everything it needs from `drift skill` -
how to write specs, place markers, link them, check for drift, and
resolve it. You prompt; the agent does the rest.

New to drift? Start with [Why drift exists](docs/01-why-drift-exists.md).

## Drift dogfoods itself.

<!-- D! id=selfhost range-start -->
Drift dogfoods itself. Every feature, every CLI command, every design
decision in this repo has a spec. 176 specs, 113 markers, 244 edges,
all in sync, all the time. `make build` runs `drift todo` as a gate:
if any spec drifted from its code, the build fails. No exceptions.
[Browse the specs](docs.drift.xml).
<!-- D! id=selfhost range-end -->


