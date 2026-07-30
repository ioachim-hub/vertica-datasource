# SQL Syntax Highlighting Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Activate the existing SQL token palette in the Grafana query editor and release it as version 2.0.11.

**Architecture:** Keep CodeMirror's existing SQL parser, Grafana-aware surface theme, and token palette. Change the editor hook to install the already-composed `oneDark` extension, which contains both the surface theme and token highlighting, and cover the wiring with a rendered-editor regression test.

**Tech Stack:** React 17, TypeScript, CodeMirror 6 prerelease packages, Jest, Grafana plugin Webpack tooling, GitHub Actions.

## Global Constraints

- Do not change autocomplete, formatting, SQL dialect, query execution, or editor layout.
- Do not add or upgrade dependencies.
- Keep backend code unchanged.
- Release version is exactly `2.0.11`; the Git tag is exactly `v2.0.11`.
- Preserve the user's existing dirty checkout by working only in `/tmp/vertica-datasource-sql-highlight`.

---

## File structure

- `src/UseCodeMirror.ts`: composes the SQL editor extensions and creates the editor.
- `src/UseCodeMirror.test.tsx`: renders the real editor and verifies SQL tokens receive highlighting markup.
- `package.json`: declares the release version consumed by build and packaging scripts.
- `CHANGELOG.md`: records the user-visible fix and unsigned internal-release note.

### Task 1: Activate SQL token highlighting

**Files:**

- Create: `src/UseCodeMirror.test.tsx`
- Modify: `src/UseCodeMirror.ts`

**Interfaces:**

- Consumes: `oneDark: Extension` from `src/theme.ts`.
- Produces: a CodeMirror editor whose parsed SQL tokens receive highlight classes.

- [ ] **Step 1: Write the failing rendered-editor test**

```tsx
import React from 'react';
import { act } from 'react-dom/test-utils';
import ReactDOM from 'react-dom';
import { CodeMirror } from './CodeMirror';

describe('CodeMirror SQL highlighting', () => {
  let container: HTMLDivElement;

  beforeEach(() => {
    container = document.createElement('div');
    document.body.appendChild(container);
  });

  afterEach(() => {
    act(() => ReactDOM.unmountComponentAtNode(container));
    container.remove();
  });

  it('adds highlighting markup to SQL keywords', () => {
    act(() => {
      ReactDOM.render(<CodeMirror content="SELECT 1" onContentChange={() => undefined} />, container);
    });

    const selectToken = Array.from(container.querySelectorAll('.cm-content span')).find(
      (element) => element.textContent === 'SELECT'
    );

    expect(selectToken).toBeDefined();
    expect(selectToken?.className).not.toBe('');
  });
});
```

- [ ] **Step 2: Run the focused test and verify RED**

Run:

```bash
yarn test:ci src/UseCodeMirror.test.tsx
```

Expected: FAIL because `SELECT` has no highlighted token span while only
`oneDarkTheme` is installed.

- [ ] **Step 3: Install the composed theme extension**

In `src/UseCodeMirror.ts`, replace:

```ts
import { oneDarkTheme } from './theme';
```

with:

```ts
import { oneDark } from './theme';
```

Then replace `oneDarkTheme` with `oneDark` in the `extensions` array passed to
`EditorState.create`.

- [ ] **Step 4: Run the focused test and verify GREEN**

Run:

```bash
yarn test:ci src/UseCodeMirror.test.tsx
```

Expected: PASS with one suite and one test passing.

- [ ] **Step 5: Run the complete frontend test suite**

Run:

```bash
yarn test:ci
```

Expected: PASS with both `DataSource.test.ts` and
`UseCodeMirror.test.tsx` passing.

- [ ] **Step 6: Commit the behavior and regression test**

```bash
git add src/UseCodeMirror.ts src/UseCodeMirror.test.tsx
git commit -m "fix(editor): enable SQL token colors"
```

### Task 2: Prepare version 2.0.11

**Files:**

- Modify: `package.json`
- Modify: `CHANGELOG.md`

**Interfaces:**

- Consumes: package version through the existing Webpack and release scripts.
- Produces: version `2.0.11` metadata and the first changelog section used as release notes.

- [ ] **Step 1: Verify the release checker rejects the unreleased version**

Run:

```bash
bash scripts/check-version.sh 2.0.11
```

Expected: FAIL because `package.json` still declares `2.0.10` or because no
fresh `dist/plugin.json` exists.

- [ ] **Step 2: Update package metadata**

Change the top-level `version` in `package.json`:

```json
"version": "2.0.11",
```

Do not edit `yarn.lock`; the root package version is not represented there.

- [ ] **Step 3: Add the changelog entry**

Insert directly below `# Changelog`:

```markdown
## 2.0.11 (2026-07-30)

### Fixed

- Apply the existing SQL syntax palette in the query editor so keywords,
  strings, numbers, functions, operators, and comments are visually distinct.

### Notes

- The `v2.0.11` archive is unsigned for internal Grafana installations that
  explicitly allow the plugin ID.
```

- [ ] **Step 4: Build and verify version consistency**

Run:

```bash
yarn build
bash scripts/check-version.sh 2.0.11
```

Expected: both commands exit 0 and `dist/plugin.json` reports version
`2.0.11`.

- [ ] **Step 5: Commit the release metadata**

```bash
git add package.json CHANGELOG.md
git commit -m "chore: prepare v2.0.11"
```

### Task 3: Verify and publish

**Files:**

- No additional source files.

**Interfaces:**

- Consumes: the two implementation commits on `fix/sql-syntax-highlighting`.
- Produces: a merged pull request and GitHub release tagged `v2.0.11`.

- [ ] **Step 1: Run full local verification**

Run:

```bash
yarn test:ci
yarn typecheck
yarn lint
yarn build
bash scripts/test-release-scripts.sh
bash scripts/check-version.sh 2.0.11
git diff --check origin/master...HEAD
git status --short
```

Expected: every command exits 0; Git reports no uncommitted source changes.
Backend tests remain unchanged and are left to hosted CI because the local Go
dependency graph exceeds this environment's temporary-storage quota.

- [ ] **Step 2: Push and open the pull request**

Push `fix/sql-syntax-highlighting` to `origin`, then open a pull request against
`master` with the title:

```text
fix(editor): enable SQL token colors
```

The body must summarize the editor fix and `2.0.11` release preparation, and
list the exact local verification commands.

- [ ] **Step 3: Wait for hosted checks and merge**

Inspect every required check on the pull request. Merge only after all required
checks pass, using the repository's permitted merge method. Resolve the exact
merged commit SHA from `origin/master`.

- [ ] **Step 4: Tag and release the merged commit**

Create annotated tag `v2.0.11` on the exact merged `origin/master` commit and
push that tag to `origin`. Do not tag the pre-merge feature-branch commit.

- [ ] **Step 5: Verify the release**

Wait for the tag-triggered release workflow to complete. Confirm the published
GitHub release is tagged `v2.0.11` and includes the
`rajsameer-vertica-datasource-2.0.11.zip` archive plus checksum asset.

