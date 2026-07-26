# How to remove a feature

Let's wrap up our tour of the core workflows by removing a feature
cleanly. When you delete a feature, the spec and marker both go away.
Drift reports NODE_REMOVED events, and the agent resolves them. Get
this wrong and you'll leave orphan state. Get it right and the
removal is seamless.

<!-- D! id=docrem range-start -->

## Remove the spec and marker

Let's remove the `delete_todo` feature we added in chapter 4. Tell the
agent:

```
Remove the delete_todo feature entirely. Delete the spec, remove
the marker, and clean up drift state.
```

The agent:

1. Deletes the spec from `app.drift.xml`.
2. Removes the marker lines (`range-start` and `range-end`) from the
   code file.
3. Optionally removes the implementing code (or leaves it unmarked if
   it's still used by something else).

## Drift detects removal

Both the spec and the marker are now gone from the scan. Run
`drift todo`:

```bash
$ drift todo
2 closure(s) with drift.

Closure b4c8d2e9  (1 nodes: 0 specs, 1 markers; 0 edges)
  Events:
    [NODE-REMOVED] marker "dtodo"
  ...

Closure c5d9e3f0  (1 nodes: 1 specs, 0 markers; 0 edges)
  Events:
    [NODE-REMOVED] spec "app.delete_todo"
  ...
```

The marker and spec are gone from the scan but still in the baseline.
The edge connecting them is now orphaned. Let's clean that up.

## Unlink and resolve

The agent unlinks the marker from the spec to remove the orphaned
edge, then resolves both closures:

```bash
$ drift unlink dtodo app.delete_todo
Unlinked marker "dtodo" from spec "app.delete_todo".

$ drift reset b4c8d2e9
Closure b4c8d2e9 resolved.

$ drift reset c5d9e3f0
Closure c5d9e3f0 resolved.
```

```bash
$ drift todo
No changes detected. 2 specs, 2 markers, 2 edges in sync.
```

The feature is cleanly removed. No orphan state remains.

## Pitfalls to avoid

A few things can go wrong during removal. If you remove only the spec
and leave the marker, the marker exists in the scan with no spec to
link to. The edge is broken. If you remove only the marker and leave
the spec, the spec is orphaned with no implementing code.

Always remove both sides. The spec from the `.drift.xml` file and the
marker from the code. And don't forget to `drift unlink` the edge
before resolving, or it'll persist in the baseline even after both
endpoints are gone.

<!-- D! id=docrem range-end -->

## Try it yourself

<!-- D! id=docrem_ex range-start -->

**Goal:** Remove a feature from your project. Have the agent delete
the spec, remove the marker, unlink the edge, and resolve the
closures.

**Verify.** `drift todo` reports "No changes detected" with the spec
and marker counts decremented. No orphan markers or specs remain.

<!-- D! id=docrem_ex range-end -->
