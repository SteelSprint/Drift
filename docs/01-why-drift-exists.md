# Why drift exists

Every project has rules. "The title must not be empty." "Rate limits
apply per user, not per IP." "All API responses include a request ID."
Some rules are written in documentation. Others live in code comments.
Many live only in the maintainers' heads.

When a human engineer writes code, they carry these rules in working
memory. They might forget one occasionally, but code review catches the
miss. At human writing speed, the system works.

LLM coding agents changed the equation. An agent generates code at a
volume and velocity that overwhelms human review. An agent does not
carry your project's implicit rules. It carries the patterns it
learned during training, which may or may not match what your project
actually requires. When an agent adds a feature, it also adds
assumptions, side effects, and new rules you never asked for.

<!-- D! id=docwhy range-start -->

## The problem: spec drift

Every line of agent-generated code is a hypothesis about what your
project should do. Most hypotheses are correct. Some are not. The
incorrect ones are the problem, not because they are bugs (tests catch
bugs), but because they encode a different understanding of the
project's rules than you hold.

This gap, between what the rules say and what the code does, is
**spec drift**: It accumulates silently. Each agent edit introduces a
small drift. Over hundreds of edits, the codebase drifts far from the
original intent. You discover the drift weeks later, if at all.

## Why traditional review fails

Three mechanisms are supposed to catch drift, but each one struggles at
agent volume:

**Code review** works when a human can read every diff. At hundreds of
files per pull request, the reviewer skims. Drift slips through.

**Testing** verifies behavior, not intent. A test confirms the code
does X. It does not confirm X is the correct rule. An agent adding a
minimum-length check to a title field produces passing tests, but the
rule was "must not be empty," not "must be at least three characters."
The test passes. The drift is real.

**Documentation** is where the rules live. But documentation does not
enforce itself. An agent does not read the documentation before editing
code unless explicitly told to, and even then, it applies its own
interpretation.

## Drift: a sync layer between specs and code

Here's the idea. Drift is a sync layer between your specs and your
code. Your agent writes both. Drift keeps them aligned.

Specs are compressed intent: short, plain-English rules stored in XML
files. Markers are comment lines that wrap the code implementing each
spec. When either side changes, drift derives a **closure** (the set
of specs and markers affected by the change) and surfaces it for the
agent to review.

The agent reads the closure, checks whether the code still matches the
spec, and fixes whichever side is wrong. You prompt; the agent writes
specs and code; drift keeps them in sync.

This sync makes the agent more accurate (it checks its own work against
the specs before reporting done) and more agile (it moves fast, knowing
drift catches misalignment). Drift doesn't slow the agent down. It
amplifies what the agent can do.

We'll see all of this in practice over the next few chapters. Let's
get started: [Getting started &rarr;](02-getting-started.md)

<!-- D! id=docwhy range-end -->

## Try it yourself

<!-- D! id=docwhy_ex range-start -->

**Goal:** Think of a recent pull request where an LLM agent added code
that violated an implicit rule in your project. Name the rule and write
down what spec text would have caught the violation.

**Verify.** The spec should be one or two sentences, specific enough
that a scanner comparing it to the agent's code would have flagged the
drift.

<!-- D! id=docwhy_ex range-end -->
