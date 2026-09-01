#!/usr/bin/env bash

set -euo pipefail

head_ref="${2:-HEAD}"
base_ref="${1:-}"

if [[ -z "$base_ref" ]]; then
  base_ref="$(git describe --tags --match 'v[0-9]*' --abbrev=0 "$head_ref")"
fi

repo_root="$(git rev-parse --show-toplevel)"
old_commit="$(git rev-parse --verify "${base_ref}^{commit}")"
new_commit="$(git rev-parse --verify "${head_ref}^{commit}")"

if [[ "$old_commit" == "$new_commit" ]]; then
  printf 'git revision: unchanged at %s\n' "$new_commit" >&2
  printf 'none\n'
  exit 0
fi

versions="$({
  # shellcheck disable=SC2016
  REPO_ROOT="$repo_root" OLD_COMMIT="$old_commit" NEW_COMMIT="$new_commit" \
    nix eval --impure --json --expr '
    let
      system = "x86_64-linux";
      repository = builtins.getEnv "REPO_ROOT";
      packages = commit:
        (builtins.getFlake "git+file://${repository}?rev=${commit}").outputs.nixpkgs.${system};
      old = packages (builtins.getEnv "OLD_COMMIT");
      new = packages (builtins.getEnv "NEW_COMMIT");
      comparison = oldVersion: newVersion: {
        old = oldVersion;
        new = newVersion;
        increased = builtins.compareVersions oldVersion newVersion < 0;
      };
    in
    {
      ffmpeg = comparison old.ffmpeg.version new.ffmpeg.version;
      "yt-dlp" = comparison old."yt-dlp".version new."yt-dlp".version;
    }
  '
})"

printf 'runtime dependency versions from %s to %s:\n' "$base_ref" "$head_ref" >&2
jq --raw-output 'to_entries[] | "  \(.key): \(.value.old) -> \(.value.new)"' <<<"$versions" >&2

if jq --exit-status 'any(.[]; .increased)' <<<"$versions" >/dev/null; then
  printf 'minor\n'
else
  printf 'none\n'
fi
