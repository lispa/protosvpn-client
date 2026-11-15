# Third-party software

## Xray-core

Windows distributions include the unmodified official Xray-core v26.11.9 executable.

- Project and corresponding source: https://github.com/XTLS/Xray-core/tree/v26.11.9
- Official release: https://github.com/XTLS/Xray-core/releases/tag/v26.11.9
- License: Mozilla Public License 2.0; full text included as `core/XRAY-LICENSE.txt`.
- Official archive and extracted executable SHA256 checksums are recorded in `core/manifest.json`.

Xray is run as a separate local subprocess. No Xray source modifications are made. The source can be obtained from the link above. `build/fetch-xray.py` downloads the pinned official release and verifies its published SHA256 digest before packaging.

The bundled core is not a TUN driver or privileged Windows service. ProtosVPN 0.2.0 uses it only for Windows per-user HTTP system proxy mode. It does not provide a kill switch or force proxy-unaware applications through the server.
