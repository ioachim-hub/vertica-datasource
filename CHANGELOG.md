# Changelog

## 2.0.9 (2026-07-30)

### Changed

- Updated the backend toolchain to Go 1.26.5 and refreshed backend dependencies
  for security maintenance.
- Migrated the frontend build away from deprecated `@grafana/toolkit` to the
  current Webpack-based tooling.
- Replaced legacy CI and release paths with current verification and unsigned
  release automation.

### Security

- `govulncheck` reported no vulnerability reachable from code built by
  `./...`. It reported `GO-2026-5970` for indirect `golang.org/x/text@v0.37.0`
  as a package result and `GO-2026-5942` for indirect
  `golang.org/x/net@v0.55.0` as a module result; neither was reached from the
  plugin's built code paths and neither is described as a reachable
  vulnerability.

### Notes

- No datasource behavior change is intended. The `v2.0.9` archive is unsigned
  for internal Grafana installations that explicitly allow the plugin ID.

## 1.0.0 (Unreleased)

Initial release.
