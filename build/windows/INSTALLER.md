# Windows installer — 0.1.2-preview

Run `ProtosVPN-0.1.2-preview-windows-amd64-setup.exe` as your normal Windows user. Close an already running ProtosVPN before installation or updates.

- NSIS Unicode wizard, English/Russian language selection.
- Installs the existing verified 0.1.2-preview Windows x64 GUI executable. Application functionality is unchanged.
- Per-user install, default `%LOCALAPPDATA%\Programs\ProtosVPN`; `RequestExecutionLevel user`, no administrative privileges requested.
- Creates Start menu shortcuts, offers an optional Desktop shortcut.
- Registers the uninstaller under HKCU so removal is available from Windows Apps settings.
- Checks per-machine and per-user Evergreen WebView2 registration before first launch. If missing, offers the official Microsoft download page, without downloading or executing third-party dependencies. Install Evergreen Runtime there, then launch from Start.
- Uninstall deletes only named application files and shortcuts; no recursive directory deletion and no WebView preference wipe. A running/locked executable blocks removal instead of forcing process termination.
- This unsigned preview may trigger SmartScreen. It still has no VPN engine; profiles remain in RAM.

## Build

`bash build/build-installer.sh` uses a Docker-contained NSIS compiler; no host packages are installed. Build the Windows desktop executable first.

## Verification performed

- NSIS 3.09 compilation: succeeded; installer and uninstaller generated with two language tables.
- `7z t` against setup EXE: Everything is Ok.
- Extracted application bytes matched `build/bin/ProtosVPN-windows-amd64.exe` exactly; README and verification file bytes also matched. Embedded `Uninstall.exe` found.
- SHA256 saved next to installer.
- Installation, registry writes, shortcuts, upgrade/uninstall, missing-WebView2 workflow and app launch have NOT been exercised on Windows. Compiler/package checks are not a substitute for those tests.
