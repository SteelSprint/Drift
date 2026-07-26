# How to add a feature

[&larr; How to add drift to a project](03-how-to-add-drift-to-a-project.md) | [Index](index.md) | [How to fix a bug &rarr;](05-how-to-fix-a-bug.md)

Now let's add a new feature to our todo app and watch the agent handle
it end-to-end. We'll add a "delete todo" feature. The agent will
write the spec, write the code, place the marker, link it, and verify
everything is clean. This is the workflow you'll use every time the
agent adds something new.

<!-- D! id=docfeat range-start -->

## Prompt the agent

You prompt; the agent writes specs and code. Here's a typical prompt:

```
Add a feature to let users delete their own todos. Only the todo
owner can delete, return an error if another user tries.
```

The agent translates this into a spec, compressed intent that the
scanner can verify against the code.

## Write the spec

The agent adds the spec to `app.drift.xml`:

<!-- D! instruction=ignore-span-start -->
```xml
<spec id="delete_todo">
  Delete a todo item. Only the owner can delete their own todo.
  Returns an error if the requester is not the owner.
</spec>
```
<!-- D! instruction=ignore-span-end -->

You might wonder: what makes a good spec? The answer is specificity
without brittleness. "Only the owner can delete" is verifiable and
stable across refactors. "Check ownership using the user_id field"
is too specific. It breaks if the field is renamed. "Handle errors
properly" is too vague to verify at all.

## Write the code and place the marker

The agent writes the implementing code and wraps it with a marker:

<!-- D! instruction=ignore-span-start -->
```python
# D! id=dtodo range-start
def delete_todo(todo_id, requester_id):
    todo = next((t for t in todos if t["id"] == todo_id), None)
    if todo is None:
        raise ValueError("todo not found")
    if todo["owner"] != requester_id:
        raise PermissionError("not the owner")
    todos.remove(todo)
# D! id=dtodo range-end
```
<!-- D! instruction=ignore-span-end -->

The marker wraps the entire implementation region. The shortcode
`dtodo` contains no dot. The scanner hashes the lines between the
markers, excluding the declarations.

## Link and verify

The agent links the marker to the spec and checks for drift:

```bash
$ drift link dtodo app.delete_todo
Linked marker "dtodo" -> spec "app.delete_todo".

$ drift todo
No changes detected. 3 specs, 3 markers, 3 edges in sync.
```

The feature is tracked. Any future change to the spec or the code
inside the marker will produce a closure for the agent to review.
That's it. The agent handled the entire feature, from prompt to
tracked implementation.

<!-- D! id=docfeat range-end -->

## Try it yourself

<!-- D! id=docfeat_ex range-start -->

**Goal:** Prompt your agent to add another feature to the todo app -
maybe "mark a todo as complete" or "edit the todo title." Watch it
write the spec, place the marker, link them, and run `drift todo`.

**Verify:** The agent's `drift todo` reports "No changes detected" with
the spec and marker counts incremented.

<!-- D! id=docfeat_ex range-end -->

---

<!-- D! id=docnav4 range-start -->
[&larr; How to add drift to a project](03-how-to-add-drift-to-a-project.md) | [Index](index.md) | [How to fix a bug &rarr;](05-how-to-fix-a-bug.md)
<!-- D! id=docnav4 range-end -->
