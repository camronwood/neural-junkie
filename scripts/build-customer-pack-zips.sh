#!/usr/bin/env bash
# Build a customer pack zip for sideload install (run from the customer pack repo).
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
CUSTOMER_PACK="${NEURAL_JUNKIE_CUSTOMER_PACK:-}"
if [[ -z "${CUSTOMER_PACK}" ]]; then
  echo "Set NEURAL_JUNKIE_CUSTOMER_PACK to a customer pack repo path." >&2
  echo "Example: NEURAL_JUNKIE_CUSTOMER_PACK=/path/to/customer-pack-repo $0" >&2
  exit 1
fi
if [[ -f "${CUSTOMER_PACK}/scripts/build-pack-zip.sh" ]]; then
  exec "${CUSTOMER_PACK}/scripts/build-pack-zip.sh"
fi
if [[ -f "${CUSTOMER_PACK}/Makefile" ]] && grep -q '^pack-zip:' "${CUSTOMER_PACK}/Makefile" 2>/dev/null; then
  exec make -C "${CUSTOMER_PACK}" pack-zip
fi
echo "No build script found in ${CUSTOMER_PACK} (expected scripts/build-pack-zip.sh or make pack-zip)." >&2
exit 1
