# Driftpin

Driftpin helps LLMs keep specs and code in sync.

## Quickstart

1. `make build`
2. Run `./drift init` in your project
3. Edit `main.pin.xml` to add your specs
4. Place `D! id=<markerid>` markers in your code above the implementations
5. Run `./drift link <marker> <module.spec>` to connect them
6. Run `./drift todo` to check for drift
7. Run `./drift list` to inspect specs, markers, links, and sync state
8. Run `./drift reset <marker> <module.spec>` to resolve drift
9. Run `./drift unlink <marker> <module.spec>` to remove a bad link

## Self-discovery

- `./drift help` — command reference with examples
- `./drift skill` — comprehensive guide for LLM agents (pipe to context)

## Anatomy

- **Specs** — `*.pin.xml` files containing `<spec id="...">` elements under `<main>` or `<module name="...">` roots
- **Markers** — `D! id=<shortcode>` comment lines in code files, placed above the implementing code
- **drift.pin** — XML state file at project root storing baseline hashes, links, and resolution state. Tool-managed — do not edit by hand. Commit to git.
- **CLI** — `drift init`, `drift todo`, `drift list`, `drift link`, `drift unlink`, `drift reset`, `drift help`, `drift skill`

See [DOCUMENTATION.md](DOCUMENTATION.md) for the full documentation.
