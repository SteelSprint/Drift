# Release instructions

Releases are triggered by git tags. There is no version file in the
repository — the version is the tag name.

## Prerequisites

- All tests pass: `go test ./...`
- All checks pass: `filament check`
- No uncommitted changes: `git status` is clean

## Steps

1. Determine the next version number (e.g. `v0.1.3`). Use semver:

   - **Patch** (`v0.1.x`): bug fixes, script hardening, docs.
   - **Minor** (`v0.x.0`): new features, new spec clauses, new commands.
   - **Major** (`vx.0.0`): breaking changes to the spec, marker format,
     or state file.

2. Tag the commit:

   ```
   git tag v0.1.3
   ```

   Do not use annotated tags (`-a`) unless you want to add a message.
   The release workflow fires on any tag matching `v*`.

3. Push the tag:

   ```
   git push origin v0.1.3
   ```

4. The GitHub Actions release workflow (`.github/workflows/release.yml`)
   fires automatically on the pushed tag. It uses goreleaser to:

   - Build binaries for linux, darwin, and windows on amd64 and arm64.
   - Package archives (tar.gz for unix, zip for windows).
   - Generate `checksums.txt`.
   - Create a GitHub Release with the archive and checksum files attached.
   - Generate a changelog from commit messages (excluding `docs:` and
     `test:` prefixes).

5. Verify the release on GitHub:

   - Check that the GitHub Release exists and all assets are attached.
   - Download `checksums.txt` and one archive to confirm the release is
     complete.
   - Run the install script on a clean machine to verify it works
     end-to-end:

     ```
     curl -fsSL https://raw.githubusercontent.com/steelsprint/filament/main/scripts/install.sh | bash
     ```

6. If something went wrong, delete the tag and the release:

   ```
   git tag -d v0.1.3
   git push origin :refs/tags/v0.1.3
   ```

   Then delete the GitHub Release from the Releases page, fix the issue,
   and re-tag from the corrected commit.

## Notes

- The version is never stored in a file. It is always the git tag.
- The install scripts fetch the latest release from the GitHub API, so
  no code change is needed to ship a new version — only a new tag.
- goreleaser config is in `.goreleaser.yml`.
- Do not re-tag a version that has already been published. If a release
  is broken, cut a new patch version instead.
