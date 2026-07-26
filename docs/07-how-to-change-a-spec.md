# How to change a spec

So far we've been changing code and letting drift catch the mismatch.
Now let's flip it: we'll change a spec and watch drift propagate the
change through the citation graph. This is where drift becomes a
control plane, one spec edit shows you every piece of code that
depends on it.

<!-- D! id=docsteer range-start -->

## Edit the spec

Let's tighten the password rule in our todo app. Suppose we originally
spec'd:

<!-- D! instruction=ignore-span-start -->
```xml
<spec id="create_todo">
  Create a new todo item. The title must not be empty.
  Returns the new todo with an auto-incremented id.
</spec>
```
<!-- D! instruction=ignore-span-end -->

Now we want to require a minimum title length. We update the spec:

<!-- D! instruction=ignore-span-start -->
```xml
<spec id="create_todo">
  Create a new todo item. The title must not be empty and must be
  at least 3 characters. Returns the new todo with an auto-incremented id.
</spec>
```
<!-- D! instruction=ignore-span-end -->

The spec content changed. When the agent runs `drift todo`, drift
reports:

```bash
$ drift todo
1 closure(s) with drift.

Closure d5e6f7a8  ...
  Events:
    [NODE-CHANGED] spec "app.create_todo"  baseline: ... → scan: ...
```

Drift detected the spec change.

## Drift propagates

Now here's where it gets powerful. If other specs cite
`app.create_todo` via a `<ref>` tag, those specs also appear in the
closure. This is the **citer chain**: when a spec changes, every spec
that transitively cites it is suspect.

Suppose `app.delete_todo` cites `app.create_todo` (both deal with
todo items, and `delete_todo` references the creation contract). The
citation graph looks like:

```
app.create_todo  (changed)
  ← cited by app.delete_todo
    ← cited by app.api_handler
```

All three specs land in the closure. Every marker linked to those specs
also appears. The closure is the complete set of code affected by this
spec change.

## The agent reads the closure as a work order

The closure tells the agent: "here's everything affected by this
change." The agent:

1. Reads the closure membership: which specs and markers are in it.
2. Reads the diff: what changed in the spec.
3. Updates the code inside each marker to match the new spec.
4. Resolves the closure.

For our title-length change, the agent checks the `create_todo` marker.
If the code already enforces the 3-character minimum (maybe the agent
added it in chapter 5), the spec and code agree. Just resolve. If not,
the agent adds the check and resolves.

If `delete_todo` or `api_handler` need updates too, the agent sees them
in the closure and fixes them before resolving. No hunting through the
codebase for "what else uses this rule." The citation graph already
knows.

## The control plane in action

This is specs as a control plane. One edit to a spec shows you all
downstream impact. The agent updates the affected code. You review
the result.

You don't need to search the codebase for dependencies. Drift surfaces
them automatically through the citation graph. We'll cover how to set
up `<ref>` citations between specs in more detail in a future chapter.
For now, the key takeaway: editing a spec is how you steer the project.

Finally, let's see how to cleanly remove a feature when it's no longer
needed: [How to remove a feature &rarr;](08-how-to-remove-a-feature.md)

<!-- D! id=docsteer range-end -->

## Try it yourself

<!-- D! id=docsteer_ex range-start -->

**Goal:** Edit a spec in your project. Tighten or loosen a
requirement. Then have the agent run `drift todo` and look at the
closure membership. If you have `<ref>` citations set up, the closure
should include transitively dependent specs.

**Verify:** Read the closure output. Have the agent update any affected
code, then resolve the closure.

<!-- D! id=docsteer_ex range-end -->
