#!/usr/bin/env bash
set -euo pipefail

plugin_json="${1:-src/plugin.json}"
jq -e '
  .id == "rajsameer-vertica-datasource" and
  .type == "datasource" and
  .backend == true and
  .executable == "gpx_vertica-datasource"
' "$plugin_json" >/dev/null
