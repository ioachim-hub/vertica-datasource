# Maintainer guide

## Scope and preservation rules

This repository is the maintained `ioachim-hub/vertica-datasource` fork of
[`rajsameer/vertica-datasource`](https://github.com/rajsameer/vertica-datasource).
Its purpose is to keep the plugin buildable and releasable while applying
security and toolchain maintenance. Preserve the datasource's runtime contract:

- plugin ID: `rajsameer-vertica-datasource`;
- backend executable: `gpx_vertica-datasource`;
- datasource settings and secure settings, query model, and response behavior;
- the ZIP's single top-level `rajsameer-vertica-datasource/` directory.

Do not use a maintenance update to redesign the datasource or add compatibility
layers for toolchains no longer supported by this fork.

## Repository map

- `src/`: frontend code and source `plugin.json` manifest.
- `pkg/`: Go backend and its contract tests.
- `.config/`: Webpack configuration that assembles `dist/`.
- `scripts/`: reusable metadata, version, and package checks.
- `.github/workflows/`: CI and unsigned release automation.
- `release/`: generated ZIP and SHA-256 checksum; do not edit its artifacts by
  hand.
- `README.md` and `CHANGELOG.md`: supported operator and release information.

## Supported toolchain

Use Go `1.26.5`, Node `24` (from `.nvmrc`), and Yarn `1.22.22` (from the
`packageManager` field). Yarn and `yarn.lock` are the only supported frontend
package-manager and lockfile authority. Do not add or regenerate
`package-lock.json`.

## Local verification

Run these commands from the repository root before proposing a maintenance or
release change:

```bash
yarn install --frozen-lockfile
yarn lint
yarn typecheck
yarn test:ci
yarn build
bash scripts/check-plugin-metadata.sh dist/plugin.json
gofmt -d Magefile.go pkg
go vet ./...
go test -race ./...
go run github.com/magefile/mage@v1.15.0 -v buildAll
go run golang.org/x/vuln/cmd/govulncheck@v1.6.0 ./...
```

`gofmt -d Magefile.go pkg` must produce no output. The final Mage command
builds the supported backend targets; do not replace it with a floating
`go install` or an unpinned Mage invocation.

### Vulnerability triage

Treat a `govulncheck` finding reachable from code built by `./...` as a release
blocker until it is fixed or the release owner explicitly accepts the risk.
Record the vulnerable symbol, dependency, affected build path, mitigation, and
re-run result. A module-level or dependency-only advisory is not evidence of a
reachable vulnerability: record its reachability analysis and follow-up, but do
not describe it as reachable. In the `v2.0.9` review, `govulncheck` reported
no symbol result: `GO-2026-5970` in indirect `golang.org/x/text@v0.37.0` was a
package result and `GO-2026-5942` in indirect `golang.org/x/net@v0.55.0` was a
module result. Neither was reached from the plugin's built code paths.

## Versioning, packaging, and unsigned releases

Set the release version once in `package.json`. The frontend build propagates it
to `dist/plugin.json`; `scripts/check-version.sh` verifies the requested tag,
`package.json`, and built manifest agree. For `v2.0.9`, use:

```bash
bash scripts/check-version.sh v2.0.9
bash scripts/package-plugin.sh v2.0.9
(cd release && sha256sum -c rajsameer-vertica-datasource-2.0.9.zip.sha256)
unzip -t release/rajsameer-vertica-datasource-2.0.9.zip
go run github.com/grafana/plugin-validator/pkg/cmd/plugincheck2@v0.41.0 release/rajsameer-vertica-datasource-2.0.9.zip
```

`scripts/package-plugin.sh` creates the deterministic unsigned archive
`release/rajsameer-vertica-datasource-2.0.9.zip` and its
`release/rajsameer-vertica-datasource-2.0.9.zip.sha256` checksum. Always run
`sha256sum -c` from `release/`, as shown above, before distributing the archive.
The archive must contain the one plugin directory and the expected frontend and
backend files. Do not introduce MD5 checksums or Grafana signing credentials.

The release workflow runs for any pushed tag matching `v*`. The tag must still
be a supported semantic version and match `package.json`, `dist/plugin.json`,
and an extractable changelog section before the publish job creates an unsigned
GitHub release. A manual workflow dispatch runs the same release gates: its
requested version (default `2.0.9`) must be a supported semantic version, match
`package.json` and the built `dist/plugin.json`, and have an exact matching
`CHANGELOG.md` section. Only after those checks does it package and upload the
artifacts; the tag-only GitHub release publication is skipped.

For the exact `v2.0.9` procedure, after all local checks pass and an authorized
reviewer approves the release:

1. Confirm the `2.0.9` changelog section is present and extractable by the
   release workflow.
2. Commit, push, and create/push tag `v2.0.9` only with explicit authorization.
3. Review the workflow's uploaded ZIP, SHA-256 file, and generated release
   notes before making the release available internally.

## Worktree safety

Assume the worktree can already be dirty. Inspect `git status --short` and the
relevant diff before editing; preserve unrelated changes, generated archives,
and user-authored documentation. Never commit, push, tag, publish a release, or
delete user files without explicit authorization. Do not claim that a Grafana or
Vertica smoke test ran unless its result is recorded from a real target
environment.
