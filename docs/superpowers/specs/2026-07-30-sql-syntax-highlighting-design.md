# SQL Syntax Highlighting Design

## Goal

Make SQL keywords and other tokens visually distinct in the Grafana query
editor, then publish the change as version 2.0.11.

## Current behavior

The query editor already uses CodeMirror's SQL parser and defines a
Grafana-aware One Dark color palette. However, `useCodeMirror` installs only
`oneDarkTheme`, which styles the editor surface without installing
`oneDarkHighlightStyle`. In addition, the dependency graph resolves separate
copies of legacy `@codemirror/language`, so the SQL parser and highlighting
engine do not share a syntax tree. As a result, parsed SQL tokens do not
receive the defined colors.

## Design

Use the existing combined `oneDark` extension in `useCodeMirror`.
`oneDark` contains both `oneDarkTheme` and `oneDarkHighlightStyle`, keeping
theme composition inside `theme.ts` and avoiding duplicated extension wiring
in the editor hook. Pin direct dependency `@codemirror/language` to `0.18.2`
so the legacy parser and highlighter share one instance; current CodeMirror 6
consumers retain their nested version.

The existing palette remains unchanged:

- SQL keywords use violet.
- strings use green.
- numbers and types use yellow.
- functions use blue.
- operators use cyan.
- comments use muted gray.

No autocomplete, formatting, SQL dialect, query execution, or editor layout
behavior changes are included.

## Regression coverage

Add a focused frontend test that renders the real editor and verifies the
`SELECT` keyword receives the palette's violet color. Existing frontend tests,
type checking, linting, and the production build must remain green.

Backend code is out of scope. Local Go vet, race tests, and multi-platform
backend builds must remain green.

## Release

Update package and plugin metadata plus the changelog to version 2.0.11.
Open an isolated pull request against the default branch, merge it after checks
pass, then tag the resulting default-branch commit as `v2.0.11`. Verify that
the tag-driven workflow publishes the expected GitHub release and archive.
