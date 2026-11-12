#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
test -f build/bin/ProtosVPN-windows-amd64.exe
mkdir -p release
sudo -n docker build -f build/installer.Dockerfile -t protosvpn-installer:ubuntu24 .
sudo -n docker run --rm --network none --user "$(id -u):$(id -g)" \
 -v "$PWD:/src" protosvpn-installer:ubuntu24 -V3 installer.nsi
sha256sum release/ProtosVPN-0.1.3-preview-windows-amd64-setup.exe > release/ProtosVPN-0.1.3-preview-setup.sha256
