# Vertica Datasource Security Maintenance and Release Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Modernize the maintained Vertica datasource fork, verify that its runtime contract is unchanged, and provide a repeatable unsigned `v2.0.9` GitHub release.

**Architecture:** Preserve the existing datasource implementation while replacing its obsolete build shell: current Go and Grafana SDK dependencies for the backend, current Grafana Webpack tooling for the frontend, shared verification scripts for CI and releases, and one tag-driven unsigned release workflow. Release metadata and archive validation are kept in small repository scripts so the same checks run locally and in GitHub Actions.

**Tech Stack:** Go 1.26.5, `grafana-plugin-sdk-go` v0.294.0, `vertica-sql-go` v1.3.8, Node 24 LTS, Yarn 1.22.22, Grafana packages 13.1.1, `@grafana/create-plugin` 7.9.0 migration tooling, Mage, GitHub Actions, Bash, `govulncheck`, Grafana plugin validator.

## Global Constraints

- Preserve plugin ID `rajsameer-vertica-datasource`.
- Preserve backend executable `gpx_vertica-datasource`.
- Preserve datasource settings, secure settings, query model, and response behavior.
- Keep the release unsigned and independent of Grafana Cloud credentials.
- Package `rajsameer-vertica-datasource/` as the ZIP's single top-level directory.
- Use Go 1.26.5 consistently.
- Use Yarn as the only frontend package manager and lockfile authority.
- Do not overwrite the existing untracked `vertica-datasource-2.0.8-rc*` archives.
- Do not commit without explicit user authorization.

---

## File Structure

- `go.mod`, `go.sum`: Go language/toolchain and resolved backend dependency graph.
- `package.json`, `yarn.lock`: frontend scripts and single dependency lock.
- `.nvmrc`: pinned Node major for local and CI builds.
- `.config/webpack/webpack.config.ts`: Grafana-supported frontend build configuration.
- `.config/webpack/constants.ts`: version replacement for `%VERSION%` and `%TODAY%`.
- `.eslintrc`, `.prettierrc.js`, `jest.config.js`, `jest-setup.js`, `tsconfig.json`: current frontend checks.
- `scripts/check-version.sh`: checks tag, package, and built plugin versions.
- `scripts/package-plugin.sh`: validates and creates the deterministic unsigned archive and checksum.
- `.github/workflows/ci.yml`: frontend/backend/security/plugin validation.
- `.github/workflows/release.yml`: manual packaging and tag-triggered GitHub release.
- `.github/workflows/release_unsigned.yml`: removed after its behavior is covered by `release.yml`.
- `AGENTS.md`: maintenance contract and copy-ready developer/release commands.
- `README.md`, `CHANGELOG.md`: supported build/install flow and `v2.0.9` release notes.

### Task 1: Protect the existing plugin contract with metadata and backend tests

**Files:**
- Create: `pkg/contract_test.go`
- Create: `scripts/check-plugin-metadata.sh`
- Test: `pkg/contract_test.go`
- Test: `scripts/check-plugin-metadata.sh`

**Interfaces:**
- Consumes: current `src/plugin.json`, `newDatasource()`, and existing query/config types.
- Produces: stable metadata assertions reused before and after the migration.

- [ ] **Step 1: Record the current dirty state without modifying it**

Run:

```bash
rtk git status --short
rtk git diff -- README.md go.mod go.sum
```

Expected: the pre-existing README and Go dependency edits plus the three RC archives are visible.

- [ ] **Step 2: Write a failing metadata check**

Create `scripts/check-plugin-metadata.sh`:

```bash
#!/usr/bin/env bash
set -euo pipefail

plugin_json="${1:-src/plugin.json}"
jq -e '
  .id == "rajsameer-vertica-datasource" and
  .type == "datasource" and
  .backend == true and
  .executable == "gpx_vertica-datasource"
' "$plugin_json" >/dev/null
```

Run:

```bash
rtk bash scripts/check-plugin-metadata.sh /dev/null
```

Expected: FAIL because `/dev/null` is not valid plugin JSON.

- [ ] **Step 3: Run the metadata check against the source manifest**

Run:

```bash
rtk bash scripts/check-plugin-metadata.sh src/plugin.json
```

Expected: PASS.

- [ ] **Step 4: Add focused Go contract tests**

Create `pkg/contract_test.go` with tests in package `main` that assert:

```go
func TestNewDatasourceImplementsRequiredHandlers(t *testing.T) {
	ds := newDatasource()
	if ds == nil {
		t.Fatal("newDatasource returned nil")
	}
	var _ backend.QueryDataHandler = ds
	var _ backend.CheckHealthHandler = ds
}
```

Add representative JSON-unmarshal tests for the existing datasource settings and query structs using the exact JSON field names currently defined in `pkg/type.go`.

- [ ] **Step 5: Verify the tests on the pre-migration code**

Run:

```bash
rtk go test ./pkg/...
```

Expected: PASS. If an SDK interface assertion does not match the current implementation, assert the concrete handler method signatures instead; do not add adapter behavior.

### Task 2: Upgrade and verify the Go backend

**Files:**
- Modify: `go.mod`
- Modify: `go.sum`
- Modify only if required by real SDK compile errors: `Magefile.go`
- Modify only if required by real SDK compile errors: `pkg/*.go`
- Test: `pkg/contract_test.go`

**Interfaces:**
- Consumes: the Task 1 contract tests.
- Produces: backend binaries compiled by Go 1.26.5 with SDK v0.294.0.

- [ ] **Step 1: Set the supported Go version and direct dependencies**

Run:

```bash
rtk go mod edit -go=1.26.5
rtk go get github.com/grafana/grafana-plugin-sdk-go@v0.294.0
rtk go get github.com/vertica/vertica-sql-go@v1.3.8
rtk go mod tidy
```

Expected: `go.mod` contains only tool-derived indirect dependencies; no manually pinned transitive dependency is added solely to silence a scanner.

- [ ] **Step 2: Run the tests and capture real SDK incompatibilities**

Run:

```bash
rtk go test ./...
```

Expected: PASS, or compile failures naming exact changed SDK interfaces.

- [ ] **Step 3: Make only required compatibility edits**

For each compile failure, update the affected `pkg/*.go` call site to the current SDK signature while preserving request and response behavior. Do not add compatibility wrappers for SDK versions no longer used.

- [ ] **Step 4: Run backend quality gates**

Run:

```bash
rtk gofmt -w Magefile.go pkg
rtk gofmt -d Magefile.go pkg
rtk go vet ./...
rtk go test -race ./...
rtk go run github.com/magefile/mage@v1.15.0 -v buildAll
```

Expected: no formatting diff, vet errors, test failures, or build failures.

- [ ] **Step 5: Run vulnerability analysis**

Run:

```bash
rtk go run golang.org/x/vuln/cmd/govulncheck@latest ./...
```

Expected: no reachable vulnerability. Record any informational dependency-only finding in the release notes with reachability and mitigation.

- [ ] **Step 6: Inspect the built binaries**

Run:

```bash
rtk find dist -maxdepth 1 -type f -name 'gpx_vertica-datasource*' -exec go version -m {} \;
```

Expected: each backend binary reports Go 1.26.5 and the intended Grafana SDK and Vertica driver versions.

### Task 3: Migrate the frontend from `@grafana/toolkit`

**Files:**
- Modify: `package.json`
- Modify: `yarn.lock`
- Delete: `package-lock.json`
- Create: `.nvmrc`
- Create: `.config/webpack/webpack.config.ts`
- Create: `.config/webpack/constants.ts`
- Create or replace: `.eslintrc`
- Create or replace: `.prettierrc.js`
- Create or replace: `jest.config.js`
- Create: `jest-setup.js`
- Modify: `tsconfig.json`
- Modify only for current API compatibility: `src/*.ts`, `src/*.tsx`

**Interfaces:**
- Consumes: `src/plugin.json`, React datasource components, and Task 1 metadata checks.
- Produces: a current production frontend in `dist/` with substituted version metadata.

- [ ] **Step 1: Generate a disposable migration reference**

Run from a temporary copy, not the working tree:

```bash
tmp_dir="$(mktemp -d)"
cp -R . "$tmp_dir/plugin"
cd "$tmp_dir/plugin"
npx @grafana/create-plugin@7.9.0 migrate
```

Expected: a reviewable reference showing the current Grafana configuration files. Do not accept deletion or replacement prompts in the real checkout.

- [ ] **Step 2: Replace deprecated scripts and dependencies**

Update `package.json` to:

```json
{
  "scripts": {
    "build": "webpack --config .config/webpack/webpack.config.ts --env production",
    "dev": "webpack --config .config/webpack/webpack.config.ts --env development",
    "lint": "eslint src --cache",
    "lint:fix": "yarn lint --fix",
    "typecheck": "tsc --noEmit",
    "test": "jest",
    "test:ci": "jest --ci --runInBand"
  },
  "engines": {
    "node": ">=24"
  }
}
```

Use the generated migration reference for the exact Webpack/Jest/ESLint development dependencies. Set `@grafana/data`, `@grafana/runtime`, and `@grafana/ui` to `13.1.1`. Remove `@grafana/toolkit` and the obsolete `sign` script.

- [ ] **Step 3: Pin Node and regenerate the single lockfile**

Create `.nvmrc`:

```text
24
```

Run:

```bash
rtk yarn install
```

After the Yarn build is verified, delete `package-lock.json` with `apply_patch`.

- [ ] **Step 4: Run the migrated frontend checks**

Run:

```bash
rtk yarn lint
rtk yarn typecheck
rtk yarn test:ci
rtk yarn build
rtk bash scripts/check-plugin-metadata.sh dist/plugin.json
```

Expected: all checks pass and `dist/plugin.json` contains real version/date values rather than `%VERSION%` or `%TODAY%`.

- [ ] **Step 5: Fix only real Grafana API incompatibilities**

For each TypeScript or runtime API error, update the smallest affected component. Keep the `VerticaQuery` and `VerticaDataSourceOptions` shapes stable and do not hide failures with `any`, blanket casts, disabled lint rules, or skipped tests.

### Task 4: Add reusable release checks and packaging

**Files:**
- Create: `scripts/check-version.sh`
- Create: `scripts/package-plugin.sh`
- Modify: `package.json`
- Modify: `src/plugin.json`
- Test: shell scripts against invalid and valid inputs.

**Interfaces:**
- Consumes: built `dist/`, semantic version argument, package metadata.
- Produces: `release/rajsameer-vertica-datasource-<version>.zip` and `.sha256`.

- [ ] **Step 1: Write a failing version mismatch check**

Create `scripts/check-version.sh`:

```bash
#!/usr/bin/env bash
set -euo pipefail

version="${1#v}"
package_version="$(jq -r .version package.json)"
plugin_version="$(jq -r .info.version dist/plugin.json)"

test "$version" = "$package_version"
test "$version" = "$plugin_version"
```

Run:

```bash
rtk bash scripts/check-version.sh v0.0.0
```

Expected: FAIL.

- [ ] **Step 2: Make package metadata use version `2.0.9`**

Set `package.json` version to `2.0.9`. Ensure the Webpack constants substitute this value into the built `plugin.json`.

Run:

```bash
rtk yarn build
rtk bash scripts/check-version.sh v2.0.9
```

Expected: PASS.

- [ ] **Step 3: Implement archive creation**

Create `scripts/package-plugin.sh`:

```bash
#!/usr/bin/env bash
set -euo pipefail

version="${1#v}"
plugin_id="rajsameer-vertica-datasource"
release_dir="release"
staging_dir="$(mktemp -d)"
trap 'rm -rf "$staging_dir"' EXIT

test -f dist/plugin.json
bash scripts/check-version.sh "$version"
bash scripts/check-plugin-metadata.sh dist/plugin.json
mkdir -p "$release_dir"
cp -R dist "$staging_dir/$plugin_id"
find "$staging_dir/$plugin_id" -type f -name 'gpx_vertica-datasource*' -exec chmod 0755 {} +
(cd "$staging_dir" && zip -qr "$OLDPWD/$release_dir/$plugin_id-$version.zip" "$plugin_id")
sha256sum "$release_dir/$plugin_id-$version.zip" > "$release_dir/$plugin_id-$version.zip.sha256"
```

- [ ] **Step 4: Verify rejection and success paths**

Run:

```bash
rtk bash scripts/package-plugin.sh 0.0.0
```

Expected: FAIL before producing an archive.

Run:

```bash
rtk bash scripts/package-plugin.sh 2.0.9
rtk unzip -l release/rajsameer-vertica-datasource-2.0.9.zip
rtk sha256sum -c release/rajsameer-vertica-datasource-2.0.9.zip.sha256
```

Expected: one top-level plugin directory, expected frontend files and backend binaries, and a valid checksum.

### Task 5: Replace legacy CI and release workflows

**Files:**
- Modify: `.github/workflows/ci.yml`
- Modify: `.github/workflows/release.yml`
- Delete: `.github/workflows/release_unsigned.yml`

**Interfaces:**
- Consumes: repository scripts and package-manager commands from Tasks 2–4.
- Produces: PR verification, manual release artifacts, and tag-triggered unsigned GitHub releases.

- [ ] **Step 1: Replace CI with maintained actions**

Configure `.github/workflows/ci.yml` with:

```yaml
permissions:
  contents: read

steps:
  - uses: actions/checkout@v4
  - uses: actions/setup-node@v4
    with:
      node-version-file: .nvmrc
      cache: yarn
  - uses: actions/setup-go@v6
    with:
      go-version: 1.26.5
      cache: true
  - run: yarn install --frozen-lockfile
  - run: yarn lint
  - run: yarn typecheck
  - run: yarn test:ci
  - run: yarn build
  - run: go vet ./...
  - run: go test -race ./...
  - run: go run github.com/magefile/mage@v1.15.0 -v buildAll
  - run: go run golang.org/x/vuln/cmd/govulncheck@latest ./...
  - run: bash scripts/check-plugin-metadata.sh dist/plugin.json
```

Pin `govulncheck` to the resolved tested version before committing rather than leaving `@latest` in the final workflow.

- [ ] **Step 2: Implement one unsigned release workflow**

Configure `.github/workflows/release.yml` for `workflow_dispatch` and tags `v*`, with `contents: write`. Reuse the same verification commands, run:

```bash
bash scripts/check-version.sh "${GITHUB_REF_NAME:-2.0.9}"
bash scripts/package-plugin.sh "${GITHUB_REF_NAME:-2.0.9}"
```

Upload `release/*` with `actions/upload-artifact@v4`. On a tag only, create the GitHub release with:

```yaml
- uses: softprops/action-gh-release@v2
  if: startsWith(github.ref, 'refs/tags/')
  with:
    files: release/*
    body_path: release-notes.md
    prerelease: ${{ contains(github.ref_name, '-') }}
```

- [ ] **Step 3: Remove obsolete release behavior**

Delete `.github/workflows/release_unsigned.yml`. Confirm no workflow contains `actions/create-release`, `actions/upload-release-asset`, `::set-output`, Node 14, Go 1.15/1.16, MD5 checksums, or Grafana signing secrets.

Run:

```bash
rtk rg -n 'create-release|upload-release-asset|set-output|node-version:.*14|go-version:.*1\\.(15|16)|md5|GRAFANA_.*TOKEN|GRAFANA_API_KEY' .github
```

Expected: no matches.

- [ ] **Step 4: Validate workflow syntax**

Run:

```bash
rtk npx --yes yaml-lint .github/workflows/ci.yml .github/workflows/release.yml
```

Expected: PASS.

### Task 6: Document the maintenance and release contract

**Files:**
- Create: `AGENTS.md`
- Modify: `README.md`
- Modify: `CHANGELOG.md`

**Interfaces:**
- Consumes: exact verified commands and release filenames from Tasks 2–5.
- Produces: repository guidance for future security-only updates.

- [ ] **Step 1: Add `AGENTS.md`**

Document:

- fork purpose and upstream;
- behavior-preservation boundaries;
- directory map;
- Go 1.26.5, Node 24, Yarn 1.22.22;
- exact frontend/backend verification commands;
- `govulncheck` triage rules;
- version propagation and archive checks;
- unsigned `v2.0.9` release procedure;
- preservation of dirty worktrees and explicit-commit authorization.

- [ ] **Step 2: Correct the README build and installation instructions**

Replace the OpenSSL legacy-provider workaround and floating Mage installation with:

```bash
yarn install --frozen-lockfile
yarn build
go test ./...
go run github.com/magefile/mage@v1.15.0 -v buildAll
```

Document internal unsigned installation and the required Grafana
`allow_loading_unsigned_plugins = rajsameer-vertica-datasource` setting.

- [ ] **Step 3: Add the `v2.0.9` changelog entry**

Add a dated `2.0.9` section describing:

- Go 1.26.5 and backend dependency security updates;
- migration away from deprecated `@grafana/toolkit`;
- current CI and unsigned release automation;
- no intended datasource behavior change.

- [ ] **Step 4: Verify documentation commands and references**

Run every documented local command and:

```bash
rtk rg -n '1\\.15|1\\.16|Node 14|grafana-toolkit|openssl-legacy-provider|md5|unsigned-v' README.md AGENTS.md .github package.json
```

Expected: matches only where historical context explicitly explains removed tooling.

### Task 7: End-to-end release candidate verification

**Files:**
- Modify only for verified defects: files introduced or changed in Tasks 1–6.
- Produce locally but do not commit: `release/rajsameer-vertica-datasource-2.0.9.zip`
- Produce locally but do not commit: `release/rajsameer-vertica-datasource-2.0.9.zip.sha256`

**Interfaces:**
- Consumes: the complete migrated repository.
- Produces: release evidence and a ready-to-tag tree.

- [ ] **Step 1: Run the complete clean-build sequence**

Run:

```bash
rtk yarn install --frozen-lockfile
rtk yarn lint
rtk yarn typecheck
rtk yarn test:ci
rtk yarn build
rtk gofmt -d Magefile.go pkg
rtk go vet ./...
rtk go test -race ./...
rtk go run github.com/magefile/mage@v1.15.0 -v buildAll
rtk go run golang.org/x/vuln/cmd/govulncheck@latest ./...
rtk bash scripts/check-plugin-metadata.sh dist/plugin.json
rtk bash scripts/package-plugin.sh 2.0.9
rtk sha256sum -c release/rajsameer-vertica-datasource-2.0.9.zip.sha256
```

Expected: every command passes.

- [ ] **Step 2: Validate the packaged plugin**

Run Grafana's current plugin validator against the ZIP using the pinned
validator version selected during implementation.

Expected: no errors; any warning is listed verbatim in the handoff.

- [ ] **Step 3: Smoke-test with Grafana and Vertica**

Install the ZIP into the internal test Grafana with unsigned loading enabled.
Verify:

- the datasource plugin loads;
- saved datasource configuration can be opened;
- health check succeeds;
- one representative table query succeeds;
- one representative time-series query succeeds;
- returned fields and timestamps match the current plugin.

- [ ] **Step 4: Review the final diff and release inputs**

Run:

```bash
rtk git diff --check
rtk git status --short
rtk git diff --stat
rtk git diff
```

Expected: no accidental changes to the old RC archives and no generated
`dist/`, `release/`, or `node_modules/` content staged for commit.

- [ ] **Step 5: Stop before commit/tag/release**

Present the exact test results, remaining warnings, proposed commit message,
tag `v2.0.9`, ZIP name, and checksum. Obtain explicit authorization before
committing, pushing, tagging, or creating the GitHub release.
