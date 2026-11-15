# Windows installer — 0.2.3-preview

Run `ProtosVPN-0.2.3-preview-windows-amd64-setup.exe` as a normal Windows user. Disconnect and close the previous client before installing/updating.

- Per-user NSIS Unicode installer; English and Russian wizard.
- Default installation: `%LOCALAPPDATA%\Programs\ProtosVPN`, no elevation.
- Both Start menu and Desktop shortcuts are mandatory and recreated on update.
- Includes the official pinned Xray-core executable in `core/xray.exe`, its MPL-2.0 license, origin/checksum manifest and `THIRD-PARTY.md` source information.
- VLESS runs as a **system proxy**, not a TUN/full-device VPN. OpenVPN connectivity is not implemented.
- WebView2 is required; the installer offers Microsoft's official download page if missing.
- The GUI starts a separate helper from the same executable; the helper owns the Xray process and WinINet proxy snapshot. GUI pipe EOF or core exit triggers restore. Settings are not changed on import or application startup.
- Previous proxy flags, server, bypass list and PAC configuration are kept in memory. If another program changes them, restoration avoids overwriting the external change.
- The helper retries failed restoration instead of immediately killing the local proxy. This is not a guarantee against killing the helper itself, power loss, or OS shutdown. If necessary, disable the manual localhost proxy in Windows Settings and restore your prior settings manually. Do not run competing proxy clients.
- Uninstall is available in Windows Apps settings. Disconnect first; uninstall removes only named binaries/docs/shortcuts, not arbitrary user files or WebView preferences.

## Build

1. Build frontend assets and Windows desktop executable (README application version must match).
2. `python3 build/fetch-xray.py` downloads pinned official cores and validates published SHA256 digests.
3. `bash build/build-installer.sh` uses Docker-contained NSIS; no host packages installed.
4. `python3 build/test_installer.py -v` checks shortcut/uninstall/version regressions.

## Verification scope

The core is exercised by real local VLESS client/server integration tests on Linux, while WinINet restore policy is tested through an injected backend and Windows ABI is cross-compiled. The NSIS package is compiled, extracted and compared against the exact intended binaries. **Real Windows WinINet calls, installer/uninstaller/shortcuts and a user's external server still require testing on Windows.**
