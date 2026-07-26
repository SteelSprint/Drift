# Prior art

I built this before I found any of the projects below. Several people arrived at this problem independently and a couple
shipped before I did, so they're listed here with an honest account of where
each approach wins. Descriptions come from published docs rather than a
line-by-line audit; open an issue if I've got yours wrong, or if work is missing.

---

## Closest neighbors

### [fiberplane/drift](https://github.com/fiberplane/drift)

Same name, same problem; ([their write-up](https://fiberplane.com/blog/drift-documentation-linter/)). It
also anchors doc files to code targets, stamps a content signature, and fails CI when
the target changed since the doc was last confirmed.

### [bgervin/spec-kit-sync](https://github.com/bgervin/spec-kit-sync)

A Spec Kit extension that reports requirements as aligned, drifted, or
unverifiable, and backfills specs for code that never had them, which this has
no equivalent for. The tradeoff is that theirs is an analysis pass with
percentages and a severity judgment, where this is a binary seal returning a
deterministic affected set, which is the shape a CI gate and an unattended agent
loop can both act on without interpretation. Theirs also requires adopting Spec
Kit; this is standalone.

---

## Adjacent work

Same problem space, different angle:

- **[github/spec-kit](https://github.com/github/spec-kit)**: the spec-driven
  development framework a lot of this orbits. The
  [living specs discussion](https://github.com/github/spec-kit/discussions/152)
  and [session-continuity thread](https://github.com/github/spec-kit/discussions/1671)
  are worth reading; people have been circling this for a while.
- **[Kiro issue #9435](https://github.com/kirodotdev/Kiro/issues/9435)**:
  proposes recording a git ref in spec metadata to detect whether code moved
  under a spec between sessions. VCS history rather than content hashing, same
  failure mode.
- **[spec-driver](https://davidlee.github.io/spec-driver/)**: stubs specs for
  unspecified modules, prunes orphaned specs, audits contract drift. Explicitly
  targets legacy codebases.
- **Carrot**: MCP server for Cursor that auto-writes specs and AST-validates
  commits against them.

---

## A note on the word "drift"

It's overloaded. In current usage it also refers to ML model and output drift
monitoring, infrastructure configuration drift, API schema drift, and multi-turn
agent behavioral drift, none of which is this problem. If you arrived here
looking for one of those, you want a different tool.