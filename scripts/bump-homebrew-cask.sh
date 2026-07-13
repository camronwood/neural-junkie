#!/usr/bin/env bash
# Back-compat wrapper — regenerates macOS cask and Linux formula in the tap.
exec "$(cd "$(dirname "$0")" && pwd)/bump-homebrew-tap.sh" "$@"
