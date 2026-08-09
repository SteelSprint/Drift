# Drift lets your agent move fast and break less

You prompt an agent. It writes code fast — and quietly drifts away
from what you asked for. Every edit adds a small assumption you never
approved; over a long task, they pile up. Drift catches that.

Drift makes your agent:

- **More accurate.** It checks its own work against your rules and requirements,
  before telling you it's done.
- **Fewer prompts.** Say what you want once; the agent keeps it
  in mind across every edit after.
- **Better memory.** Your project's rules and requirements live in the repo and are statically linked to your code.
- **Fewer mistakes.** When a change breaks a rule, the agent
  finds out before you do.
- **Pushback when it matters.** Ask for something that contradicts
  your own requirements, and the agent flags it: *"Are you sure? This goes
  against what you wanted earlier."*
- **Long tasks that stay reliable.** Steps get checked much more often, so a
  50-step task doesn't quietly pile up 50 small wrong turns.
- **Work that outlives the conversation.** Close the tab, start
  fresh, hand it to another agent — the requirements are still there, no
  re-explaining required.

## No new workflow to learn

You keep prompting the way you already do. Drift is a tool your agent
picks up, not one you have to learn. The agent writes the specs, places
the markers, runs the checks, and resolves any drift — your job is to
review.

Behind the scenes: **specs** are your rules and requirements in plain English, **markers**
wrap the code that implements each requirement, and **drift** is the signal
that fires when the two disagree. Start with [Why drift exists](docs/01-why-drift-exists.md)
for the full picture.

## Quickstart

1. Install drift (see [Install](#install) below).
2. Give the binary to your LLM agent and tell it to run `drift skill`.

That's it. The agent learns everything it needs from `drift skill` -
how to write specs, place markers, link them, check for drift, and
resolve it. You prompt; the agent does the rest.

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
curl -fsSL https://raw.githubusercontent.com/SteelSprint/Drift/main/scripts/install.sh | DRIFT_VERSION=v1.0.0 bash

# Windows
$env:DRIFT_VERSION='v1.0.0'; irm https://raw.githubusercontent.com/SteelSprint/Drift/main/scripts/install.ps1 | iex
```

Installs to `~/.local/bin/drift` (macOS/Linux) or
`%USERPROFILE%\.local\bin\drift.exe` (Windows); override with `DESTDIR`.
To build from source: `make build`.

## Drift dogfoods itself.

<!-- D! id=selfhost range-start -->
Drift dogfoods itself. Every feature, every CLI command, every design
decision in this repo has a spec. `make build` runs `drift todo`
as a gate: if any spec drifted from its code, the build fails. No
exceptions. [Browse the specs](docs.drift.xml).
<!-- D! id=selfhost range-end -->

## Prior art

I built this without knowing about [fiberplane/drift](https://github.com/fiberplane/drift)
or [spec-kit-sync](https://github.com/bgervin/spec-kit-sync), both of which
attack the same problem and one of which has the same name. That's on me for not
searching properly. See [PRIOR_ART.md](./PRIOR_ART.md) for what they do, how this
differs, and where their approach is better than mine.

