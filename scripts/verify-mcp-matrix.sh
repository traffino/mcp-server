#!/usr/bin/env bash
# Verifies the three places that must stay in sync when adding an MCP server:
#   1. cmd/<name>/        — Go binary source
#   2. docker/<name>.Dockerfile
#   3. docker.yml workflow matrix
#
# Rules:
#   - Every cmd/<name>/ MUST have a matching docker/<name>.Dockerfile.
#     (Exception: pure-proxy/external servers may have only a Dockerfile.)
#   - The set of Dockerfiles MUST equal the docker.yml matrix exactly.
#
# Fails (exit 1) on any drift, with a diagnostic that names the missing entries.

set -euo pipefail

cd "$(dirname "$0")/.."

cmd_names=$(find cmd -mindepth 1 -maxdepth 1 -type d -printf '%f\n' | sort)
docker_names=$(find docker -maxdepth 1 -name '*.Dockerfile' -printf '%f\n' | sed 's/\.Dockerfile$//' | sort)

matrix_line=$(grep -E '^\s*server:\s*\[' .github/workflows/docker.yml || true)
if [[ -z "$matrix_line" ]]; then
  echo "ERROR: could not find 'server: [...]' matrix line in .github/workflows/docker.yml" >&2
  exit 1
fi
matrix_names=$(echo "$matrix_line" | sed -E 's/.*\[([^]]+)\].*/\1/; s/[[:space:]]//g; s/,/\n/g' | sort)

fail=0

missing_dockerfile=$(comm -23 <(echo "$cmd_names") <(echo "$docker_names") || true)
if [[ -n "$missing_dockerfile" ]]; then
  echo "ERROR: cmd/<name>/ without matching docker/<name>.Dockerfile:" >&2
  echo "$missing_dockerfile" | sed 's/^/  - /' >&2
  fail=1
fi

docker_only=$(comm -23 <(echo "$docker_names") <(echo "$matrix_names") || true)
matrix_only=$(comm -13 <(echo "$docker_names") <(echo "$matrix_names") || true)

if [[ -n "$docker_only" ]]; then
  echo "ERROR: Dockerfile present but missing from docker.yml matrix:" >&2
  echo "$docker_only" | sed 's/^/  - /' >&2
  fail=1
fi
if [[ -n "$matrix_only" ]]; then
  echo "ERROR: docker.yml matrix entry without matching Dockerfile:" >&2
  echo "$matrix_only" | sed 's/^/  - /' >&2
  fail=1
fi

if [[ "$fail" -eq 0 ]]; then
  count=$(echo "$matrix_names" | wc -l)
  echo "OK: $count MCP servers consistent across cmd/, docker/ and docker.yml matrix."
fi

exit "$fail"
