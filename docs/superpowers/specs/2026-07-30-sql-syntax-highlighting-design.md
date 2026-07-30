# SQL Syntax Highlighting Design

## Goal

Make SQL keywords and other tokens visually distinct in the Grafana query
editor, then publish the change as version 2.0.11.

## Current behavior

The query editor already uses CodeMirror's SQL parser and defines a
Grafana-aware One Dark color palette. However, `useCodeMirror` installs only
`oneDarkTheme`, which styles the editor surface without installing
`oneDarkHighlightStyle`. As a result, parsed SQL tokens do not receive the
defined colors.

## Design

Use the existing combined `oneDark` extension in `useCodeMirror`.
`oneDark` contains both `oneDarkTheme` and `oneDarkHighlightStyle`, keeping
theme composition inside `theme.ts` and avoiding duplicated extension wiring
in the editor hook.

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

Add a focused frontend test that verifies the editor installs the combined
`oneDark` extension rather than the surface-only `oneDarkTheme`. Existing
frontend tests, type checking, linting, and the production build must remain
green.

Backend code is out of scope. The local backend baseline may be skipped if the
environment cannot compile its dependency graph within the temporary-storage
quota; the hosted release workflow remains responsible for its clean build.

## Release

Update package and plugin metadata plus the changelog to version 2.0.11.
Open an isolated pull request against the default branch, merge it after checks
pass, then tag the resulting default-branch commit as `v2.0.11`. Verify that
the tag-driven workflow publishes the expected GitHub release and archive.

