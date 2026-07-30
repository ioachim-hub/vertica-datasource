#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_dir="$(cd "$script_dir/.." && pwd)"
version="${1#v}"
bash "$script_dir/check-semver.sh" "$version"
package_version="$(jq -r .version "$repo_dir/package.json")"
plugin_version="$(jq -r .info.version "$repo_dir/dist/plugin.json")"

test "$version" = "$package_version"
test "$version" = "$plugin_version"
