<p align="center">
  <img src="Drift%20Headline%20Image.png" alt="Drift - sync layer between specs and code for LLM agents" width="800" style="max-width: 100%; height: auto;" />
</p>

# Drift

**Drift is a sync layer between your specs and your code. Your agent
writes both. Drift keeps them aligned.**

You prompt your agent like you already do. The agent writes specs
(capturing your intent) and code (implementing it). Drift connects
them and keeps them in sync. When future edits break alignment, the
agent catches it, and fixes it before you do.

Specs are plain-English rules in XML files. Markers are comment lines
that wrap the implementing code. Any language, any text file. Single
static binary, zero dependencies.

**Full documentation: [docs/index.md](docs/index.md)**

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

## Development principles

<!-- D! id=selfhost range-start -->
Drift dogfoods itself: it tracks its own specs and markers. `drift todo`
must be clean before any commit. This is a hard gate, not a suggestion.
The project is its own primary test case. If drift can't track itself
correctly, it can't track anything. A bug that breaks `drift todo` on
drift's own codebase blocks all other work until fixed.
<!-- D! id=selfhost range-end -->

<!-- D! id=testfirst range-start -->
Bugs are fixed test-first. Write the test that reproduces the bug,
confirm it fails for the right reason, then fix the code and confirm
the test passes. The failing test is proof you understand the bug
before you touch the fix. Never fix a bug without first writing the
test that reproduces it.
<!-- D! id=testfirst range-end -->
