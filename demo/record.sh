#!/usr/bin/env bash
set -euo pipefail
repo=$(cd "$(dirname "$0")/.." && pwd)
cd "$repo"

demo=$(mktemp -d)
export XDG_CONFIG_HOME="$demo/config"
export XDG_CACHE_HOME="$demo/cache"
export GIXT_DEMO_WORKSPACE="$demo/workspace"
mkdir -p "$GIXT_DEMO_WORKSPACE"
go build -o "$demo/gixt" ./cmd/gixt
export PATH="$demo:$PATH"
cd "$GIXT_DEMO_WORKSPACE"

exec vhs "$repo/demo/combined.tape" -o "$repo/assets/gixt-combined-demo.gif"
