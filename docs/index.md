# Drift documentation

[Getting started &rarr;](01-why-drift-exists.md)

Drift is a sync layer between your specs and your code. Your agent
writes both. Drift keeps them aligned.

First time? Start with [Why drift exists](01-why-drift-exists.md).

## Chapters

1. **[Why drift exists](01-why-drift-exists.md)**: the problem.
   Intent drifts from code at agent speed. Sync amplifies what agents
   can do.

2. **[Getting started](02-getting-started.md)**: install drift, give
   it to your agent, watch it work.

3. **[How to add drift to a project](03-how-to-add-drift-to-a-project.md)**:
   the agent surveys existing code, writes retroactive specs, places
   markers, and resolves closures.

4. **[How to add a feature](04-how-to-add-a-feature.md)**: prompt the
   agent; it writes the spec, the code, the marker, and verifies
   alignment.

5. **[How to fix a bug](05-how-to-fix-a-bug.md)**: the self-correction
   loop. Drift detects misalignment, the agent reads the diff, decides
   what's wrong, and fixes it.

6. **[How to refactor a module](06-how-to-refactor-a-module.md)**:
   why drift flags refactors, marker placement strategy, moving
   markers across files.

7. **[How to change a spec](07-how-to-change-a-spec.md)**: edit a
   spec, watch drift propagate through the citation graph, and have
   the agent update all affected code.

8. **[How to remove a feature](08-how-to-remove-a-feature.md)**:
   clean removal. Delete spec, remove marker, unlink, resolve.

## In-CLI guides

- **`drift skill`**: the complete agent guide (what your LLM reads)
- **`drift help`**: command reference with examples

---

<!-- D! id=docnavs range-start -->
[Index](index.md) | [Getting started &rarr;](01-why-drift-exists.md)
<!-- D! id=docnavs range-end -->
