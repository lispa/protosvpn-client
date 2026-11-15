#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
test -f build/bin/ProtosVPN-windows-amd64.exe
python3 -c 'import json,hashlib,pathlib; r=pathlib.Path("build/cores"); m=json.loads((r/"manifest.json").read_text()); expected=next(x["binarySHA256"] for x in m if x["platform"]=="windows-amd64"); assert hashlib.sha256((r/"windows-amd64/xray.exe").read_bytes()).hexdigest()==expected,"Xray core checksum mismatch"'
mkdir -p release
sudo -n docker build -f build/installer.Dockerfile -t protosvpn-installer:ubuntu24 .
sudo -n docker run --rm --network none --user "$(id -u):$(id -g)" \
 -v "$PWD:/src" protosvpn-installer:ubuntu24 -V3 installer.nsi
sha256sum release/ProtosVPN-0.2.1-preview-windows-amd64-setup.exe > release/ProtosVPN-0.2.1-preview-setup.sha256
