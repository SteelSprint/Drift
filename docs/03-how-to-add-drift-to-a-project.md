# How to add drift to a project

In this chapter, we'll walk through adding drift to an existing
codebase. A small Python todo app. You'll see exactly what the agent
does during bootstrap: surveying code, writing retroactive specs,
placing markers, and resolving the initial closures. This is the
workflow every new drift user goes through.

If you don't have a project handy, create a simple `app.py` with a few
functions. We'll use this todo app throughout the rest of the book:

```python
todos = []

def create_todo(title, user_id):
    if not title:
        raise ValueError("title must not be empty")
    todo = {"id": len(todos) + 1, "title": title, "owner": user_id}
    todos.append(todo)
    return todo

def list_todos(user_id):
    return [t for t in todos if t["owner"] == user_id]
```

<!-- D! id=docaddp range-start -->

## Survey existing code

Tell the agent to survey the codebase and identify rules worth specing:

```
Run drift skill, then survey this codebase. Identify the key rules
and behaviors this project enforces. Write a spec for each rule and
place markers around the code that implements it.
```

The agent reads `drift skill`, walks the codebase, and identifies
behaviors that are rules, not just implementation details. In our
todo app, good spec candidates include:

- "The title must not be empty" (validation in `create_todo`)
- "Todos are scoped per user" (filtering in `list_todos`)

These are rules the project enforces. If the agent later changes the
code in a way that breaks one of these rules, drift will catch it.

## Write specs

The agent writes each rule as a spec, compressed intent in a
`*.drift.xml` file:

<!-- D! instruction=ignore-span-start -->
```xml
<module name="app">
  <spec id="create_todo">
  Create a new todo item. The title must not be empty.
  Returns the new todo with an auto-incremented id.
  </spec>

  <spec id="list_todos">
  List all todos belonging to the requesting user.
  Does not return todos owned by other users.
  </spec>
</module>
```
<!-- D! instruction=ignore-span-end -->

You might notice these specs are short. That's the point. A good spec
captures the rule in one or two sentences. It doesn't dictate
implementation. "The title must not be empty" is verifiable. "Check
if len(title) == 0 on line 4" is too brittle. It breaks on refactor.

## Place markers

Next, the agent wraps the implementing code with markers:

<!-- D! instruction=ignore-span-start -->
```python
# D! id=ctodo range-start
def create_todo(title, user_id):
    if not title:
        raise ValueError("title must not be empty")
    todo = {"id": len(todos) + 1, "title": title, "owner": user_id}
    todos.append(todo)
    return todo
# D! id=ctodo range-end
```
<!-- D! instruction=ignore-span-end -->

The shortcode `ctodo` contains no dot. Any text file is a valid marker
host, Python, Go, JavaScript, YAML, you name it. The scanner hashes
the lines between the markers, excluding the declarations themselves.

## Link and resolve

The agent links each marker to its spec:

```bash
$ drift link ctodo app.create_todo
Linked marker "ctodo" -> spec "app.create_todo".
```

Then it runs `drift todo`:

```bash
$ drift todo
2 closure(s) with drift.

Closure a1b2c3d4  (2 nodes: 1 specs, 1 markers; 1 edge)
  Events:
    [NODE-ADDED] spec "app.create_todo"
    [NODE-ADDED] marker "ctodo"
  ...

Closure b2c3d4e5  (2 nodes: 1 specs, 1 markers; 1 edge)
  Events:
    [NODE-ADDED] spec "app.list_todos"
    ...
```

The new specs and markers produce closures. They exist in the scan
but aren't in the baseline yet. The agent reviews each closure via
`drift diff <hash>`, confirms the spec matches the code, and runs
`drift reset <hash>` to establish the baseline.

After resolving all closures:

```bash
$ drift todo
No changes detected. 2 specs, 2 markers, 2 edges in sync.
```

Your todo app is now drift-tracked! Any future change to a spec or its
implementing code will produce a closure for the agent to review.

## What to commit to git

Commit these to git. They're shared baselines:

- `.drift/state.xml`: The baseline state
- `.drift/baselines.bin`: content snapshots
- `*.drift.xml`: spec files

The `.drift/.gitignore` file (created by `drift init`) automatically
excludes local-only files like `user-settings.xml`, `state.lock`, and
`friction.json`. You don't need to manage this manually.

Now let's add a new feature to our tracked todo app:
[How to add a feature &rarr;](04-how-to-add-a-feature.md)

<!-- D! id=docaddp range-end -->

## Try it yourself

<!-- D! id=docaddp_ex range-start -->

**Goal:** Take the todo app above (or a small project of your own).
Give drift to your agent and tell it to survey the code, write specs,
place markers, link them, and resolve the closures.

**Verify.** `drift todo` reports "No changes detected." Then edit one
function inside a marker and run `drift todo` again. You should see a
closure appear containing the changed marker and its linked spec.

<!-- D! id=docaddp_ex range-end -->
