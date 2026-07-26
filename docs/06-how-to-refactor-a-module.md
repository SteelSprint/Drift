# How to refactor a module

Refactoring is where drift's design gets interesting. You might
expect drift to ignore "pure" refactors (moves, renames, structural
reorganization that don't change behavior). It doesn't. Drift flags
every content change inside a marker, including refactors. Let's see
why.

<!-- D! id=docrefac range-start -->

## Refactors produce drift

Let's refactor our todo app. Suppose the agent extracts a validation
helper from `create_todo`:

<!-- D! instruction=ignore-span-start -->
```python
def validate_title(title):
    if not title:
        raise ValueError("title must not be empty")
    if len(title) < 3:
        raise ValueError("title must be at least 3 characters")

# D! id=ctodo range-start
def create_todo(title, user_id):
    validate_title(title)
    todo = {"id": len(todos) + 1, "title": title, "owner": user_id}
    todos.append(todo)
    return todo
# D! id=ctodo range-end
```
<!-- D! instruction=ignore-span-end -->

The behavior is the same. But the content inside the `ctodo` marker
changed. The validation logic moved out. When the agent runs
`drift todo`, drift reports:

```bash
$ drift todo
1 closure(s) with drift.

Closure c4d5e6f7  ...
  Events:
    [NODE-CHANGED] marker "ctodo"  baseline: ... → scan: ...
```

Drift flagged it. You might wonder: why? If the refactor didn't change
behavior, shouldn't drift ignore it?

## Why drift doesn't skip refactors

Because drift can't tell the difference. A hash change is a hash
change. Drift sees the bytes inside the marker changed, nothing more.
It doesn't analyze whether the change is structural or behavioral.

And that's the right design. Refactors that accidentally change
behavior are the most dangerous bugs. They look innocent in a diff -
just moving code around, but they can introduce subtle regressions.
By flagging every change, drift forces the agent to verify: "I moved
this code. Did I also change what it does?"

The agent reads the diff, confirms the change is purely structural,
and resolves. A few seconds of review for a safety guarantee.

## Marker placement strategy

You can reduce refactor noise by placing markers thoughtfully. Keep
markers on the **public entry-point function**: The function the spec
describes, not on internal helpers.

In our example, the marker wraps `create_todo`, not `validate_title`.
When the agent refactors `validate_title` (extracting more logic,
renaming variables), the `ctodo` marker doesn't change, as long as
the call to `validate_title(title)` stays inside the marker.

If you find yourself marking every internal helper, you're
over-marking. One marker per spec, on the entry point. Helpers stay
outside.

## When you need to move a marker

Sometimes a refactor moves the implementing code to a different file
or function entirely. In that case:

1. The agent places the marker around the new location.
2. The agent removes the marker from the old location.
3. `drift todo` reports NODE_REMOVED for the old marker and NODE_ADDED
   for the new one.
4. The agent links the new marker to the spec (if the shortcode
   changed) and resolves both closures.

The spec doesn't change, only the marker location moves.

So far we've been changing code and letting drift catch the mismatch.
Next, let's flip it, change a spec and watch drift propagate the
change through the citation graph: [How to change a spec &rarr;](07-how-to-change-a-spec.md)

<!-- D! id=docrefac range-end -->

## Try it yourself

<!-- D! id=docrefac_ex range-start -->

**Goal:** Tell your agent to pick a function inside a marker and
extract a helper from it, moving 2-3 lines into a separate function
outside the marker. Then have the agent run `drift todo` and read the
diff. Confirm the change is purely structural.

**Verify:** The diff shows the extraction. The spec still describes the
function's behavior. The agent resolves the closure.

<!-- D! id=docrefac_ex range-end -->
