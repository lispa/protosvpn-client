#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
: "${GO_SDK_DIR:=$HOME/.local/toolchains/go}"
: "${WAILS_CLI:=$HOME/go/bin/wails}"
mkdir -p build/qa build/bin "$HOME/.cache/go-build"
sudo -n docker run --rm --network none --memory=1g --memory-swap=2g --cpus=2 --user "$(id -u):$(id -g)" \
 -e HOME=/tmp -e GOPATH=/go -e GOMODCACHE=/go/pkg/mod -e GOFLAGS=-p=1 \
 -v "$PWD:/src" -v "$GO_SDK_DIR:/opt/go:ro" -v "$WAILS_CLI:/opt/wails:ro" \
 -v "$HOME/go:/go:ro" -v "$HOME/.cache/go-build:/tmp/go-build" \
 protosvpn-builder:ubuntu24 sh -c '
 /opt/wails build -s -m -skipbindings -tags desktop,webkit2_41 -trimpath -o ProtosVPN-linux-amd64
 go test -p 1 -tags desktop,webkit2_41 .
 go build -p 1 -trimpath -tags desktop,production,webkit2_41,native_smoke -o build/qa/ProtosVPN-native-smoke .
 '
