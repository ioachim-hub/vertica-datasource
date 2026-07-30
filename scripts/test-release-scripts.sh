#!/usr/bin/env bash
set -euo pipefail

repo_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
test_dir="$(mktemp -d)"
test_root="$test_dir/repo"
trap 'rm -rf "$test_dir"' EXIT

fail() {
  echo "FAIL: $*" >&2
  exit 1
}

mkdir -p "$test_root/dist" "$test_root/scripts"
cp "$repo_dir/package.json" "$test_root/package.json"
cp "$repo_dir/dist/plugin.json" "$test_root/dist/plugin.json"
cp "$repo_dir/dist/module.js" "$test_root/dist/module.js"
cp "$repo_dir/dist/gpx_vertica-datasource_linux_amd64" "$test_root/dist/gpx_vertica-datasource_linux_amd64"
cp "$repo_dir/scripts/check-semver.sh" "$test_root/scripts/check-semver.sh"
cp "$repo_dir/scripts/check-version.sh" "$test_root/scripts/check-version.sh"
cp "$repo_dir/scripts/check-plugin-metadata.sh" "$test_root/scripts/check-plugin-metadata.sh"
cp "$repo_dir/scripts/package-plugin.sh" "$test_root/scripts/package-plugin.sh"

version="$(jq -r .version "$test_root/package.json")"
invalid_version="0.0.0-invalid"
plugin_id="rajsameer-vertica-datasource"
archive="$test_root/release/$plugin_id-$version.zip"
checksum="$archive.sha256"

valid_semvers=("2.0.9" "2.0.9-rc.1" "2.0.9-alpha+001")
for candidate in "${valid_semvers[@]}"; do
  if ! bash "$test_root/scripts/check-semver.sh" "$candidate"; then
    fail "check-semver.sh rejected valid version $candidate"
  fi
done

invalid_semvers=("v2.0.9" "02.0.9" "2.00.9" "2.0.09" "2.0.9-01" "2.0.9-alpha..1" "2.0.9-alpha." "2.0.9+build..1")
for candidate in "${invalid_semvers[@]}"; do
  if bash "$test_root/scripts/check-semver.sh" "$candidate"; then
    fail "check-semver.sh accepted invalid version $candidate"
  fi
done

if bash "$test_root/scripts/check-version.sh" "$invalid_version"; then
  fail "check-version.sh accepted a mismatched version"
fi

if ! (cd "$test_root/scripts" && bash ./check-version.sh "v$version"); then
  fail "check-version.sh rejected the package version"
fi

cp "$test_root/package.json" "$test_root/package.json.original"
jq '.version = "9.9.9"' "$test_root/package.json.original" > "$test_root/package.json"
if bash "$test_root/scripts/check-version.sh" "$version"; then
  fail "check-version.sh accepted a package version mismatch"
fi
mv "$test_root/package.json.original" "$test_root/package.json"

cp "$test_root/dist/plugin.json" "$test_root/dist/plugin.json.original"
jq '.info.version = "9.9.9"' "$test_root/dist/plugin.json.original" > "$test_root/dist/plugin.json"
if bash "$test_root/scripts/check-version.sh" "$version"; then
  fail "check-version.sh accepted a manifest version mismatch"
fi
mv "$test_root/dist/plugin.json.original" "$test_root/dist/plugin.json"

if bash "$test_root/scripts/package-plugin.sh" "$invalid_version"; then
  fail "package-plugin.sh accepted a mismatched version"
fi
test ! -e "$test_root/release/$plugin_id-$invalid_version.zip" || fail "mismatched version produced an archive"

stale_entry="$plugin_id/removed-from-dist.txt"
printf 'stale archive fixture\n' > "$test_root/dist/removed-from-dist.txt"
(cd "$test_root/scripts" && bash ./package-plugin.sh "$version")
test -f "$archive" || fail "release archive was not created"
test -f "$checksum" || fail "release checksum was not created"
(cd "$test_root/release" && sha256sum -c "$(basename "$checksum")") >/dev/null
unzip -Z1 "$archive" | grep -Fx "$stale_entry" >/dev/null || fail "stale archive fixture was not packaged"

rm -f "$test_root/dist/removed-from-dist.txt"
(cd "$test_root/scripts" && bash ./package-plugin.sh "$version")
if unzip -Z1 "$archive" | grep -Fx "$stale_entry" >/dev/null; then
  fail "repackaging retained a file removed from dist"
fi

entries="$(unzip -Z1 "$archive")"
printf '%s\n' "$entries" | awk -F/ -v plugin_id="$plugin_id" '
  BEGIN { valid = 1 }
  $1 != plugin_id { valid = 0 }
  END { exit !(NR > 0 && valid != 0) }
' || fail "archive does not contain exactly one top-level plugin directory"
printf '%s\n' "$entries" | grep -Fx "$plugin_id/plugin.json" >/dev/null || fail "archive is missing plugin.json"
printf '%s\n' "$entries" | grep -Fx "$plugin_id/module.js" >/dev/null || fail "archive is missing module.js"
printf '%s\n' "$entries" | grep -Eq "^$plugin_id/gpx_vertica-datasource" || fail "archive is missing backend binaries"

rm -f "$archive" "$checksum"
external_target="$test_dir/external-target"
touch -t 200001010000 "$external_target"
target_before="$(stat -c '%a:%Y' "$external_target")"
ln -s "$external_target" "$test_root/dist/external-link"

if bash "$test_root/scripts/package-plugin.sh" "$version"; then
  symlink_rejected=false
else
  symlink_rejected=true
fi
test "$target_before" = "$(stat -c '%a:%Y' "$external_target")" || fail "symlink target was mutated"
test ! -e "$archive" || fail "symlink rejection produced an archive"
test "$symlink_rejected" = true || fail "package-plugin.sh accepted a dist symlink"

echo "release script tests passed"
