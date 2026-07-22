#!/usr/bin/env bash
# Dev-link + sync the local software-development pack (with real sd-mcp-server) into
# ~/.neural-junkie/packs/software-development. Requires the sibling pack repo.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
PACK_SRC="${NJ_SD_PACK_DIR:-$(cd "${ROOT}/../neural-junkie-pack-software-development" 2>/dev/null && pwd || true)}"
if [[ -z "${PACK_SRC}" || ! -f "${PACK_SRC}/pack.yaml" ]]; then
  echo "software-development pack repo not found. Set NJ_SD_PACK_DIR or clone beside neural-junkie." >&2
  exit 1
fi
BIN="${PACK_SRC}/assets/mcp/bin/sd-mcp-server"
if [[ ! -f "${BIN}" ]] || file "${BIN}" | grep -q 'shell script\|ASCII text'; then
  echo "Building sd-mcp-server in ${PACK_SRC}..."
  (cd "${PACK_SRC}" && make build-mcp)
fi
if ! file "${BIN}" | grep -Eq 'Mach-O|ELF|executable'; then
  echo "Expected a real binary at ${BIN}, got: $(file "${BIN}")" >&2
  exit 1
fi

cd "${ROOT}"
go run ./scripts/cmd/sync-sd-pack --pack-dir "${PACK_SRC}"
echo "Synced software-development from ${PACK_SRC}"
file "${HOME}/.neural-junkie/packs/software-development/assets/mcp/bin/sd-mcp-server"
