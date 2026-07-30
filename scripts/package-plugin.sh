#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_dir="$(cd "$script_dir/.." && pwd)"
version="${1#v}"
plugin_id="rajsameer-vertica-datasource"
release_dir="$repo_dir/release"
archive="$release_dir/$plugin_id-$version.zip"
staging_dir="$(mktemp -d)"
trap 'rm -rf "$staging_dir"' EXIT

test -d "$repo_dir/dist"
if [[ -n "$(find "$repo_dir/dist" -type l -print -quit)" ]]; then
  echo "dist must not contain symlinks" >&2
  exit 1
fi
test -f "$repo_dir/dist/plugin.json"
bash "$script_dir/check-version.sh" "$version"
bash "$script_dir/check-plugin-metadata.sh" "$repo_dir/dist/plugin.json"

mkdir -p "$release_dir"
archive_staging_dir="$(mktemp -d "$release_dir/.package-$plugin_id-$version.XXXXXX")"
temporary_archive="$archive_staging_dir/$(basename "$archive")"
temporary_checksum="$temporary_archive.sha256"
trap 'rm -rf "$staging_dir" "$archive_staging_dir"' EXIT

cp -R "$repo_dir/dist" "$staging_dir/$plugin_id"
find "$staging_dir/$plugin_id" -type d -exec chmod 0755 {} +
find "$staging_dir/$plugin_id" -type f -exec chmod 0644 {} +
find "$staging_dir/$plugin_id" -type f -name 'gpx_vertica-datasource*' -exec chmod 0755 {} +
find "$staging_dir/$plugin_id" -exec touch -t 198001010000 {} +

(
  cd "$staging_dir"
  LC_ALL=C find "$plugin_id" -print | LC_ALL=C sort | zip -X -q "$temporary_archive" -@
)
(
  cd "$archive_staging_dir"
  sha256sum "$(basename "$temporary_archive")" > "$(basename "$temporary_checksum")"
)
mv -f "$temporary_archive" "$archive"
mv -f "$temporary_checksum" "$archive.sha256"
