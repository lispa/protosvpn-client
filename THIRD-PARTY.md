# Third-party software

## Xray-core

Windows installer and portable distributions include two unmodified official Xray-core executables:

| Variant | Pinned version | Installed executable | Included license |
| --- | --- | --- | --- |
| Modern (default) | v26.11.9 | `core/xray.exe` | `core/XRAY-LICENSE.txt` |
| Legacy (explicit compatibility opt-in only) | v25.12.8 | `core/xray-legacy.exe` | `core/XRAY-LEGACY-LICENSE.txt` |

Both versions are licensed under the Mozilla Public License 2.0. The full license text for each pinned source version is included separately. This notice is included as `THIRD-PARTY.md` in both Windows distribution formats.

### Official releases and corresponding source

- Modern source: https://github.com/XTLS/Xray-core/tree/v26.11.9
- Modern release: https://github.com/XTLS/Xray-core/releases/tag/v26.11.9
- Legacy source: https://github.com/XTLS/Xray-core/tree/v25.12.8
- Legacy release: https://github.com/XTLS/Xray-core/releases/tag/v25.12.8

Xray runs as a separate local subprocess. No Xray source modifications are made; corresponding source is available at the links above. `build/fetch-xray.py` downloads both pinned official releases for Windows and Linux amd64 and verifies each ZIP against its official release `.dgst` SHA256 before extracting it. `core/manifest.json` records each variant, platform, version, official source archive URL, archive SHA256 and extracted executable SHA256. Modern records remain first for compatibility with tools that select the first matching platform. Windows portable and installer build scripts check the modern and legacy executable hashes separately before packaging.

Linux binaries in `build/cores/linux-amd64/` are development/verification tools only. The Linux management-only application package does not bundle either core or enable a Linux tunnel.

### Legacy compatibility and security limitations

Compatibility mode is off by default and requires explicit selection and a UI warning. The legacy pin is not a security-equivalent replacement for the modern core: it predates later fixes and security policy changes and should not be assumed to receive current protections. It is included only for interoperability with older plaintext VLESS configurations, not as an automatic fallback when modern validation fails.

With the same synthetic public-server plaintext VLESS configuration (`example.com`, no TLS or VLESS encryption), Linux `xray-legacy run -test` from v25.12.8 reports `Configuration OK`, whereas v26.11.9 rejects it because public plaintext VLESS is prohibited. This is an offline configuration-validation result, not evidence of a successful connection or a security audit. No server traffic is required by that test.

Plaintext VLESS provides no transport encryption or authenticated server identity. Intermediaries can observe or modify traffic not independently protected by the application, and the VLESS credential lacks transport confidentiality. Prefer a correctly configured TLS/REALITY or otherwise encrypted modern profile. Enabling legacy compatibility does not add encryption, make a plaintext server safe, or provide a kill switch.

Neither bundled core is a TUN driver or privileged Windows service. ProtosVPN uses the selected core for Windows per-user HTTP system proxy mode. It does not force proxy-unaware applications through the server; some applications may bypass the proxy.
