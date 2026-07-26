# How to fix a bug

[&larr; How to add a feature](04-how-to-add-a-feature.md) | [Index](index.md) | [How to refactor a module &rarr;](06-how-to-refactor-a-module.md)

Now let's see what happens when the agent changes existing code. This
is where drift earns its keep. The agent catches its own mistakes
before you ever see them. We'll deliberately introduce a change that
drifts from the spec, watch drift catch it, and walk through the
resolution.

<!-- D! id=docfix range-start -->

## The setup

Recall our `create_todo` spec: "The title must not be empty." Now
suppose the agent is fixing a bug and decides to also enforce a
minimum title length. A reasonable improvement, but one the spec
didn't ask for:

<!-- D! instruction=ignore-span-start -->
```python
# D! id=ctodo range-start
def create_todo(title, user_id):
    if not title:
        raise ValueError("title must not be empty")
    if len(title) < 3:                              # <- new
        raise ValueError("title must be at least 3 characters")
    todo = {"id": len(todos) + 1, "title": title, "owner": user_id}
    todos.append(todo)
    return todo
# D! id=ctodo range-end
```
<!-- D! instruction=ignore-span-end -->

The agent added a new rule: titles must be at least 3 characters.
This is a perfectly good rule, but the spec only says "must not be
empty." There's a mismatch.

## Drift catches it

When the agent next runs `drift todo`, drift reports the mismatch:

```bash
$ drift todo
1 closure(s) with drift.

Closure a3f7b2c1  (2 nodes: 1 specs, 1 markers; 1 edge)
  Events:
    [NODE-CHANGED] marker "ctodo"  baseline: a1b2c3d4 → scan: e5f6g7h8
  Members:
    specs:   app.create_todo
    markers: ctodo
  Inspect: drift diff a3f7b2c1
  Resolve: drift reset a3f7b2c1
```

Drift caught it! The closure shows that `ctodo` changed. The spec
`app.create_todo` is in the closure too, because the marker links to
it. If other specs cited `app.create_todo` via `<ref>`, they would
also appear. Drift propagates along the citation graph. (We'll see
that in action in chapter 7.)

## Read the diff

The agent runs `drift diff a3f7b2c1` to see exactly what changed:

```bash
$ drift diff a3f7b2c1
```

The diff shows the baseline code (what was there before) vs the current
code inside the marker. You'll see the two new lines, the
minimum-length check. The spec says "must not be empty," but the code
now also says "must be at least 3 characters." That's a new rule.

## Decide: spec wrong, code wrong, or intentional?

This is the heart of the self-correction loop. The agent reads the
diff and decides:

**If the new rule is correct**, update the spec to match:

<!-- D! instruction=ignore-span-start -->
```xml
<spec id="create_todo">
  Create a new todo item. The title must not be empty and must be
  at least 3 characters. Returns the new todo with an auto-incremented id.
</spec>
```
<!-- D! instruction=ignore-span-end -->

Then the agent resolves the closure with `drift reset a3f7b2c1`.

**If the new rule is wrong**, the agent removes the check from the
code, then resolves.

**If the change was purely intentional** (the code and spec already
agree, the hash just changed for structural reasons), the agent
resolves without changing anything.

Either way, the agent doesn't blindly reset. It reads the diff, checks
the spec, and decides which side is wrong. That decision, made before
reporting done, is what makes the agent more accurate.

## The six drift event types

You'll encounter six types of drift events as you use drift. Here's
what each one means:

A **NODE_CHANGED** event fires when a spec or marker's hash differs
from baseline. The content changed. This is the most common event
and the one we just saw.

A **NODE_ADDED** event fires when a new spec or marker appears in the
scan that isn't in the baseline. This happens when the agent adds a
new feature (chapter 4) or bootstraps drift on an existing codebase
(chapter 3).

A **NODE_REMOVED** event fires when a baseline spec or marker is
missing from the scan. This happens when a feature is deleted
(chapter 8).

An **EDGE_ADDED** event fires when a new spec-to-spec citation (via
`<ref>`) appears. An **EDGE_REMOVED** event fires when an existing
citation disappears from the scan.

An **EDGE_BROKEN** event fires when a citation points to a spec that
doesn't exist, usually a typo in the ref target, or the target spec
was deleted without updating the referrer.

Each event seeds a closure. The agent handles them all the same way:
read the diff, decide what's wrong, fix it, resolve.


<!-- D! id=docfix range-end -->

## Try it yourself

<!-- D! id=docfix_ex range-start -->

**Goal:** Tell your agent to edit a function inside a marker in your
project, add a check, change a condition, or rename a variable. Then
have the agent run `drift todo` and read the closure diff with
`drift diff <hash>`. Does the code still match the spec?

**Verify:** If the code matches the spec, the agent resolves the
closure with `drift reset <hash>`. If not, the agent fixes the code or
updates the spec first, then resolves. The agent's `drift todo` should
report clean after resolving.

<!-- D! id=docfix_ex range-end -->

---

<!-- D! id=docnav5 range-start -->
[&larr; How to add a feature](04-how-to-add-a-feature.md) | [Index](index.md) | [How to refactor a module &rarr;](06-how-to-refactor-a-module.md)
<!-- D! id=docnav5 range-end -->
